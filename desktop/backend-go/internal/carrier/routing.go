package carrier

import (
	"context"
)

// Topic constants map to SorxMain subsystem names used as AMQP routing key prefixes.
const (
	// TopicBoardroom routes to the multi-agent deliberation subsystem.
	TopicBoardroom = "boardroom"

	// TopicCritic routes to the triple-layer verification subsystem.
	TopicCritic = "critic"

	// TopicMCTS routes to the Monte Carlo Tree Search planning subsystem.
	TopicMCTS = "mcts"

	// TopicPDDL routes to the PDDL+ automated planning subsystem.
	TopicPDDL = "pddl"

	// TopicEvents routes to the system event bus.
	TopicEvents = "events"
)

// Queue names for each SorxMain subsystem.
const (
	QueueBoardroom = "sorx.boardroom"
	QueueCritic    = "sorx.critic"
	QueuePDDL      = "sorx.pddl"
	QueueMCTS      = "sorx.mcts"
	QueueEvents    = "sorx.events"
)

// RoutingKey constructs a dot-separated AMQP topic routing key from a topic
// and an action. For example, RoutingKey("mcts", "analyze_revenue") returns
// "mcts.analyze_revenue".
func RoutingKey(topic, action string) string {
	return topic + "." + action
}

// Boardroom sends a multi-agent deliberation request to SorxMain and waits
// for the response. It is a convenience wrapper around Send that sets the
// routing key and method automatically.
func (c *Client) Boardroom(ctx context.Context, action string, params map[string]any) (*Response, error) {
	return c.Send(ctx, Request{
		Method:     "deliberate",
		RoutingKey: RoutingKey(TopicBoardroom, action),
		Params:     params,
	})
}

// Critic sends a triple-layer verification request to SorxMain and waits for
// the response.
func (c *Client) Critic(ctx context.Context, action string, params map[string]any) (*Response, error) {
	return c.Send(ctx, Request{
		Method:     "verify",
		RoutingKey: RoutingKey(TopicCritic, action),
		Params:     params,
	})
}

// MCTS sends a Monte Carlo Tree Search request to SorxMain and waits for the
// response.
func (c *Client) MCTS(ctx context.Context, action string, params map[string]any) (*Response, error) {
	return c.Send(ctx, Request{
		Method:     "search",
		RoutingKey: RoutingKey(TopicMCTS, action),
		Params:     params,
	})
}

// PDDL sends a PDDL+ planning request to SorxMain and waits for the response.
func (c *Client) PDDL(ctx context.Context, action string, params map[string]any) (*Response, error) {
	return c.Send(ctx, Request{
		Method:     "plan",
		RoutingKey: RoutingKey(TopicPDDL, action),
		Params:     params,
	})
}
