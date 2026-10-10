package carrier

// ProactiveConsumer listens for commands from the Optimal reasoning engine
// on the template-specific command queue: sorx.commands.{os_instance_id}.
//
// Command types:
//   - execute_action: Run an integration action locally (e.g., gmail.list_messages)
//   - request_decision: Present a decision to the user
//   - proactive_signal: Alert about something Optimal detected
//
// Results are published back to the sorx.results queue so Optimal can
// continue its SORX skill execution.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueCommands is the per-instance command queue name template.
// The full queue name is "sorx.commands.{os_instance_id}".
const QueueCommands = "sorx.commands"

// QueueResults is the queue to which action results are published.
const QueueResults = "sorx.results"

// ActionHandler is called when Optimal requests an integration action.
// The handler should execute the action locally and return the result.
type ActionHandler func(ctx context.Context, cmd ActionCommand) (any, error)

// ActionCommand represents a command from Optimal to execute locally.
type ActionCommand struct {
	Type          string         `json:"type"`
	CorrelationID string         `json:"correlation_id"`
	Action        string         `json:"action"`
	Integration   string         `json:"integration"`
	Params        map[string]any `json:"params"`
	ExecutionID   string         `json:"execution_id"`
	StepID        string         `json:"step_id"`
	OSInstanceID  string         `json:"os_instance_id"`
}

// ActionResult is sent back to Optimal after executing an action.
type ActionResult struct {
	Type          string `json:"type"` // "action_result"
	CorrelationID string `json:"correlation_id"`
	ExecutionID   string `json:"execution_id"`
	StepID        string `json:"step_id"`
	Result        any    `json:"result"`
	Error         string `json:"error,omitempty"`
}

// ProactiveConsumer listens on the instance command queue and dispatches
// each command to the registered ActionHandler, then publishes the result
// back to sorx.results.
//
// Zero value is not usable; use NewProactiveConsumer.
type ProactiveConsumer struct {
	cfg           Config
	conn          *amqp.Connection
	channel       *amqp.Channel
	commandQueue  string // "sorx.commands.{os_instance_id}"
	resultsQueue  string // "sorx.results"
	logger        *slog.Logger
	actionHandler ActionHandler

	// done is closed when Stop is called.
	done chan struct{}
}

// NewProactiveConsumer creates a ProactiveConsumer, connects to RabbitMQ,
// declares the command and results queues, and is ready to Start consuming.
//
// cfg must have Enabled=true and a non-empty OSInstanceID; otherwise an error
// is returned. The actionHandler must be non-nil.
func NewProactiveConsumer(cfg Config, actionHandler ActionHandler, logger *slog.Logger) (*ProactiveConsumer, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("proactive consumer: CARRIER is disabled")
	}
	if cfg.OSInstanceID == "" {
		return nil, fmt.Errorf("proactive consumer: OSInstanceID is required")
	}
	if actionHandler == nil {
		return nil, fmt.Errorf("proactive consumer: actionHandler must be non-nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	pc := &ProactiveConsumer{
		cfg:           cfg,
		commandQueue:  commandQueueName(cfg.OSInstanceID),
		resultsQueue:  QueueResults,
		logger:        logger.With("component", "proactive_consumer"),
		actionHandler: actionHandler,
		done:          make(chan struct{}),
	}

	if err := pc.connect(); err != nil {
		return nil, fmt.Errorf("proactive consumer: connection failed: %w", err)
	}

	return pc, nil
}

// Start begins consuming commands from the command queue in a background
// goroutine. It returns immediately; call Stop to shut down the consumer.
// Start must be called exactly once after NewProactiveConsumer.
func (pc *ProactiveConsumer) Start() {
	go pc.consume()
}

// Stop shuts down the ProactiveConsumer gracefully, closing the AMQP channel
// and connection.
func (pc *ProactiveConsumer) Stop() error {
	select {
	case <-pc.done:
		return nil // already stopped
	default:
	}

	close(pc.done)

	if pc.channel != nil {
		_ = pc.channel.Close()
	}
	if pc.conn != nil && !pc.conn.IsClosed() {
		return pc.conn.Close()
	}
	return nil
}

// ============================================================================
// Internal — connection lifecycle
// ============================================================================

// connect dials RabbitMQ, declares the command and results queues.
func (pc *ProactiveConsumer) connect() error {
	pc.logger.Info("proactive consumer: connecting to RabbitMQ", "url_masked", maskURL(pc.cfg.URL))

	conn, err := amqp.Dial(pc.cfg.URL)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	if err := ch.Qos(pc.cfg.Prefetch, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("set QoS: %w", err)
	}

	if err := pc.declareQueues(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("declare queues: %w", err)
	}

	pc.conn = conn
	pc.channel = ch

	pc.logger.Info("proactive consumer: connected",
		"command_queue", pc.commandQueue,
		"results_queue", pc.resultsQueue,
	)

	return nil
}

// declareQueues idempotently declares the command queue and results queue.
// The command queue is durable and non-exclusive so Optimal can publish
// commands before this consumer connects. The results queue mirrors the same
// durability policy.
func (pc *ProactiveConsumer) declareQueues(ch *amqp.Channel) error {
	// Durable, non-exclusive command queue — Optimal publishes here before we connect.
	if _, err := ch.QueueDeclare(
		pc.commandQueue, // name
		true,            // durable
		false,           // auto-delete
		false,           // exclusive — must be false so Optimal can publish while we're offline
		false,           // no-wait
		nil,             // args
	); err != nil {
		return fmt.Errorf("declare command queue %q: %w", pc.commandQueue, err)
	}

	// Durable results queue — Optimal consumes from here.
	if _, err := ch.QueueDeclare(
		pc.resultsQueue, // name
		true,            // durable
		false,           // auto-delete
		false,           // exclusive
		false,           // no-wait
		nil,             // args
	); err != nil {
		return fmt.Errorf("declare results queue %q: %w", pc.resultsQueue, err)
	}

	return nil
}

// ============================================================================
// Internal — consuming and publishing
// ============================================================================

// consume runs the AMQP delivery loop. It processes each delivery by
// dispatching to the actionHandler, then publishes the result to sorx.results.
// Returns when the done channel is closed or the AMQP channel is closed.
func (pc *ProactiveConsumer) consume() {
	deliveries, err := pc.channel.Consume(
		pc.commandQueue, // queue
		"",              // consumer tag — broker assigns
		false,           // auto-ack — we ack only after successful dispatch
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		pc.logger.Error("proactive consumer: failed to start consumer", "error", err)
		return
	}

	pc.logger.Info("proactive consumer: listening for commands", "queue", pc.commandQueue)

	for {
		select {
		case <-pc.done:
			return

		case delivery, ok := <-deliveries:
			if !ok {
				// Channel closed — consumer is no longer viable.
				pc.logger.Warn("proactive consumer: delivery channel closed; stopping consumer")
				return
			}
			pc.handleDelivery(delivery)
		}
	}
}

// handleDelivery decodes the delivery, calls actionHandler, and publishes the result.
// The delivery is always acknowledged (ack on success, nack-without-requeue on
// decode errors) so the queue does not accumulate unprocessable messages.
func (pc *ProactiveConsumer) handleDelivery(d amqp.Delivery) {
	var cmd ActionCommand
	if err := json.Unmarshal(d.Body, &cmd); err != nil {
		pc.logger.Error("proactive consumer: failed to unmarshal command",
			"correlation_id", d.CorrelationId,
			"error", err,
		)
		// Nack without requeue — the message is malformed.
		_ = d.Nack(false, false)
		return
	}

	pc.logger.Debug("proactive consumer: received command",
		"type", cmd.Type,
		"action", cmd.Action,
		"correlation_id", cmd.CorrelationID,
		"execution_id", cmd.ExecutionID,
		"step_id", cmd.StepID,
	)

	// Only execute_action commands trigger local handler dispatch.
	// Other types are fire-and-forget signals that do not need a result
	// published back to sorx.results; we handle them in-place and ack.
	switch cmd.Type {
	case "request_decision":
		pc.handleRequestDecision(cmd)
		_ = d.Ack(false)
		return
	case "proactive_signal":
		pc.handleProactiveSignal(cmd)
		_ = d.Ack(false)
		return
	case "execute_action":
		// falls through to the action handler below
	default:
		pc.logger.Warn("proactive consumer: unknown command type — acking without action",
			"type", cmd.Type,
			"correlation_id", cmd.CorrelationID,
		)
		_ = d.Ack(false)
		return
	}

	// Use a bounded context for action execution.
	ctx, cancel := context.WithTimeout(context.Background(), pc.cfg.SendTimeout)
	defer cancel()

	result, execErr := pc.actionHandler(ctx, cmd)

	var resultPayload ActionResult
	resultPayload.Type = "action_result"
	resultPayload.CorrelationID = cmd.CorrelationID
	resultPayload.ExecutionID = cmd.ExecutionID
	resultPayload.StepID = cmd.StepID
	resultPayload.Result = result

	if execErr != nil {
		resultPayload.Error = execErr.Error()
		pc.logger.Warn("proactive consumer: action execution failed",
			"action", cmd.Action,
			"correlation_id", cmd.CorrelationID,
			"error", execErr,
		)
	}

	if err := pc.publishResult(resultPayload, cmd.CorrelationID); err != nil {
		pc.logger.Error("proactive consumer: failed to publish result",
			"correlation_id", cmd.CorrelationID,
			"error", err,
		)
		// Still ack the delivery — the failure to publish is logged but the
		// message itself was processed; requeueing would cause duplicate execution.
	}

	_ = d.Ack(false)
}

// handleRequestDecision processes a request_decision command from Optimal.
// Optimal sends this when it needs the user to make a choice before the SORX
// execution can continue. The frontend integration (SSE push / DB notification)
// is a separate task; for now we log the full decision context at INFO level so
// that ops teams can observe pending decisions in structured logs.
func (pc *ProactiveConsumer) handleRequestDecision(cmd ActionCommand) {
	// Extract decision-specific fields from Params; all are optional.
	question, _ := cmd.Params["question"].(string)
	options, _ := cmd.Params["options"].([]interface{})
	deadline, _ := cmd.Params["deadline"].(string)
	context, _ := cmd.Params["context"].(string)

	optionStrs := make([]string, 0, len(options))
	for _, o := range options {
		if s, ok := o.(string); ok {
			optionStrs = append(optionStrs, s)
		}
	}

	pc.logger.Info("proactive consumer: decision requested by Optimal",
		"correlation_id", cmd.CorrelationID,
		"execution_id", cmd.ExecutionID,
		"step_id", cmd.StepID,
		"os_instance_id", cmd.OSInstanceID,
		"question", question,
		"options", optionStrs,
		"deadline", deadline,
		"context", context,
	)
	// TODO: publish as an SSE event or DB notification so the frontend can
	// render a decision prompt to the user. Tracked separately.
}

// handleProactiveSignal processes a proactive_signal command from Optimal.
// Optimal sends this when it detects an anomaly, metric decline, or other
// notable event that the user should be aware of. We log it at INFO with full
// details. Frontend notification delivery is a separate task.
func (pc *ProactiveConsumer) handleProactiveSignal(cmd ActionCommand) {
	// Extract signal-specific fields from Params; all are optional.
	signalType, _ := cmd.Params["signal_type"].(string)
	severity, _ := cmd.Params["severity"].(string)
	message, _ := cmd.Params["message"].(string)
	metric, _ := cmd.Params["metric"].(string)
	value, _ := cmd.Params["value"]
	threshold, _ := cmd.Params["threshold"]

	pc.logger.Info("proactive consumer: proactive signal received from Optimal",
		"correlation_id", cmd.CorrelationID,
		"execution_id", cmd.ExecutionID,
		"os_instance_id", cmd.OSInstanceID,
		"signal_type", signalType,
		"severity", severity,
		"message", message,
		"metric", metric,
		"value", value,
		"threshold", threshold,
	)
	// TODO: push as a frontend notification (SSE or DB record). Tracked separately.
}

// publishResult serialises result to JSON and delivers it to sorx.results.
// It publishes directly to the default exchange using the results queue name
// as the routing key (direct-to-queue pattern, no topic exchange required).
func (pc *ProactiveConsumer) publishResult(result ActionResult, correlationID string) error {
	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	pub := amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: correlationID,
		Timestamp:     time.Now().UTC(),
		DeliveryMode:  amqp.Persistent,
		Body:          body,
	}

	if err := pc.channel.PublishWithContext(
		context.Background(),
		"",              // default exchange — direct-to-queue
		pc.resultsQueue, // routing key = queue name for default exchange
		false,           // mandatory
		false,           // immediate
		pub,
	); err != nil {
		return fmt.Errorf("publish to %q: %w", pc.resultsQueue, err)
	}

	pc.logger.Debug("proactive consumer: published result",
		"correlation_id", correlationID,
		"queue", pc.resultsQueue,
		"error_present", result.Error != "",
	)

	return nil
}

// ============================================================================
// Helpers
// ============================================================================

// commandQueueName returns the per-OS command queue name derived from the instance ID.
func commandQueueName(osInstanceID string) string {
	return QueueCommands + "." + osInstanceID
}
