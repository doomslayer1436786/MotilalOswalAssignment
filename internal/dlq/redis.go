package dlq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type RedisDLQ struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisDLQ(addr, password string, logger *zap.Logger) (*RedisDLQ, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisDLQ{
		client: client,
		logger: logger,
	}, nil
}

func (d *RedisDLQ) Close() error {
	return d.client.Close()
}

// PushMessage pushes a failed message to the dead letter queue
func (d *RedisDLQ) PushMessage(ctx context.Context, topic string, partition int, offset int64, payload interface{}, errorMsg string) error {
	dlqMsg := map[string]interface{}{
		"eventId":   extractEventID(payload),
		"topic":     topic,
		"partition": partition,
		"offset":    offset,
		"payload":   payload,
		"error":     errorMsg,
		"failedAt":  time.Now().UTC(),
	}

	jsonData, err := json.Marshal(dlqMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	// Push to Redis list (newest first)
	key := fmt.Sprintf("dlq:%s", topic)
	err = d.client.LPush(ctx, key, jsonData).Err()
	if err != nil {
		return fmt.Errorf("failed to push to DLQ: %w", err)
	}

	// Log the DLQ push
	eventID := extractEventID(payload)
	d.logger.Error("message pushed to DLQ",
		zap.String("eventId", eventID),
		zap.String("topic", topic),
		zap.Int("partition", partition),
		zap.Int64("offset", offset),
		zap.String("error", errorMsg),
	)

	return nil
}

// GetMessages retrieves messages from the dead letter queue
func (d *RedisDLQ) GetMessages(ctx context.Context, topic string, start, stop int64) ([]string, error) {
	key := fmt.Sprintf("dlq:%s", topic)
	return d.client.LRange(ctx, key, start, stop).Result()
}

// extractEventID attempts to extract eventId from the payload
func extractEventID(payload interface{}) string {
	if payloadMap, ok := payload.(map[string]interface{}); ok {
		if eventID, exists := payloadMap["eventId"]; exists {
			if eventIDStr, ok := eventID.(string); ok {
				return eventIDStr
			}
		}
	}
	return "unknown"
}
