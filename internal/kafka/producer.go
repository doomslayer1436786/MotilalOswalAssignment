package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

func NewProducer(brokers []string, topic string, logger *zap.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	return &Producer{
		writer: writer,
		logger: logger,
	}
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

// PublishEvent publishes an event to Kafka with the appropriate key
func (p *Producer) PublishEvent(ctx context.Context, event interface{}) error {
	// Marshal the event to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Extract the key based on event type
	key, err := p.extractKey(event)
	if err != nil {
		return fmt.Errorf("failed to extract key: %w", err)
	}

	// Create Kafka message
	message := kafka.Message{
		Key:   []byte(key),
		Value: jsonData,
		Time:  time.Now(),
	}

	// Publish to Kafka
	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	// Log successful publish
	eventID := p.extractEventID(event)
	eventType := p.extractEventType(event)
	p.logger.Info("event published to Kafka",
		zap.String("eventId", eventID),
		zap.String("type", eventType),
		zap.String("key", key),
		zap.String("topic", p.writer.Topic),
	)

	return nil
}

// extractKey extracts the appropriate key based on event type
func (p *Producer) extractKey(event interface{}) (string, error) {
	eventMap, ok := event.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("event is not a map")
	}

	eventType, ok := eventMap["type"].(string)
	if !ok {
		return "", fmt.Errorf("event type is not a string")
	}

	data, ok := eventMap["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("event data is not a map")
	}

	switch eventType {
	case "UserCreated":
		if userID, ok := data["userId"].(string); ok {
			return userID, nil
		}
		return "", fmt.Errorf("userId not found in UserCreated event")
	case "OrderPlaced":
		if orderID, ok := data["orderId"].(string); ok {
			return orderID, nil
		}
		return "", fmt.Errorf("orderId not found in OrderPlaced event")
	case "PaymentSettled":
		if orderID, ok := data["orderId"].(string); ok {
			return orderID, nil
		}
		return "", fmt.Errorf("orderId not found in PaymentSettled event")
	case "InventoryAdjusted":
		if sku, ok := data["sku"].(string); ok {
			return sku, nil
		}
		return "", fmt.Errorf("sku not found in InventoryAdjusted event")
	case "ProductReview":
		if reviewID, ok := data["reviewId"].(string); ok {
			return reviewID, nil
		}
		return "", fmt.Errorf("reviewId not found in ProductReview event")
	default:
		return "", fmt.Errorf("unknown event type: %s", eventType)
	}
}

// extractEventID extracts the eventId from the event
func (p *Producer) extractEventID(event interface{}) string {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if eventID, ok := eventMap["eventId"].(string); ok {
			return eventID
		}
	}
	return "unknown"
}

// extractEventType extracts the type from the event
func (p *Producer) extractEventType(event interface{}) string {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if eventType, ok := eventMap["type"].(string); ok {
			return eventType
		}
	}
	return "unknown"
}
