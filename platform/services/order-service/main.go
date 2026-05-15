package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

type Order struct {
	ID         int             `json:"id"`
	CustomerID string          `json:"customer_id"`
	Status     string          `json:"status"`
	Items      json.RawMessage `json:"items"`
	Total      float64         `json:"total"`
	Currency   string          `json:"currency"`
	Notes      string          `json:"notes,omitempty"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
}

type CreateOrderRequest struct {
	Items    []OrderItem `json:"items"`
	Currency string      `json:"currency"`
	Notes    string      `json:"notes,omitempty"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// Valid state transitions
var validTransitions = map[string][]string{
	"pending":    {"confirmed", "cancelled"},
	"confirmed":  {"processing", "cancelled"},
	"processing": {"shipped", "cancelled"},
	"shipped":    {"delivered"},
	"delivered":  {},
	"cancelled":  {},
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	waitForDB()
	migrate()

	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/", handleOrders)
	mux.HandleFunc("/status", handleUpdateStatus)

	port := getEnv("PORT", "8081")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Order service listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}

func migrate() {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			customer_id VARCHAR(255) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			items JSONB NOT NULL,
			total DECIMAL(12,2) NOT NULL DEFAULT 0,
			currency VARCHAR(3) NOT NULL DEFAULT 'GBP',
			notes TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_customer ON orders(customer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status)`,
		`CREATE TABLE IF NOT EXISTS order_events (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL REFERENCES orders(id),
			event_type VARCHAR(50) NOT NULL,
			old_status VARCHAR(20),
			new_status VARCHAR(20),
			metadata JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_order_events_order ON order_events(order_id)`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}
	log.Println("Order service migrations complete")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if err := db.Ping(); err != nil {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status, "service": "order-service"})
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// GET / - list orders (filtered by customer from header)
		// GET /{id} - get specific order
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" && path != "/" {
			getOrder(w, r, path)
		} else {
			listOrders(w, r)
		}
	case http.MethodPost:
		createOrder(w, r)
	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func createOrder(w http.ResponseWriter, r *http.Request) {
	customerID := r.Header.Get("X-User-Email")
	if customerID == "" {
		httpError(w, "missing customer identity", http.StatusBadRequest)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		httpError(w, "order must have at least one item", http.StatusBadRequest)
		return
	}

	// Calculate total
	var total float64
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}

	currency := req.Currency
	if currency == "" {
		currency = "GBP"
	}

	itemsJSON, _ := json.Marshal(req.Items)

	var orderID int
	err := db.QueryRow(
		`INSERT INTO orders (customer_id, status, items, total, currency, notes)
		 VALUES ($1, 'pending', $2, $3, $4, $5) RETURNING id`,
		customerID, itemsJSON, total, currency, req.Notes,
	).Scan(&orderID)

	if err != nil {
		log.Printf("Create order error: %v", err)
		httpError(w, "failed to create order", http.StatusInternalServerError)
		return
	}

	// Record event
	db.Exec(
		`INSERT INTO order_events (order_id, event_type, new_status)
		 VALUES ($1, 'order_created', 'pending')`,
		orderID,
	)

	// Publish to SQS for downstream services (inventory reservation, etc.)
	publishEvent("order.created", map[string]interface{}{
		"order_id":    orderID,
		"customer_id": customerID,
		"items":       req.Items,
		"total":       total,
		"currency":    currency,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     orderID,
		"status": "pending",
		"total":  total,
	})
}

func listOrders(w http.ResponseWriter, r *http.Request) {
	customerID := r.Header.Get("X-User-Email")
	status := r.URL.Query().Get("status")

	query := "SELECT id, customer_id, status, items, total, currency, notes, created_at, updated_at FROM orders WHERE 1=1"
	args := []interface{}{}
	argN := 1

	if customerID != "" {
		query += fmt.Sprintf(" AND customer_id = $%d", argN)
		args = append(args, customerID)
		argN++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, status)
		argN++
	}
	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.Status, &o.Items, &o.Total, &o.Currency, &o.Notes, &o.CreatedAt, &o.UpdatedAt); err != nil {
			httpError(w, "scan failed", http.StatusInternalServerError)
			return
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		httpError(w, "iteration failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func getOrder(w http.ResponseWriter, r *http.Request, id string) {
	var o Order
	err := db.QueryRow(
		"SELECT id, customer_id, status, items, total, currency, notes, created_at, updated_at FROM orders WHERE id = $1",
		id,
	).Scan(&o.ID, &o.CustomerID, &o.Status, &o.Items, &o.Total, &o.Currency, &o.Notes, &o.CreatedAt, &o.UpdatedAt)

	if err != nil {
		httpError(w, "order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID   int    `json:"order_id"`
		NewStatus string `json:"new_status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM orders WHERE id = $1", req.OrderID).Scan(&currentStatus)
	if err != nil {
		httpError(w, "order not found", http.StatusNotFound)
		return
	}

	// Validate transition
	allowed, ok := validTransitions[currentStatus]
	if !ok {
		httpError(w, "invalid current status", http.StatusBadRequest)
		return
	}

	valid := false
	for _, s := range allowed {
		if s == req.NewStatus {
			valid = true
			break
		}
	}
	if !valid {
		httpError(w, fmt.Sprintf("cannot transition from %s to %s", currentStatus, req.NewStatus), http.StatusConflict)
		return
	}

	_, err = db.Exec(
		"UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2",
		req.NewStatus, req.OrderID,
	)
	if err != nil {
		httpError(w, "failed to update order", http.StatusInternalServerError)
		return
	}

	// Record event
	db.Exec(
		`INSERT INTO order_events (order_id, event_type, old_status, new_status)
		 VALUES ($1, 'status_changed', $2, $3)`,
		req.OrderID, currentStatus, req.NewStatus,
	)

	// Publish event
	publishEvent("order.status_changed", map[string]interface{}{
		"order_id":   req.OrderID,
		"old_status": currentStatus,
		"new_status": req.NewStatus,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":   req.OrderID,
		"old_status": currentStatus,
		"new_status": req.NewStatus,
	})
}

func publishEvent(eventType string, payload map[string]interface{}) {
	sqsQueue := os.Getenv("SQS_QUEUE_URL")
	if sqsQueue == "" {
		log.Printf("Event (no SQS): %s %v", eventType, payload)
		return
	}

	event := map[string]interface{}{
		"type":      eventType,
		"payload":   payload,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(event)
	log.Printf("Event -> SQS: %s", string(data))
	// Students implement actual SQS SendMessage here
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func waitForDB() {
	for i := 0; i < 120; i++ {
		if err := db.Ping(); err == nil {
			return
		}
		log.Printf("Waiting for database... (%d/120)", i+1)
		time.Sleep(time.Second)
	}
	log.Fatal("Database not ready after 120s")
}
