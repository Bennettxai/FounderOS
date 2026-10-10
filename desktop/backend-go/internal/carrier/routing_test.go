package carrier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// RoutingKey
// ============================================================================

func TestRoutingKey_ProducesCorrectFormat(t *testing.T) {
	got := RoutingKey("mcts", "analyze_revenue")

	assert.Equal(t, "mcts.analyze_revenue", got)
}

func TestRoutingKey_TableDriven(t *testing.T) {
	tests := []struct {
		topic    string
		action   string
		expected string
	}{
		{TopicBoardroom, "strategy", "boardroom.strategy"},
		{TopicCritic, "verify_plan", "critic.verify_plan"},
		{TopicMCTS, "analyze_revenue", "mcts.analyze_revenue"},
		{TopicPDDL, "generate", "pddl.generate"},
		{TopicEvents, "user_joined", "events.user_joined"},
		// Edge cases
		{"", "action", ".action"},
		{"topic", "", "topic."},
		{"", "", "."},
	}

	for _, tc := range tests {
		t.Run(tc.topic+"."+tc.action, func(t *testing.T) {
			got := RoutingKey(tc.topic, tc.action)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// ============================================================================
// Topic constants
// ============================================================================

func TestTopicConstants_HaveExpectedValues(t *testing.T) {
	assert.Equal(t, "boardroom", TopicBoardroom)
	assert.Equal(t, "critic", TopicCritic)
	assert.Equal(t, "mcts", TopicMCTS)
	assert.Equal(t, "pddl", TopicPDDL)
	assert.Equal(t, "events", TopicEvents)
}

// ============================================================================
// Queue constants
// ============================================================================

func TestQueueConstants_HaveExpectedValues(t *testing.T) {
	assert.Equal(t, "sorx.boardroom", QueueBoardroom)
	assert.Equal(t, "sorx.critic", QueueCritic)
	assert.Equal(t, "sorx.pddl", QueuePDDL)
	assert.Equal(t, "sorx.mcts", QueueMCTS)
	assert.Equal(t, "sorx.events", QueueEvents)
}

func TestQueueConstants_AllHaveSorxPrefix(t *testing.T) {
	queues := []string{
		QueueBoardroom,
		QueueCritic,
		QueuePDDL,
		QueueMCTS,
		QueueEvents,
	}

	for _, q := range queues {
		assert.Contains(t, q, "sorx.", "queue %q should have sorx. prefix", q)
	}
}

func TestTopicAndQueueAlignment(t *testing.T) {
	// Each queue name should embed its corresponding topic.
	pairs := []struct {
		topic string
		queue string
	}{
		{TopicBoardroom, QueueBoardroom},
		{TopicCritic, QueueCritic},
		{TopicMCTS, QueueMCTS},
		{TopicPDDL, QueuePDDL},
		{TopicEvents, QueueEvents},
	}

	for _, p := range pairs {
		assert.Contains(t, p.queue, p.topic,
			"queue %q should contain topic %q", p.queue, p.topic)
	}
}
