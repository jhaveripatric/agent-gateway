package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ErrTimeout is returned when an RPC call times out.
var ErrTimeout = fmt.Errorf("request timeout")

// Call publishes an event and waits for response.
func (c *Client) Call(ctx context.Context, eventType string, data map[string]any, timeout time.Duration) (*Response, error) {
	correlationID := uuid.New().String()

	// Create response channel
	respChan := make(chan *Response, 1)
	c.mu.Lock()
	c.pending[correlationID] = respChan
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, correlationID)
		c.mu.Unlock()
	}()

	// Build CloudEvent
	event := map[string]any{
		"specversion":     "1.0",
		"id":              uuid.New().String(),
		"type":            eventType,
		"source":          "/agent-gateway",
		"time":            time.Now().Format(time.RFC3339),
		"datacontenttype": "application/json",
		"data":            data,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}

	// Extract routing key from event type
	routingKey := extractRoutingKey(eventType)

	// Publish
	err = c.channel.PublishWithContext(ctx,
		c.exchange,
		routingKey,
		false, false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			ReplyTo:       c.replyQueue,
			Body:          body,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("publish: %w", err)
	}

	// Wait for response
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// extractRoutingKey converts event type to routing key.
// io.agenteco.auth.login.requested.v1 -> auth.login.requested
func extractRoutingKey(eventType string) string {
	parts := strings.Split(eventType, ".")
	if len(parts) < 5 || parts[0] != "io" || parts[1] != "agenteco" {
		return eventType
	}

	// Remove io.agenteco. prefix and .vN suffix
	middle := parts[2 : len(parts)-1]
	return strings.Join(middle, ".")
}
