package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Event represents a message from SQS
type Event struct {
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp string                 `json:"timestamp"`
}

func main() {
	sqsQueue := os.Getenv("SQS_QUEUE_URL")
	if sqsQueue == "" {
		log.Fatal("SQS_QUEUE_URL is required")
	}

	// Internal service URLs for event-driven calls
	services := map[string]string{
		"inventory":    getEnv("INVENTORY_SERVICE_URL", "http://inventory-service:8082"),
		"payment":      getEnv("PAYMENT_SERVICE_URL", "http://payment-service:8083"),
		"notification": getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8084"),
		"shipping":     getEnv("SHIPPING_SERVICE_URL", "http://shipping-service:8085"),
		"order":        getEnv("ORDER_SERVICE_URL", "http://order-service:8081"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Health check endpoint
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "worker"})
		})
		port := getEnv("HEALTH_PORT", "8090")
		log.Printf("Worker health check on :%s", port)
		http.ListenAndServe(":"+port, mux)
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down worker...")
		cancel()
	}()

	log.Println("Worker started, polling SQS for events...")
	pollAndProcess(ctx, sqsQueue, services)
}

func pollAndProcess(ctx context.Context, queueURL string, services map[string]string) {
	client := &http.Client{Timeout: 10 * time.Second}

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker stopped")
			return
		default:
			messages := receiveSQSMessages(ctx, queueURL)

			for _, raw := range messages {
				var event Event
				if err := json.Unmarshal([]byte(raw), &event); err != nil {
					log.Printf("Failed to parse event: %v", err)
					continue
				}

				log.Printf("Processing event: %s", event.Type)

				if err := handleEvent(client, services, event); err != nil {
					log.Printf("Failed to handle event %s: %v", event.Type, err)
					// In production: don't delete from SQS, let it retry or go to DLQ
					continue
				}

				log.Printf("Successfully processed: %s", event.Type)
				// Delete message from SQS after successful processing
			}

			if len(messages) == 0 {
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func handleEvent(client *http.Client, services map[string]string, event Event) error {
	switch event.Type {

	case "order.created":
		// 1. Reserve inventory
		log.Printf("  -> Reserving inventory for order")
		// POST to inventory-service/reserve with order items
		// If reservation fails, update order status to "cancelled"

		// 2. Process payment
		log.Printf("  -> Processing payment")
		// POST to payment-service/charge
		// If payment fails, release inventory reservation

		// 3. Send confirmation notification
		log.Printf("  -> Sending order confirmation")
		// POST to notification-service/send with order_confirmed template

		// 4. Update order to confirmed
		log.Printf("  -> Confirming order")
		// PUT to order-service/status with new_status: "confirmed"

	case "order.status_changed":
		newStatus, _ := event.Payload["new_status"].(string)

		switch newStatus {
		case "processing":
			// Create shipment
			log.Printf("  -> Creating shipment for order")
			// POST to shipping-service/shipments

		case "shipped":
			// Notify customer
			log.Printf("  -> Sending shipping notification")
			// POST to notification-service/send with order_shipped template

		case "delivered":
			log.Printf("  -> Sending delivery notification")
			// POST to notification-service/send with order_delivered template

		case "cancelled":
			// Release inventory
			log.Printf("  -> Releasing inventory reservation")
			// POST to inventory-service/release

			// Process refund if payment was made
			log.Printf("  -> Processing refund")
			// POST to payment-service/refund
		}

	case "payment.completed":
		log.Printf("  -> Payment successful, confirming order")
		// Update order status to confirmed

	case "payment.failed":
		log.Printf("  -> Payment failed, cancelling order")
		// Release inventory reservation
		// Update order status to cancelled
		// Send payment failed notification

	case "shipment.created":
		log.Printf("  -> Shipment created, updating order to processing")
		// Update order status

	case "shipment.delivered":
		log.Printf("  -> Shipment delivered, updating order")
		// Update order status to delivered
		// Send delivery notification

	default:
		log.Printf("  -> Unknown event type: %s (skipping)", event.Type)
	}

	_ = client
	_ = services
	return nil
}

func receiveSQSMessages(ctx context.Context, queueURL string) []string {
	// Students implement with AWS SDK SQS ReceiveMessage
	// Use long polling: WaitTimeSeconds = 20
	// MaxNumberOfMessages = 10
	// Honour ctx during the long-poll so SIGTERM unblocks cleanly
	_ = ctx
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
