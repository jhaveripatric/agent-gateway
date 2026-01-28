package rpc

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c *Client) consume() {
	msgs, err := c.channel.Consume(c.replyQueue, "", true, true, false, false, nil)
	if err != nil {
		log.Printf("failed to consume: %v", err)
		return
	}

	for msg := range msgs {
		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg amqp.Delivery) {
	// Parse CloudEvent
	var event struct {
		Type string         `json:"type"`
		Data map[string]any `json:"data"`
	}

	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("failed to parse response: %v", err)
		return
	}

	// Find pending request by correlation ID
	c.mu.RLock()
	respChan, ok := c.pending[msg.CorrelationId]
	c.mu.RUnlock()

	if !ok {
		// No pending request - might be response to another gateway instance
		return
	}

	// Send response
	select {
	case respChan <- &Response{Type: event.Type, Data: event.Data}:
	default:
		// Channel full or closed
	}
}
