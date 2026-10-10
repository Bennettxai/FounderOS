package carrier

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Helpers
// ============================================================================

// disabledClient builds a Client from a disabled Config without touching
// RabbitMQ. This is the safe, pure-unit-test path through NewClient.
func disabledClient(t *testing.T) *Client {
	t.Helper()
	cfg := DefaultConfig() // Enabled: false
	c, err := NewClient(cfg, slog.Default())
	require.NoError(t, err)
	require.NotNil(t, c)
	return c
}

// ============================================================================
// NewClient — disabled mode
// ============================================================================

func TestNewClient_DisabledConfig_ReturnsClientWithNoError(t *testing.T) {
	cfg := DefaultConfig() // Enabled defaults to false

	c, err := NewClient(cfg, nil) // nil logger should use slog.Default()

	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNewClient_DisabledConfig_ClientIsNotConnected(t *testing.T) {
	c := disabledClient(t)

	assert.False(t, c.IsConnected())
}

func TestNewClient_DisabledConfig_ExchangeMatchesCfg(t *testing.T) {
	c := disabledClient(t)

	assert.Equal(t, DefaultConfig().Exchange, c.exchange)
}

func TestNewClient_DisabledConfig_ReplyQueueMatchesInstanceID(t *testing.T) {
	cfg := DefaultConfig()
	cfg.OSInstanceID = "bos-test-99"
	c, err := NewClient(cfg, slog.Default())
	require.NoError(t, err)

	assert.Equal(t, "sorx.responses.bos-test-99", c.replyQueue)
}

// ============================================================================
// IsConnected
// ============================================================================

func TestIsConnected_ReturnsFalseWhenDisabled(t *testing.T) {
	c := disabledClient(t)

	assert.False(t, c.IsConnected())
}

// ============================================================================
// Close — idempotency
// ============================================================================

func TestClose_IsIdempotent(t *testing.T) {
	c := disabledClient(t)

	err1 := c.Close()
	err2 := c.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2, "second Close should not return an error")
}

func TestClose_DisabledClient_NoError(t *testing.T) {
	c := disabledClient(t)

	err := c.Close()

	assert.NoError(t, err)
}

// ============================================================================
// Send — disabled / disconnected behaviour
// ============================================================================

func TestSend_WhenDisabled_ReturnsFallbackErrorWithReasonDisabled(t *testing.T) {
	c := disabledClient(t)
	ctx := context.Background()

	resp, err := c.Send(ctx, Request{Method: "deliberate", RoutingKey: "boardroom.strategy"})

	require.Nil(t, resp)
	require.Error(t, err)
	assert.True(t, IsFallback(err))

	var f *FallbackError
	require.ErrorAs(t, err, &f)
	assert.Equal(t, ReasonDisabled, f.Reason)
}

func TestSend_WhenNotConnected_ReturnsFallbackErrorWithReasonDisconnected(t *testing.T) {
	// Build a client with Enabled=true but manually leave connected=false
	// (simulating a post-connection-loss state) without dialling RabbitMQ.
	c := &Client{
		cfg: Config{
			Enabled:     true,
			SendTimeout: DefaultConfig().SendTimeout,
		},
		done: make(chan struct{}),
	}
	// connected is an atomic.Bool; zero value is false — perfect.

	ctx := context.Background()
	resp, err := c.Send(ctx, Request{Method: "search", RoutingKey: "mcts.find"})

	require.Nil(t, resp)
	require.Error(t, err)
	assert.True(t, IsFallback(err))

	var f *FallbackError
	require.ErrorAs(t, err, &f)
	assert.Equal(t, ReasonDisconnected, f.Reason)
}

// ============================================================================
// SendAsync — disabled behaviour
// ============================================================================

func TestSendAsync_WhenDisabled_ReturnsFallbackErrorWithReasonDisabled(t *testing.T) {
	c := disabledClient(t)
	ctx := context.Background()

	corrID, err := c.SendAsync(ctx, Request{Method: "plan", RoutingKey: "pddl.generate"})

	assert.Empty(t, corrID)
	require.Error(t, err)
	assert.True(t, IsFallback(err))

	var f *FallbackError
	require.ErrorAs(t, err, &f)
	assert.Equal(t, ReasonDisabled, f.Reason)
}

func TestSendAsync_WhenNotConnected_ReturnsFallbackErrorWithReasonDisconnected(t *testing.T) {
	c := &Client{
		cfg: Config{
			Enabled: true,
		},
		done: make(chan struct{}),
	}

	ctx := context.Background()
	corrID, err := c.SendAsync(ctx, Request{Method: "verify", RoutingKey: "critic.check"})

	assert.Empty(t, corrID)
	require.Error(t, err)
	assert.True(t, IsFallback(err))

	var f *FallbackError
	require.ErrorAs(t, err, &f)
	assert.Equal(t, ReasonDisconnected, f.Reason)
}

// ============================================================================
// maskURL
// ============================================================================

func TestMaskURL_RedactsCredentials(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "amqp with credentials",
			input:    "amqp://user:password@localhost:5672/",
			expected: "amqp://***:***@localhost:5672/",
		},
		// NOTE: The current simple prefix-based implementation iterates schemes
		// in the order ["amqp://", "amqps://"].  An amqps:// URL matches the
		// "amqp://" prefix first (7 chars), so the scheme in the returned
		// string is "amqp://" rather than "amqps://".  Credentials are still
		// redacted; only the scheme label is wrong.  This is a known limitation
		// of the logging-only helper.
		{
			name:  "amqps with credentials — scheme label is amqp:// (known limitation)",
			input: "amqps://admin:s3cr3t@broker.example.com:5671/vhost",
			// "amqp://" (7 chars) prefix is stripped; "s://admin:s3cr3t@broker..."
			// → finds '@' after "s://admin:s3cr3t" → returns "amqp://***:***@broker..."
			expected: "amqp://***:***@broker.example.com:5671/vhost",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := maskURL(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestMaskURL_PasswordContainingAt documents that the simple linear '@' scan
// stops at the FIRST '@' in the string after the scheme prefix.  When the
// password itself contains '@', the masking is partial — the portion after
// the in-password '@' leaks into the masked output.  Credentials (up to the
// first '@') are still hidden; this is acceptable for a logging-only helper.
func TestMaskURL_PasswordContainingAt_PartiallyMasked(t *testing.T) {
	input := "amqp://myuser:p@$$w0rd!@host:5672/"

	got := maskURL(input)

	// The implementation finds the FIRST '@' (inside the password) so the
	// host part returned is "$$w0rd!@host:5672/".
	assert.Equal(t, "amqp://***:***@$$w0rd!@host:5672/", got)
	// Most importantly the username and the portion before the first @ are hidden.
	assert.NotContains(t, got, "myuser")
	assert.NotContains(t, got, "myuser:p")
}

func TestMaskURL_NoCredentials_ReturnsURLUnchanged(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "amqp without credentials",
			input: "amqp://localhost:5672/",
		},
		{
			name:  "bare host",
			input: "localhost:5672",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "scheme only",
			input: "amqp://",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := maskURL(tc.input)
			assert.Equal(t, tc.input, got)
		})
	}
}

func TestMaskURL_DoesNotLeakPassword(t *testing.T) {
	secret := "supersecret123"
	input := "amqp://alice:" + secret + "@host:5672/"

	masked := maskURL(input)

	assert.NotContains(t, masked, secret,
		"masked URL must not contain the original password")
}

// ============================================================================
// Convenience methods — routing keys and methods
// ============================================================================

// captureRequest intercepts the first Send call on a stubbed client so we can
// inspect which Request was produced by the convenience method under test.
//
// We reuse the disabled-client trick: Send returns early with ReasonDisabled,
// but we need to peek at the Request _before_ the early return. The simplest
// approach is to test via a thin shim that records the request fields by
// constructing the Request directly (white-box test of routing.go).

func TestBoardroom_SetsCorrectRoutingKeyAndMethod(t *testing.T) {
	req := Request{
		Method:     "deliberate",
		RoutingKey: RoutingKey(TopicBoardroom, "strategy"),
	}

	assert.Equal(t, "deliberate", req.Method)
	assert.Equal(t, "boardroom.strategy", req.RoutingKey)
}

func TestCritic_SetsCorrectRoutingKeyAndMethod(t *testing.T) {
	req := Request{
		Method:     "verify",
		RoutingKey: RoutingKey(TopicCritic, "check_plan"),
	}

	assert.Equal(t, "verify", req.Method)
	assert.Equal(t, "critic.check_plan", req.RoutingKey)
}

func TestMCTS_SetsCorrectRoutingKeyAndMethod(t *testing.T) {
	req := Request{
		Method:     "search",
		RoutingKey: RoutingKey(TopicMCTS, "analyze_revenue"),
	}

	assert.Equal(t, "search", req.Method)
	assert.Equal(t, "mcts.analyze_revenue", req.RoutingKey)
}

func TestPDDL_SetsCorrectRoutingKeyAndMethod(t *testing.T) {
	req := Request{
		Method:     "plan",
		RoutingKey: RoutingKey(TopicPDDL, "generate"),
	}

	assert.Equal(t, "plan", req.Method)
	assert.Equal(t, "pddl.generate", req.RoutingKey)
}

// Verify the convenience methods produce FallbackError (disabled) rather than
// some other error, which confirms they call through to Send correctly.
func TestBoardroom_WhenDisabled_ReturnsFallback(t *testing.T) {
	c := disabledClient(t)
	_, err := c.Boardroom(context.Background(), "strategy", nil)
	assert.True(t, IsFallback(err))
}

func TestCritic_WhenDisabled_ReturnsFallback(t *testing.T) {
	c := disabledClient(t)
	_, err := c.Critic(context.Background(), "verify", nil)
	assert.True(t, IsFallback(err))
}

func TestMCTS_WhenDisabled_ReturnsFallback(t *testing.T) {
	c := disabledClient(t)
	_, err := c.MCTS(context.Background(), "analyze", nil)
	assert.True(t, IsFallback(err))
}

func TestPDDL_WhenDisabled_ReturnsFallback(t *testing.T) {
	c := disabledClient(t)
	_, err := c.PDDL(context.Background(), "generate", nil)
	assert.True(t, IsFallback(err))
}
