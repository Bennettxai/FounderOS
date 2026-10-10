package carrier

// registration.go implements template registration and heartbeat with Optimal.
//
// On startup, BOS announces itself to Optimal via events.register, advertising
// its instance ID, template type, installed modules, and supported integrations.
// Periodic heartbeats (events.heartbeat) keep the connection status alive so
// Optimal can detect disconnected instances and hold commands until reconnect.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RegistrationMessage is sent to Optimal on startup via events.register.
type RegistrationMessage struct {
	// Type is always "registration".
	Type string `json:"type"`

	// OSInstanceID uniquely identifies this BOS instance.
	OSInstanceID string `json:"os_instance_id"`

	// TemplateType identifies the kind of template being registered
	// (e.g. "bos", "custom_os", "content_os").
	TemplateType string `json:"template_type"`

	// InstalledModules lists active modules for this workspace
	// (e.g. ["crm", "projects", "calendar"]).
	InstalledModules []string `json:"installed_modules"`

	// Capabilities lists integrations this instance supports
	// (e.g. ["gmail", "google_calendar", "slack", "hubspot"]).
	Capabilities []string `json:"capabilities"`

	// ReplyQueue is the AMQP queue name where Optimal should send responses
	// for synchronous CARRIER requests from this instance.
	ReplyQueue string `json:"reply_queue"`

	// CommandQueue is the AMQP queue name where Optimal should publish
	// proactive commands (execute_action, request_decision, proactive_signal).
	CommandQueue string `json:"command_queue"`

	// RegisteredAt is the UTC Unix timestamp when this registration was sent.
	RegisteredAt int64 `json:"registered_at"`
}

// HeartbeatMessage is sent periodically to keep the instance marked as connected.
type HeartbeatMessage struct {
	// Type is always "heartbeat".
	Type string `json:"type"`

	// OSInstanceID identifies this BOS instance.
	OSInstanceID string `json:"os_instance_id"`

	// Timestamp is the UTC Unix timestamp of this heartbeat.
	Timestamp int64 `json:"timestamp"`
}

// RegisterWithOptimal sends a registration message to Optimal via the
// events.register routing key on the sorx.carrier exchange. It must be called
// once after the CARRIER client is connected.
//
// modules is the list of active workspace modules (e.g. ["crm", "projects"]).
// capabilities is the list of supported integrations (e.g. ["gmail", "slack"]).
func (c *Client) RegisterWithOptimal(ctx context.Context, modules []string, capabilities []string) error {
	if !c.cfg.Enabled {
		return fmt.Errorf("registration: CARRIER is disabled")
	}
	if !c.IsConnected() {
		return fmt.Errorf("registration: CARRIER is not connected")
	}

	reg := RegistrationMessage{
		Type:             "registration",
		OSInstanceID:     c.osInstanceID,
		TemplateType:     "bos",
		InstalledModules: modules,
		Capabilities:     capabilities,
		ReplyQueue:       c.replyQueue,
		CommandQueue:     commandQueueName(c.osInstanceID),
		RegisteredAt:     time.Now().UTC().UnixMilli(),
	}

	body, err := json.Marshal(reg)
	if err != nil {
		return fmt.Errorf("registration: marshal: %w", err)
	}

	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("registration: channel not available")
	}

	pub := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	}

	if err := ch.PublishWithContext(
		ctx,
		c.exchange,                          // sorx.carrier exchange
		RoutingKey(TopicEvents, "register"), // events.register
		false,                               // mandatory
		false,                               // immediate
		pub,
	); err != nil {
		return fmt.Errorf("registration: publish: %w", err)
	}

	c.logger.Info("carrier: registered with Optimal",
		"os_instance_id", c.osInstanceID,
		"template_type", reg.TemplateType,
		"modules", modules,
		"capabilities", capabilities,
		"command_queue", reg.CommandQueue,
	)

	return nil
}

// SendHeartbeat publishes a single heartbeat message to Optimal via the
// events.heartbeat routing key. Returns an error if CARRIER is disabled or
// disconnected; these are non-fatal during normal operation.
func (c *Client) SendHeartbeat(ctx context.Context) error {
	if !c.cfg.Enabled {
		return fmt.Errorf("heartbeat: CARRIER is disabled")
	}
	if !c.IsConnected() {
		return fmt.Errorf("heartbeat: CARRIER is not connected")
	}

	hb := HeartbeatMessage{
		Type:         "heartbeat",
		OSInstanceID: c.osInstanceID,
		Timestamp:    time.Now().UTC().UnixMilli(),
	}

	body, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("heartbeat: marshal: %w", err)
	}

	c.mu.RLock()
	ch := c.channel
	c.mu.RUnlock()

	if ch == nil {
		return fmt.Errorf("heartbeat: channel not available")
	}

	pub := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Transient, // heartbeats are ephemeral
		Timestamp:    time.Now().UTC(),
		Body:         body,
	}

	if err := ch.PublishWithContext(
		ctx,
		c.exchange,                           // sorx.carrier exchange
		RoutingKey(TopicEvents, "heartbeat"), // events.heartbeat
		false,                                // mandatory
		false,                                // immediate
		pub,
	); err != nil {
		return fmt.Errorf("heartbeat: publish: %w", err)
	}

	c.logger.Debug("carrier: heartbeat sent", "os_instance_id", c.osInstanceID)

	return nil
}

// StartHeartbeat begins sending periodic heartbeats to Optimal in a background
// goroutine. The goroutine stops when the client's done channel is closed
// (i.e. when Close is called), or when ctx is cancelled.
//
// Heartbeat failures are logged at Warn level and do not stop the loop — a
// temporary disconnection should not prevent reconnection attempts.
func (c *Client) StartHeartbeat(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		c.logger.Info("carrier: heartbeat started", "interval", interval)

		for {
			select {
			case <-c.done:
				c.logger.Debug("carrier: heartbeat stopped (client closed)")
				return

			case <-ctx.Done():
				c.logger.Debug("carrier: heartbeat stopped (context cancelled)")
				return

			case <-ticker.C:
				if err := c.SendHeartbeat(ctx); err != nil {
					// Non-fatal — CARRIER may be reconnecting.
					c.logger.Warn("carrier: heartbeat failed", "error", err)
				}
			}
		}
	}()
}

// SetTemplateType overrides the template type reported during registration.
// Must be called before RegisterWithOptimal if a non-default type is needed.
// This is a no-op on the current Client; use OptimalConfig.TemplateType and
// pass the resolved value to RegisterWithOptimal instead.
//
// This stub exists as a documentation anchor for callers that want to use
// "custom_os" or "content_os" template types.
func templateTypeOrDefault(t string) string {
	if t == "" {
		return "bos"
	}
	return t
}

// logRegistrationSkipped logs a structured message explaining why registration
// was skipped. Used by main.go for consistent startup log output.
func logRegistrationSkipped(logger *slog.Logger, reason string) {
	logger.Info("carrier: Optimal registration skipped", "reason", reason)
}
