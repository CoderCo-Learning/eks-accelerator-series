package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	waitForDB()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Health check
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			status := "ok"
			if err := db.Ping(); err != nil {
				status = "unhealthy"
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": status, "service": "scheduler"})
		})
		port := getEnv("HEALTH_PORT", "8091")
		log.Printf("Scheduler health check on :%s", port)
		http.ListenAndServe(":"+port, mux)
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down scheduler...")
		cancel()
	}()

	log.Println("Scheduler started")

	// Run scheduled jobs
	go runEvery(ctx, 1*time.Minute, "expire_reservations", expireReservations)
	go runEvery(ctx, 5*time.Minute, "abandoned_carts", detectAbandonedOrders)
	go runEvery(ctx, 15*time.Minute, "retry_failed_payments", retryFailedPayments)
	go runEvery(ctx, 1*time.Hour, "daily_digest", generateDigest)
	go runEvery(ctx, 30*time.Minute, "cleanup_old_events", cleanupOldEvents)

	<-ctx.Done()
	log.Println("Scheduler stopped")
}

func runEvery(ctx context.Context, interval time.Duration, name string, fn func()) {
	log.Printf("Scheduled job '%s' every %s", name, interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	log.Printf("Running job: %s", name)
	fn()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Printf("Running job: %s", name)
			start := time.Now()
			fn()
			log.Printf("Job '%s' completed in %s", name, time.Since(start))
		}
	}
}

// expireReservations releases inventory reservations that have expired
func expireReservations() {
	// Find expired reservations and release them
	rows, err := db.Query(
		`SELECT r.order_id, r.product_id, r.quantity
		 FROM reservations r
		 WHERE r.status = 'active' AND r.expires_at < NOW()`,
	)
	if err != nil {
		log.Printf("expire_reservations query error: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var orderID int
		var productID string
		var quantity int
		rows.Scan(&orderID, &productID, &quantity)

		tx, err := db.Begin()
		if err != nil {
			log.Printf("expire_reservations begin error: %v", err)
			continue
		}
		tx.Exec("UPDATE products SET reserved = GREATEST(reserved - $1, 0), updated_at = NOW() WHERE id = $2",
			quantity, productID)
		tx.Exec("UPDATE reservations SET status = 'expired' WHERE order_id = $1 AND product_id = $2 AND status = 'active'",
			orderID, productID)
		tx.Commit()
		count++
	}

	if count > 0 {
		log.Printf("Expired %d reservations", count)
	}
}

// detectAbandonedOrders finds orders stuck in pending for too long
func detectAbandonedOrders() {
	cutoff := time.Now().Add(-30 * time.Minute)

	rows, err := db.Query(
		`SELECT id, customer_id FROM orders
		 WHERE status = 'pending' AND created_at < $1`,
		cutoff,
	)
	if err != nil {
		log.Printf("abandoned_carts query error: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var orderID int
		var customerID string
		rows.Scan(&orderID, &customerID)

		// Cancel the order
		db.Exec("UPDATE orders SET status = 'cancelled', updated_at = NOW() WHERE id = $1", orderID)

		// Publish cancellation event (would trigger inventory release via worker)
		log.Printf("Cancelled abandoned order #%d (customer: %s)", orderID, customerID)
		count++
	}

	if count > 0 {
		log.Printf("Cancelled %d abandoned orders", count)
	}
}

// retryFailedPayments attempts to re-process failed payments
func retryFailedPayments() {
	cutoff := time.Now().Add(-1 * time.Hour)

	var count int
	db.QueryRow(
		`SELECT COUNT(*) FROM payments
		 WHERE status = 'failed' AND created_at > $1`,
		cutoff,
	).Scan(&count)

	if count > 0 {
		log.Printf("Found %d failed payments eligible for retry", count)
		// In production: re-queue payment attempts via SQS
		// POST to payment-service/charge for each
	}
}

// generateDigest creates summary stats
func generateDigest() {
	var totalOrders, pendingOrders, completedOrders int
	var totalRevenue float64

	today := time.Now().UTC().Truncate(24 * time.Hour)

	db.QueryRow("SELECT COUNT(*) FROM orders WHERE created_at >= $1", today).Scan(&totalOrders)
	db.QueryRow("SELECT COUNT(*) FROM orders WHERE status = 'pending'").Scan(&pendingOrders)
	db.QueryRow("SELECT COUNT(*) FROM orders WHERE status = 'delivered' AND updated_at >= $1", today).Scan(&completedOrders)
	db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status = 'completed' AND created_at >= $1", today).Scan(&totalRevenue)

	log.Printf("Daily digest - Orders today: %d, Pending: %d, Completed: %d, Revenue: %.2f",
		totalOrders, pendingOrders, completedOrders, totalRevenue)

	// In production: send digest via notification-service
}

// cleanupOldEvents removes old tracking/audit data
func cleanupOldEvents() {
	cutoff := time.Now().AddDate(0, 0, -90) // 90 days retention

	result, err := db.Exec("DELETE FROM order_events WHERE created_at < $1", cutoff)
	if err != nil {
		log.Printf("cleanup error: %v", err)
		return
	}

	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Printf("Cleaned up %d old order events", rows)
	}
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
