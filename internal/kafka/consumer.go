package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Consumer struct {
	reader *kafka.Reader
	logger *zap.Logger
}

func NewConsumer(brokers []string, topic, groupID string, logger *zap.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &Consumer{
		reader: reader,
		logger: logger,
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

// ReadMessage reads a message from Kafka
func (c *Consumer) ReadMessage(ctx context.Context) (*kafka.Message, error) {
	message, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	return &message, nil
}

// CommitMessage commits the offset for a message
func (c *Consumer) CommitMessage(ctx context.Context, message *kafka.Message) error {
	return c.reader.CommitMessages(ctx, *message)
}

// ParseEvent parses a Kafka message into an event structure
func (c *Consumer) ParseEvent(message *kafka.Message) (map[string]interface{}, error) {
	var event map[string]interface{}
	err := json.Unmarshal(message.Value, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Validate required fields
	if _, ok := event["eventId"]; !ok {
		return nil, fmt.Errorf("eventId field is required")
	}

	if _, ok := event["type"]; !ok {
		return nil, fmt.Errorf("type field is required")
	}

	if _, ok := event["data"]; !ok {
		return nil, fmt.Errorf("data field is required")
	}

	return event, nil
}

// LogMessage logs a message with structured fields
func (c *Consumer) LogMessage(level string, msg string, message *kafka.Message, event map[string]interface{}, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("topic", message.Topic),
		zap.Int("partition", message.Partition),
		zap.Int64("offset", message.Offset),
		zap.String("key", string(message.Key)),
	}

	if event != nil {
		if eventID, ok := event["eventId"].(string); ok {
			baseFields = append(baseFields, zap.String("eventId", eventID))
		}
		if eventType, ok := event["type"].(string); ok {
			baseFields = append(baseFields, zap.String("type", eventType))
		}
	}

	baseFields = append(baseFields, fields...)

	switch level {
	case "error":
		c.logger.Error(msg, baseFields...)
	case "warn":
		c.logger.Warn(msg, baseFields...)
	case "info":
		c.logger.Info(msg, baseFields...)
	case "debug":
		c.logger.Debug(msg, baseFields...)
	default:
		c.logger.Info(msg, baseFields...)
	}
}
