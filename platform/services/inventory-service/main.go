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

type Product struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	SKU       string  `json:"sku"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	Reserved  int     `json:"reserved"`
	Available int     `json:"available"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type Reservation struct {
	ID        int    `json:"id"`
	OrderID   int    `json:"order_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
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
	mux.HandleFunc("/products", handleProducts)
	mux.HandleFunc("/products/", handleProduct)
	mux.HandleFunc("/reserve", handleReserve)
	mux.HandleFunc("/release", handleRelease)
	mux.HandleFunc("/low-stock", handleLowStock)

	port := getEnv("PORT", "8082")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Inventory service listening on :%s", port)
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
		`CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			sku VARCHAR(100) UNIQUE NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			stock INTEGER NOT NULL DEFAULT 0,
			reserved INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku)`,
		`CREATE TABLE IF NOT EXISTS reservations (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL,
			product_id VARCHAR(50) NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reservations_order ON reservations(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reservations_status ON reservations(status)`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}
	log.Println("Inventory service migrations complete")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if err := db.Ping(); err != nil {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status, "service": "inventory-service"})
}

func handleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(
			"SELECT id, name, sku, price, stock, reserved, created_at, updated_at FROM products ORDER BY name LIMIT 100",
		)
		if err != nil {
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		products := []Product{}
		for rows.Next() {
			var p Product
			rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Price, &p.Stock, &p.Reserved, &p.CreatedAt, &p.UpdatedAt)
			p.Available = p.Stock - p.Reserved
			products = append(products, p)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)

	case http.MethodPost:
		var p struct {
			ID    string  `json:"id"`
			Name  string  `json:"name"`
			SKU   string  `json:"sku"`
			Price float64 `json:"price"`
			Stock int     `json:"stock"`
		}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		_, err := db.Exec(
			"INSERT INTO products (id, name, sku, price, stock) VALUES ($1, $2, $3, $4, $5)",
			p.ID, p.Name, p.SKU, p.Price, p.Stock,
		)
		if err != nil {
			httpError(w, "failed to create product: "+err.Error(), http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": p.ID, "status": "created"})

	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleProduct(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/products/")
	if id == "" {
		httpError(w, "product id required", http.StatusBadRequest)
		return
	}

	var p Product
	err := db.QueryRow(
		"SELECT id, name, sku, price, stock, reserved, created_at, updated_at FROM products WHERE id = $1",
		id,
	).Scan(&p.ID, &p.Name, &p.SKU, &p.Price, &p.Stock, &p.Reserved, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		httpError(w, "product not found", http.StatusNotFound)
		return
	}
	p.Available = p.Stock - p.Reserved

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func handleReserve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID int `json:"order_id"`
		Items   []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		httpError(w, "transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	expiresAt := time.Now().Add(15 * time.Minute)

	for _, item := range req.Items {
		// Check available stock with row lock
		var stock, reserved int
		err := tx.QueryRow(
			"SELECT stock, reserved FROM products WHERE id = $1 FOR UPDATE",
			item.ProductID,
		).Scan(&stock, &reserved)
		if err != nil {
			httpError(w, fmt.Sprintf("product %s not found", item.ProductID), http.StatusNotFound)
			return
		}

		available := stock - reserved
		if available < item.Quantity {
			httpError(w, fmt.Sprintf("insufficient stock for %s: available %d, requested %d",
				item.ProductID, available, item.Quantity), http.StatusConflict)
			return
		}

		// Reserve stock
		if _, err := tx.Exec("UPDATE products SET reserved = reserved + $1, updated_at = NOW() WHERE id = $2",
			item.Quantity, item.ProductID); err != nil {
			httpError(w, "reservation update failed", http.StatusInternalServerError)
			return
		}

		if _, err := tx.Exec(
			"INSERT INTO reservations (order_id, product_id, quantity, status, expires_at) VALUES ($1, $2, $3, 'active', $4)",
			req.OrderID, item.ProductID, item.Quantity, expiresAt,
		); err != nil {
			httpError(w, "reservation insert failed", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		httpError(w, "reservation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":   req.OrderID,
		"status":     "reserved",
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func handleRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID int `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		httpError(w, "transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	rows, err := tx.Query(
		"SELECT product_id, quantity FROM reservations WHERE order_id = $1 AND status = 'active'",
		req.OrderID,
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	type item struct {
		productID string
		quantity  int
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.productID, &it.quantity); err != nil {
			rows.Close()
			httpError(w, "scan failed", http.StatusInternalServerError)
			return
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		httpError(w, "iteration failed", http.StatusInternalServerError)
		return
	}

	for _, it := range items {
		if _, err := tx.Exec("UPDATE products SET reserved = reserved - $1, updated_at = NOW() WHERE id = $2",
			it.quantity, it.productID); err != nil {
			httpError(w, "release update failed", http.StatusInternalServerError)
			return
		}
	}

	if _, err := tx.Exec("UPDATE reservations SET status = 'released' WHERE order_id = $1 AND status = 'active'", req.OrderID); err != nil {
		httpError(w, "release update failed", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		httpError(w, "commit failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id": req.OrderID,
		"released": len(items),
	})
}

func handleLowStock(w http.ResponseWriter, r *http.Request) {
	threshold := 10
	rows, err := db.Query(
		"SELECT id, name, sku, stock, reserved FROM products WHERE (stock - reserved) < $1 ORDER BY (stock - reserved) ASC",
		threshold,
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type LowStockItem struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		SKU       string `json:"sku"`
		Stock     int    `json:"stock"`
		Reserved  int    `json:"reserved"`
		Available int    `json:"available"`
	}

	items := []LowStockItem{}
	for rows.Next() {
		var i LowStockItem
		rows.Scan(&i.ID, &i.Name, &i.SKU, &i.Stock, &i.Reserved)
		i.Available = i.Stock - i.Reserved
		items = append(items, i)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
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
