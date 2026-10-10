package carrier

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/google/uuid"
)

// Client is the CARRIER AMQP bridge. It maintains a single AMQP connection and
// channel, publishes requests to the sorx.carrier exchange, and matches
// responses arriving on the per-OS reply queue to their originating callers via
// correlation IDs.
//
// Client is safe for concurrent use. Zero value is not usable; use NewClient.
type Client struct {
	cfg          Config
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchange     string // "sorx.carrier"
	replyQueue   string // "sorx.responses.{os_instance_id}"
	osInstanceID string
	logger       *slog.Logger

	// pending maps correlation_id → chan *Response for in-flight synchronous calls.
	pending sync.Map

	// done is closed when Close is called to stop background goroutines.
	done chan struct{}

	// mu protects conn and channel during reconnection.
	mu sync.RWMutex

	// connected tracks whether the AMQP channel is usable.
	connected atomic.Bool
}

// NewClient creates a CARRIER client, connects to RabbitMQ, declares the
// exchange and all well-known queues, and starts the response consumer.
//
// Returns an error if the initial connection or topology setup fails.
// If cfg.Enabled is false the client is returned in a disabled state; all
// Send calls will immediately return a FallbackError with ReasonDisabled.
func NewClient(cfg Config, logger *slog.Logger) (*Client, error) {
	if logger == nil {
		logger = slog.Default()
	}

	c := &Client{
		cfg:          cfg,
		exchange:     cfg.Exchange,
		osInstanceID: cfg.OSInstanceID,
		replyQueue:   replyQueueName(cfg.OSInstanceID),
		logger:       logger.With("component", "carrier"),
		done:         make(chan struct{}),
	}

	if !cfg.Enabled {
		c.logger.Info("carrier disabled via feature flag; operating in fallback-only mode")
		return c, nil
	}

	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("carrier: initial connection failed: %w", err)
	}

	go c.watchConnection()

	return c, nil
}

// Close shuts down the CARRIER client gracefully. Pending synchronous Send
// calls will receive a FallbackError with ReasonDisconnected.
func (c *Client) Close() error {
	select {
	case <-c.done:
		return nil // already closed
	default:
	}

	close(c.done)

	// Drain pending callers so they are not blocked forever.
	c.pending.Range(func(key, value any) bool {
		ch, ok := value.(chan *Response)
		if ok {
			close(ch)
		}
		c.pending.Delete(key)
		return true
	})

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil && !c.conn.IsClosed() {
		return c.conn.Close()
	}
	return nil
}

// IsConnected reports whether the underlying AMQP channel is ready for use.
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

// Send publishes a request to SorxMain and blocks until the matching response
// arrives or the context / send timeout expires.
//
// When SorxMain is unavailable (CARRIER disabled, AMQP down, timeout), Send
// returns a *FallbackError. Use IsFallback to detect this condition and
// degrade gracefully.
func (c *Client) Send(ctx context.Context, msg Request) (*Response, error) {
	if !c.cfg.Enabled {
		return nil, newFallback(ReasonDisabled, nil)
	}
	if !c.IsConnected() {
		return nil, newFallback(ReasonDisconnected, nil)
	}

	// Assign an ID if the caller did not provide one.
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	correlationID := msg.ID

	// Fill in sender context and defaults.
	msg.Context.OSInstanceID = c.osInstanceID
	msg.sanitize()

	// Register a reply channel before publishing to avoid a race where the
	// response arrives before we start waiting.
	replyCh := make(chan *Response, 1)
	c.pending.Store(correlationID, replyCh)
	defer c.pending.Delete(correlationID)

	if err := c.publish(msg, correlationID); err != nil {
		return nil, newFallback(ReasonDisconnected, err)
	}

	// Determine effective deadline: the shorter of ctx and cfg.SendTimeout.
	timeout := c.cfg.SendTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp, ok := <-replyCh:
		if !ok {
			// Channel was closed by Close(); connection lost mid-flight.
			return nil, newFallback(ReasonDisconnected, nil)
		}
		if resp.Error != nil {
			return resp, resp.Error
		}
		return resp, nil

	case <-timer.C:
		return nil, newFallback(ReasonTimeout, fmt.Errorf("no response after %s", timeout))

	case <-ctx.Done():
		return nil, ctx.Err()

	case <-c.done:
		return nil, newFallback(ReasonDisconnected, nil)
	}
}

// SendAsync publishes a request and returns immediately with the correlation ID
// that SorxMain will echo in its response. The caller is responsible for
// listening on the reply queue independently. Useful for fire-and-forget
// notifications or when the caller has its own response routing.
func (c *Client) SendAsync(ctx context.Context, msg Request) (string, error) {
	if !c.cfg.Enabled {
		return "", newFallback(ReasonDisabled, nil)
	}
	if !c.IsConnected() {
		return "", newFallback(ReasonDisconnected, nil)
	}

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	correlationID := msg.ID
	msg.Context.OSInstanceID = c.osInstanceID
	msg.sanitize()

	if err := c.publish(msg, correlationID); err != nil {
		return "", newFallback(ReasonDisconnected, err)
	}
	return correlationID, nil
}

// ============================================================================
// Internal — connection lifecycle
// ============================================================================

// connect dials RabbitMQ, declares topology, and starts the consumer goroutine.
// It must be called with no locks held. On success it sets connected to true.
func (c *Client) connect() error {
	c.logger.Info("carrier: connecting to RabbitMQ", "url_masked", maskURL(c.cfg.URL))

	conn, err := amqp.Dial(c.cfg.URL)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	if err := ch.Qos(c.cfg.Prefetch, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("set QoS: %w", err)
	}

	if err := c.declareTopology(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("declare topology: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.channel = ch
	c.mu.Unlock()

	c.connected.Store(true)
	c.logger.Info("carrier: connected to RabbitMQ", "reply_queue", c.replyQueue)

	go c.consume(ch)

	return nil
}

// declareTopology idempotently declares the exchange, well-known queues, and
// the per-OS reply queue on the provided channel.
func (c *Client) declareTopology(ch *amqp.Channel) error {
	// Declare the topic exchange.
	if err := ch.ExchangeDeclare(
		c.exchange, // name
		"topic",    // kind
		true,       // durable
		false,      // auto-delete
		false,      // internal
		false,      // no-wait
		nil,        // args
	); err != nil {
		return fmt.Errorf("declare exchange %q: %w", c.exchange, err)
	}

	// Declare each well-known subsystem queue and bind it.
	subsystems := []struct {
		queue   string
		pattern string
	}{
		{QueueBoardroom, TopicBoardroom + ".*"},
		{QueueCritic, TopicCritic + ".*"},
		{QueuePDDL, TopicPDDL + ".*"},
		{QueueMCTS, TopicMCTS + ".*"},
		{QueueEvents, TopicEvents + ".*"},
	}

	for _, s := range subsystems {
		if _, err := ch.QueueDeclare(
			s.queue, // name
			true,    // durable
			false,   // auto-delete
			false,   // exclusive
			false,   // no-wait
			amqp.Table{"x-max-priority": int32(MaxPriority)},
		); err != nil {
			return fmt.Errorf("declare queue %q: %w", s.queue, err)
		}

		if err := ch.QueueBind(s.queue, s.pattern, c.exchange, false, nil); err != nil {
			return fmt.Errorf("bind queue %q to pattern %q: %w", s.queue, s.pattern, err)
		}
	}

	// Declare the per-OS reply queue. Exclusive + auto-delete so it disappears
	// when this client disconnects.
	if _, err := ch.QueueDeclare(
		c.replyQueue, // name
		false,        // durable — ephemeral reply queue
		true,         // auto-delete
		true,         // exclusive
		false,        // no-wait
		nil,          // args
	); err != nil {
		return fmt.Errorf("declare reply queue %q: %w", c.replyQueue, err)
	}

	return nil
}

// watchConnection monitors the AMQP connection for unexpected closure and
// triggers reconnection with exponential backoff.
func (c *Client) watchConnection() {
	for {
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		// Block until the broker closes the connection.
		closeErr := <-conn.NotifyClose(make(chan *amqp.Error, 1))

		select {
		case <-c.done:
			return
		default:
		}

		c.connected.Store(false)

		if closeErr != nil {
			c.logger.Warn("carrier: AMQP connection closed unexpectedly",
				"code", closeErr.Code,
				"reason", closeErr.Reason,
			)
		}

		c.reconnect()
	}
}

// reconnect attempts to re-establish the AMQP connection with exponential
// backoff (1s → 2s → 4s → … → 30s cap). It returns only when connected or
// when the client is closed.
func (c *Client) reconnect() {
	const (
		initialBackoff = 1 * time.Second
		maxBackoff     = 30 * time.Second
	)

	backoff := initialBackoff
	attempt := 0

	for {
		select {
		case <-c.done:
			return
		default:
		}

		attempt++
		c.logger.Info("carrier: attempting reconnect", "attempt", attempt, "backoff", backoff)

		if err := c.connect(); err != nil {
			c.logger.Warn("carrier: reconnect failed", "attempt", attempt, "error", err)

			select {
			case <-c.done:
				return
			case <-time.After(backoff):
			}

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		c.logger.Info("carrier: reconnected successfully", "attempt", attempt)
		return
	}
}

// ============================================================================
// Internal — publishing and consuming
// ============================================================================

// publish serialises msg to JSON and delivers it to the AMQP exchange.
func (c *Client) publish(msg Request, correlationID string) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("channel not available")
	}

	pub := amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: correlationID,
		ReplyTo:       c.replyQueue,
		MessageId:     msg.ID,
		Timestamp:     time.Now().UTC(),
		DeliveryMode:  amqp.Transient, // speed over durability for reasoning requests
		Priority:      uint8(msg.Priority),
		Expiration:    fmt.Sprintf("%d", msg.TTL),
		Body:          body,
	}

	if err := ch.PublishWithContext(
		context.Background(),
		c.exchange,     // exchange
		msg.RoutingKey, // routing key
		false,          // mandatory
		false,          // immediate
		pub,
	); err != nil {
		c.connected.Store(false)
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

// consume runs in its own goroutine. It reads deliveries from the reply queue
// and routes each message to the pending caller channel matched by correlation ID.
// When the channel is closed (connection loss or client shutdown) it returns.
func (c *Client) consume(ch *amqp.Channel) {
	deliveries, err := ch.Consume(
		c.replyQueue, // queue
		"",           // consumer tag — broker assigns one
		true,         // auto-ack — responses are idempotent
		true,         // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		c.logger.Error("carrier: failed to start consumer", "error", err)
		c.connected.Store(false)
		return
	}

	c.logger.Debug("carrier: consumer started", "queue", c.replyQueue)

	for {
		select {
		case <-c.done:
			return

		case delivery, ok := <-deliveries:
			if !ok {
				// Channel closed — watchConnection will handle reconnection.
				c.logger.Debug("carrier: delivery channel closed")
				return
			}
			c.handleDelivery(delivery)
		}
	}
}

// handleDelivery decodes a delivery and forwards it to the waiting Send caller.
// Unmatched messages (expired pending entries, duplicate deliveries) are logged
// and dropped.
func (c *Client) handleDelivery(d amqp.Delivery) {
	correlationID := d.CorrelationId
	if correlationID == "" {
		c.logger.Warn("carrier: received delivery with empty correlation ID; dropping")
		return
	}

	value, ok := c.pending.Load(correlationID)
	if !ok {
		c.logger.Debug("carrier: no pending caller for correlation ID; dropping",
			"correlation_id", correlationID,
		)
		return
	}

	replyCh, ok := value.(chan *Response)
	if !ok {
		c.logger.Error("carrier: pending map held unexpected type",
			"correlation_id", correlationID,
		)
		return
	}

	var resp Response
	if err := json.Unmarshal(d.Body, &resp); err != nil {
		c.logger.Error("carrier: failed to unmarshal response",
			"correlation_id", correlationID,
			"error", err,
		)
		// Deliver a synthetic error response so the caller is not blocked.
		resp = Response{
			CorrelationID: correlationID,
			Error: &ResponseError{
				Code:    "DECODE_ERROR",
				Message: "failed to decode SorxMain response",
				Details: err.Error(),
			},
		}
	}

	select {
	case replyCh <- &resp:
	default:
		// The channel already has a value (should not happen with buffer=1).
		c.logger.Warn("carrier: reply channel full; dropping response",
			"correlation_id", correlationID,
		)
	}
}

// ============================================================================
// Helpers
// ============================================================================

// maskURL redacts credentials from an AMQP URL for safe logging.
// amqp://user:pass@host:port/vhost → amqp://***:***@host:port/vhost
func maskURL(raw string) string {
	// Simple prefix-based masking to avoid importing net/url just for logging.
	for _, scheme := range []string{"amqp://", "amqps://"} {
		if len(raw) <= len(scheme) {
			continue
		}
		rest := raw[len(scheme):]
		// Find the '@' that separates credentials from host.
		for i := 0; i < len(rest); i++ {
			if rest[i] == '@' {
				return scheme + "***:***@" + rest[i+1:]
			}
		}
	}
	return raw
}
