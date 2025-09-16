package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"kafka-pipeline/internal/dlq"
	"kafka-pipeline/internal/kafka"
	"kafka-pipeline/internal/store"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kafkaGo "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var (
	messagesProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_processed_total",
			Help: "Total number of messages processed",
		},
		[]string{"type"},
	)

	dlqCountTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_count_total",
			Help: "Total number of messages sent to DLQ",
		},
	)

	dbLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_latency_seconds",
			Help:    "Database operation latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(messagesProcessedTotal)
	prometheus.MustRegister(dlqCountTotal)
	prometheus.MustRegister(dbLatencySeconds)
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
	kafkaGroupID := getEnv("KAFKA_GROUP_ID", "consumer-group")
	mssqlConn := getEnv("MSSQL_CONN", "server=localhost;user id=sa;password=Your_strong_pwd1;database=events;encrypt=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	servicePort := getEnv("SERVICE_PORT", "8081")

	// Initialize Kafka consumer
	brokers := strings.Split(kafkaBrokers, ",")
	consumer := kafka.NewConsumer(brokers, kafkaTopic, kafkaGroupID, logger)
	defer consumer.Close()

	// Initialize MS SQL store
	sqlStore, err := store.NewMSSQLStore(mssqlConn, logger)
	if err != nil {
		logger.Fatal("Failed to initialize SQL store", zap.Error(err))
	}
	defer sqlStore.Close()

	// Initialize Redis DLQ
	dlq, err := dlq.NewRedisDLQ(redisAddr, redisPassword, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Redis DLQ", zap.Error(err))
	}
	defer dlq.Close()

	// Start metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		server := &http.Server{
			Addr:    ":" + servicePort,
			Handler: mux,
		}

		logger.Info("Starting metrics server", zap.String("port", servicePort))
		if err := server.ListenAndServe(); err != nil {
			logger.Error("Metrics server error", zap.Error(err))
		}
	}()

	// Start consuming messages
	logger.Info("Starting consumer",
		zap.String("topic", kafkaTopic),
		zap.String("groupID", kafkaGroupID),
	)

	ctx := context.Background()
	for {
		message, err := consumer.ReadMessage(ctx)
		if err != nil {
			logger.Error("Failed to read message", zap.Error(err))
			continue
		}

		// Process message
		if err := processMessage(ctx, message, consumer, sqlStore, dlq, logger); err != nil {
			logger.Error("Failed to process message", zap.Error(err))
		}
	}
}

func processMessage(ctx context.Context, message *kafkaGo.Message, consumer *kafka.Consumer, sqlStore *store.MSSQLStore, dlq *dlq.RedisDLQ, logger *zap.Logger) error {
	// Parse event
	event, err := consumer.ParseEvent(message)
	if err != nil {
		// Push to DLQ and commit offset
		dlqErr := dlq.PushMessage(ctx, message.Topic, message.Partition, message.Offset, string(message.Value), err.Error())
		if dlqErr != nil {
			logger.Error("Failed to push to DLQ", zap.Error(dlqErr))
		} else {
			dlqCountTotal.Inc()
		}

		consumer.LogMessage("error", "Failed to parse event", message, nil, zap.Error(err))

		// Commit offset even after DLQ push
		if commitErr := consumer.CommitMessage(ctx, message); commitErr != nil {
			logger.Error("Failed to commit offset after parse error", zap.Error(commitErr))
		}
		return err
	}

	// Process based on event type
	eventType := event["type"].(string)

	start := time.Now()
	err = processEventByType(ctx, event, sqlStore, logger)
	duration := time.Since(start)

	// Record DB latency
	dbLatencySeconds.WithLabelValues("upsert").Observe(duration.Seconds())

	if err != nil {
		// Push to DLQ and commit offset
		dlqErr := dlq.PushMessage(ctx, message.Topic, message.Partition, message.Offset, event, err.Error())
		if dlqErr != nil {
			logger.Error("Failed to push to DLQ", zap.Error(dlqErr))
		} else {
			dlqCountTotal.Inc()
		}

		consumer.LogMessage("error", "Failed to process event", message, event,
			zap.Error(err),
			zap.Duration("latency_ms", duration),
		)

		// Commit offset even after DLQ push
		if commitErr := consumer.CommitMessage(ctx, message); commitErr != nil {
			logger.Error("Failed to commit offset after processing error", zap.Error(commitErr))
		}
		return err
	}

	// Success - increment counter and commit offset
	messagesProcessedTotal.WithLabelValues(eventType).Inc()

	consumer.LogMessage("info", "Event processed successfully", message, event,
		zap.Duration("latency_ms", duration),
	)

	if err := consumer.CommitMessage(ctx, message); err != nil {
		logger.Error("Failed to commit offset", zap.Error(err))
		return err
	}

	return nil
}

func processEventByType(ctx context.Context, event map[string]interface{}, sqlStore *store.MSSQLStore, logger *zap.Logger) error {
	eventType := event["type"].(string)
	data := event["data"].(map[string]interface{})

	switch eventType {
	case "UserCreated":
		user := &store.User{
			UserID:    data["userId"].(string),
			Name:      data["name"].(string),
			Email:     data["email"].(string),
			CreatedAt: parseTime(data["createdAt"]),
			UpdatedAt: time.Now(),
		}
		return sqlStore.UpsertUser(ctx, user)

	case "OrderPlaced":
		order := &store.Order{
			OrderID:   data["orderId"].(string),
			UserID:    data["userId"].(string),
			Total:     parseFloat64(data["total"]),
			Status:    "placed",
			CreatedAt: parseTime(data["createdAt"]),
			UpdatedAt: time.Now(),
		}
		return sqlStore.UpsertOrder(ctx, order)

	case "PaymentSettled":
		payment := &store.Payment{
			OrderID:   data["orderId"].(string),
			Status:    data["status"].(string),
			Amount:    parseFloat64(data["amount"]),
			SettledAt: parseTime(data["settledAt"]),
			UpdatedAt: time.Now(),
		}
		return sqlStore.UpsertPayment(ctx, payment)

	case "InventoryAdjusted":
		inventory := &store.Inventory{
			SKU:            data["sku"].(string),
			Quantity:       parseInt(data["delta"]),
			LastAdjustedAt: parseTime(data["adjustedAt"]),
		}
		return sqlStore.UpsertInventory(ctx, inventory)

	case "ProductReview":
		review := &store.ProductReview{
			ReviewID:    data["reviewId"].(string),
			ProductName: data["productName"].(string),
			Username:    data["username"].(string),
			Rating:      parseInt(data["rating"]),
			Remarks:     data["remarks"].(string),
			CreatedAt:   parseTime(data["createdAt"]),
			UpdatedAt:   time.Now(),
		}
		return sqlStore.UpsertProductReview(ctx, review)

	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}
}

func parseTime(value interface{}) time.Time {
	if str, ok := value.(string); ok {
		if t, err := time.Parse(time.RFC3339, str); err == nil {
			return t
		}
	}
	return time.Now()
}

func parseFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func parseInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
