package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

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
	mux.HandleFunc("/charge", handleCharge)
	mux.HandleFunc("/refund", handleRefund)
	mux.HandleFunc("/ledger", handleLedger)
	mux.HandleFunc("/balance/", handleBalance)

	port := getEnv("PORT", "8083")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Payment service listening on :%s", port)
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
		`CREATE TABLE IF NOT EXISTS payments (
			id VARCHAR(50) PRIMARY KEY,
			order_id INTEGER NOT NULL,
			customer_id VARCHAR(255) NOT NULL,
			amount DECIMAL(12,2) NOT NULL,
			currency VARCHAR(3) NOT NULL DEFAULT 'GBP',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			method VARCHAR(50),
			reference VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_order ON payments(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_customer ON payments(customer_id)`,
		`CREATE TABLE IF NOT EXISTS ledger_entries (
			id SERIAL PRIMARY KEY,
			payment_id VARCHAR(50) NOT NULL REFERENCES payments(id),
			entry_type VARCHAR(20) NOT NULL,
			debit DECIMAL(12,2) NOT NULL DEFAULT 0,
			credit DECIMAL(12,2) NOT NULL DEFAULT 0,
			currency VARCHAR(3) NOT NULL DEFAULT 'GBP',
			description TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ledger_payment ON ledger_entries(payment_id)`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}
	log.Println("Payment service migrations complete")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if err := db.Ping(); err != nil {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status, "service": "payment-service"})
}

func handleCharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID    int     `json:"order_id"`
		CustomerID string  `json:"customer_id"`
		Amount     float64 `json:"amount"`
		Currency   string  `json:"currency"`
		Method     string  `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		httpError(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	paymentID := generatePaymentID()
	currency := req.Currency
	if currency == "" {
		currency = "GBP"
	}

	// Simulate payment processing (90% success rate)
	status := "completed"
	if rand.Intn(10) == 0 {
		status = "failed"
	}

	tx, err := db.Begin()
	if err != nil {
		httpError(w, "transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO payments (id, order_id, customer_id, amount, currency, status, method)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		paymentID, req.OrderID, req.CustomerID, req.Amount, currency, status, req.Method,
	)
	if err != nil {
		httpError(w, "failed to process payment", http.StatusInternalServerError)
		return
	}

	if status == "completed" {
		// Double-entry: debit customer, credit revenue
		tx.Exec(
			`INSERT INTO ledger_entries (payment_id, entry_type, debit, currency, description)
			 VALUES ($1, 'charge', $2, $3, $4)`,
			paymentID, req.Amount, currency, fmt.Sprintf("Order #%d payment", req.OrderID),
		)
		tx.Exec(
			`INSERT INTO ledger_entries (payment_id, entry_type, credit, currency, description)
			 VALUES ($1, 'revenue', $2, $3, $4)`,
			paymentID, req.Amount, currency, fmt.Sprintf("Order #%d revenue", req.OrderID),
		)
	}

	tx.Commit()

	// Publish event
	publishEvent("payment."+status, map[string]interface{}{
		"payment_id":  paymentID,
		"order_id":    req.OrderID,
		"customer_id": req.CustomerID,
		"amount":      req.Amount,
		"currency":    currency,
		"status":      status,
	})

	code := http.StatusCreated
	if status == "failed" {
		code = http.StatusPaymentRequired
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payment_id": paymentID,
		"order_id":   req.OrderID,
		"amount":     req.Amount,
		"currency":   currency,
		"status":     status,
	})
}

func handleRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PaymentID string  `json:"payment_id"`
		Amount    float64 `json:"amount"`
		Reason    string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get original payment
	var originalAmount float64
	var orderID int
	var paymentStatus, customerID, currency string
	err := db.QueryRow(
		"SELECT amount, order_id, status, customer_id, currency FROM payments WHERE id = $1",
		req.PaymentID,
	).Scan(&originalAmount, &orderID, &paymentStatus, &customerID, &currency)
	if err != nil {
		httpError(w, "payment not found", http.StatusNotFound)
		return
	}

	if paymentStatus != "completed" {
		httpError(w, "can only refund completed payments", http.StatusConflict)
		return
	}

	refundAmount := req.Amount
	if refundAmount <= 0 || refundAmount > originalAmount {
		refundAmount = originalAmount
	}

	refundID := generatePaymentID()

	tx, err := db.Begin()
	if err != nil {
		httpError(w, "transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO payments (id, order_id, customer_id, amount, currency, status, method, reference)
		 VALUES ($1, $2, $3, $4, $5, 'completed', 'refund', $6)`,
		refundID, orderID, customerID, refundAmount, currency, req.PaymentID,
	)
	if err != nil {
		httpError(w, "refund insert failed", http.StatusInternalServerError)
		return
	}

	// Reverse ledger entries
	tx.Exec(
		`INSERT INTO ledger_entries (payment_id, entry_type, credit, currency, description)
		 VALUES ($1, 'refund', $2, $3, $4)`,
		refundID, refundAmount, currency, fmt.Sprintf("Refund for payment %s: %s", req.PaymentID, req.Reason),
	)

	if refundAmount >= originalAmount {
		tx.Exec("UPDATE payments SET status = 'refunded', updated_at = NOW() WHERE id = $1", req.PaymentID)
	} else {
		tx.Exec("UPDATE payments SET status = 'partially_refunded', updated_at = NOW() WHERE id = $1", req.PaymentID)
	}

	if err := tx.Commit(); err != nil {
		httpError(w, "refund failed", http.StatusInternalServerError)
		return
	}

	publishEvent("payment.refunded", map[string]interface{}{
		"refund_id":  refundID,
		"payment_id": req.PaymentID,
		"order_id":   orderID,
		"amount":     refundAmount,
		"reason":     req.Reason,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"refund_id":  refundID,
		"payment_id": req.PaymentID,
		"amount":     refundAmount,
		"status":     "completed",
	})
}

func handleLedger(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(
		`SELECT le.id, le.payment_id, le.entry_type, le.debit, le.credit, le.currency, le.description, le.created_at
		 FROM ledger_entries le ORDER BY le.created_at DESC LIMIT 100`,
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Entry struct {
		ID          int     `json:"id"`
		PaymentID   string  `json:"payment_id"`
		EntryType   string  `json:"entry_type"`
		Debit       float64 `json:"debit"`
		Credit      float64 `json:"credit"`
		Currency    string  `json:"currency"`
		Description string  `json:"description"`
		CreatedAt   string  `json:"created_at"`
	}

	entries := []Entry{}
	for rows.Next() {
		var e Entry
		rows.Scan(&e.ID, &e.PaymentID, &e.EntryType, &e.Debit, &e.Credit, &e.Currency, &e.Description, &e.CreatedAt)
		entries = append(entries, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func handleBalance(w http.ResponseWriter, r *http.Request) {
	customerID := strings.TrimPrefix(r.URL.Path, "/balance/")
	if customerID == "" {
		httpError(w, "customer id required", http.StatusBadRequest)
		return
	}

	var totalCharged, totalRefunded float64
	db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE customer_id = $1 AND status IN ('completed', 'partially_refunded', 'refunded') AND (method IS NULL OR method != 'refund')",
		customerID,
	).Scan(&totalCharged)

	db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE customer_id = $1 AND method = 'refund'",
		customerID,
	).Scan(&totalRefunded)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"customer_id":    customerID,
		"total_charged":  totalCharged,
		"total_refunded": totalRefunded,
		"net":            totalCharged - totalRefunded,
	})
}

func generatePaymentID() string {
	return fmt.Sprintf("pay_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
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
