package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"kafka-pipeline/internal/kafka"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	eventsProducedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_produced_total",
			Help: "Total number of events produced to Kafka",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(eventsProducedTotal)
}

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Get configuration from environment
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "events")
	servicePort := getEnv("SERVICE_PORT", "8080")

	// Initialize Kafka producer
	brokers := strings.Split(kafkaBrokers, ",")
	producer := kafka.NewProducer(brokers, kafkaTopic, logger)
	defer producer.Close()

	// Create HTTP server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Producer endpoint
	mux.HandleFunc("/produce", func(w http.ResponseWriter, r *http.Request) {
		handleProduce(w, r, producer, logger)
	})

	// Start server
	server := &http.Server{
		Addr:         ":" + servicePort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("Starting producer service", zap.String("port", servicePort))
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func handleProduce(w http.ResponseWriter, r *http.Request, producer *kafka.Producer, logger *zap.Logger) {
	// Increment request counter
	httpRequestsTotal.WithLabelValues(r.Method, "/produce", "200").Inc()

	if r.Method != http.MethodPost {
		httpRequestsTotal.WithLabelValues(r.Method, "/produce", "405").Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var event map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/produce", "400").Inc()
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate event structure
	if err := validateEvent(event); err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/produce", "400").Inc()
		http.Error(w, fmt.Sprintf("Invalid event: %v", err), http.StatusBadRequest)
		return
	}

	// Publish to Kafka
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := producer.PublishEvent(ctx, event); err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/produce", "500").Inc()
		logger.Error("Failed to publish event", zap.Error(err))
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

	// Increment events produced counter
	eventType := event["type"].(string)
	eventsProducedTotal.WithLabelValues(eventType).Inc()

	// Log successful production
	eventID := event["eventId"].(string)
	logger.Info("Event produced successfully",
		zap.String("eventId", eventID),
		zap.String("type", eventType),
	)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Event produced successfully"))
}

func validateEvent(event map[string]interface{}) error {
	// Check required fields
	if _, ok := event["eventId"]; !ok {
		return fmt.Errorf("eventId is required")
	}

	if _, ok := event["type"]; !ok {
		return fmt.Errorf("type is required")
	}

	if _, ok := event["timestamp"]; !ok {
		return fmt.Errorf("timestamp is required")
	}

	if _, ok := event["data"]; !ok {
		return fmt.Errorf("data is required")
	}

	// Validate event type
	eventType, ok := event["type"].(string)
	if !ok {
		return fmt.Errorf("type must be a string")
	}

	validTypes := map[string]bool{
		"UserCreated":       true,
		"OrderPlaced":       true,
		"PaymentSettled":    true,
		"InventoryAdjusted": true,
	}

	if !validTypes[eventType] {
		return fmt.Errorf("invalid event type: %s", eventType)
	}

	// Validate data field
	data, ok := event["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("data must be an object")
	}

	// Validate data fields based on event type
	switch eventType {
	case "UserCreated":
		if _, ok := data["userId"]; !ok {
			return fmt.Errorf("userId is required for UserCreated event")
		}
		if _, ok := data["name"]; !ok {
			return fmt.Errorf("name is required for UserCreated event")
		}
		if _, ok := data["email"]; !ok {
			return fmt.Errorf("email is required for UserCreated event")
		}
	case "OrderPlaced":
		if _, ok := data["orderId"]; !ok {
			return fmt.Errorf("orderId is required for OrderPlaced event")
		}
		if _, ok := data["userId"]; !ok {
			return fmt.Errorf("userId is required for OrderPlaced event")
		}
		if _, ok := data["total"]; !ok {
			return fmt.Errorf("total is required for OrderPlaced event")
		}
	case "PaymentSettled":
		if _, ok := data["orderId"]; !ok {
			return fmt.Errorf("orderId is required for PaymentSettled event")
		}
		if _, ok := data["status"]; !ok {
			return fmt.Errorf("status is required for PaymentSettled event")
		}
		if _, ok := data["amount"]; !ok {
			return fmt.Errorf("amount is required for PaymentSettled event")
		}
	case "InventoryAdjusted":
		if _, ok := data["sku"]; !ok {
			return fmt.Errorf("sku is required for InventoryAdjusted event")
		}
		if _, ok := data["delta"]; !ok {
			return fmt.Errorf("delta is required for InventoryAdjusted event")
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
