package rpc

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client manages RabbitMQ connections for RPC-style communication.
type Client struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	replyQueue string

	pending map[string]chan *Response
	mu      sync.RWMutex

	closed bool
}

// Config holds RPC client configuration.
type Config struct {
	URL      string
	Exchange string
}

// Response holds the response from an agent.
type Response struct {
	Type string
	Data map[string]any
}

// NewClient creates a new RPC client connected to RabbitMQ.
func NewClient(cfg Config) (*Client, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	// Declare exchange
	err = ch.ExchangeDeclare(cfg.Exchange, "topic", true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	// Declare exclusive reply queue
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare reply queue: %w", err)
	}

	// Bind reply queue to catch all response events
	patterns := []string{"auth.session.#", "auth.permission.#"}
	for _, pattern := range patterns {
		if err := ch.QueueBind(q.Name, pattern, cfg.Exchange, false, nil); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("bind reply queue: %w", err)
		}
	}

	client := &Client{
		conn:       conn,
		channel:    ch,
		exchange:   cfg.Exchange,
		replyQueue: q.Name,
		pending:    make(map[string]chan *Response),
	}

	// Start consumer
	go client.consume()

	return client, nil
}

// Close shuts down the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()

	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// Ready returns true if the connection is active.
func (c *Client) Ready() bool {
	return c.conn != nil && !c.conn.IsClosed()
}
