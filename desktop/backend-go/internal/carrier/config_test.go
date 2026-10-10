package carrier

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// DefaultConfig
// ============================================================================

func TestDefaultConfig_ReturnsExpectedDefaults(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.URL)
	assert.Equal(t, "sorx.carrier", cfg.Exchange)
	assert.Equal(t, 60*time.Second, cfg.SendTimeout)
	assert.Equal(t, 10, cfg.Prefetch)
	assert.False(t, cfg.Enabled, "carrier should be disabled by default")
	assert.Empty(t, cfg.OSInstanceID, "OSInstanceID should be empty by default")
}

// ============================================================================
// ConfigFromEnv
// ============================================================================

func TestConfigFromEnv_OverridesURL(t *testing.T) {
	t.Setenv("CARRIER_AMQP_URL", "amqp://user:pass@broker:5672/")
	t.Setenv("CARRIER_ENABLED", "false")

	cfg, err := ConfigFromEnv(DefaultConfig())

	require.NoError(t, err)
	assert.Equal(t, "amqp://user:pass@broker:5672/", cfg.URL)
}

func TestConfigFromEnv_OverridesOSInstanceID(t *testing.T) {
	t.Setenv("OS_INSTANCE_ID", "bos-prod-1")
	t.Setenv("CARRIER_ENABLED", "true")

	cfg, err := ConfigFromEnv(DefaultConfig())

	require.NoError(t, err)
	assert.Equal(t, "bos-prod-1", cfg.OSInstanceID)
}

func TestConfigFromEnv_OverridesEnabled_True(t *testing.T) {
	t.Setenv("CARRIER_ENABLED", "true")
	t.Setenv("OS_INSTANCE_ID", "bos-test-42") // required when enabled

	cfg, err := ConfigFromEnv(DefaultConfig())

	require.NoError(t, err)
	assert.True(t, cfg.Enabled)
}

func TestConfigFromEnv_OverridesEnabled_False(t *testing.T) {
	t.Setenv("CARRIER_ENABLED", "false")

	cfg, err := ConfigFromEnv(DefaultConfig())

	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
}

func TestConfigFromEnv_OverridesSendTimeout(t *testing.T) {
	t.Setenv("CARRIER_SEND_TIMEOUT", "30s")
	t.Setenv("CARRIER_ENABLED", "false")

	cfg, err := ConfigFromEnv(DefaultConfig())

	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.SendTimeout)
}

func TestConfigFromEnv_OverridesPrefetch(t *testing.T) {
	t.Setenv("CARRIER_PREFETCH", "20")
	t.Setenv("CARRIER_ENABLED", "false")

	cfg, err := ConfigFromEnv(DefaultConfig())

	require.NoError(t, err)
	assert.Equal(t, 20, cfg.Prefetch)
}

func TestConfigFromEnv_PreservesBaseWhenNoEnvVarsSet(t *testing.T) {
	// No env vars set — all keys unset so t.Setenv is not needed, but we
	// clear any vars that might have leaked from sibling tests.
	t.Setenv("CARRIER_AMQP_URL", "")
	t.Setenv("OS_INSTANCE_ID", "")
	t.Setenv("CARRIER_ENABLED", "")
	t.Setenv("CARRIER_SEND_TIMEOUT", "")
	t.Setenv("CARRIER_PREFETCH", "")

	base := Config{
		URL:          "amqp://custom:custom@myhost:5672/",
		Exchange:     "my.exchange",
		OSInstanceID: "my-instance",
		SendTimeout:  45 * time.Second,
		Prefetch:     5,
		Enabled:      false,
	}

	cfg, err := ConfigFromEnv(base)

	require.NoError(t, err)
	assert.Equal(t, base.URL, cfg.URL)
	assert.Equal(t, base.Exchange, cfg.Exchange)
	assert.Equal(t, base.OSInstanceID, cfg.OSInstanceID)
	assert.Equal(t, base.SendTimeout, cfg.SendTimeout)
	assert.Equal(t, base.Prefetch, cfg.Prefetch)
	assert.Equal(t, base.Enabled, cfg.Enabled)
}

// ============================================================================
// ConfigFromEnv — validation errors
// ============================================================================

func TestConfigFromEnv_ReturnsError_WhenEnabledWithoutInstanceID(t *testing.T) {
	t.Setenv("CARRIER_ENABLED", "true")
	t.Setenv("OS_INSTANCE_ID", "") // explicitly absent

	base := DefaultConfig() // OSInstanceID is empty
	_, err := ConfigFromEnv(base)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "OS_INSTANCE_ID")
}

func TestConfigFromEnv_ReturnsError_InvalidCarrierEnabled(t *testing.T) {
	t.Setenv("CARRIER_ENABLED", "not-a-bool")

	_, err := ConfigFromEnv(DefaultConfig())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CARRIER_ENABLED")
	assert.Contains(t, err.Error(), "not-a-bool")
}

func TestConfigFromEnv_ReturnsError_InvalidSendTimeout(t *testing.T) {
	t.Setenv("CARRIER_SEND_TIMEOUT", "notaduration")
	t.Setenv("CARRIER_ENABLED", "false")

	_, err := ConfigFromEnv(DefaultConfig())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CARRIER_SEND_TIMEOUT")
	assert.Contains(t, err.Error(), "notaduration")
}

func TestConfigFromEnv_ReturnsError_InvalidPrefetch_NonInteger(t *testing.T) {
	t.Setenv("CARRIER_PREFETCH", "banana")
	t.Setenv("CARRIER_ENABLED", "false")

	_, err := ConfigFromEnv(DefaultConfig())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CARRIER_PREFETCH")
}

func TestConfigFromEnv_ReturnsError_InvalidPrefetch_Zero(t *testing.T) {
	t.Setenv("CARRIER_PREFETCH", "0")
	t.Setenv("CARRIER_ENABLED", "false")

	_, err := ConfigFromEnv(DefaultConfig())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CARRIER_PREFETCH")
}

func TestConfigFromEnv_ReturnsError_InvalidPrefetch_Negative(t *testing.T) {
	t.Setenv("CARRIER_PREFETCH", "-5")
	t.Setenv("CARRIER_ENABLED", "false")

	_, err := ConfigFromEnv(DefaultConfig())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CARRIER_PREFETCH")
}

// ============================================================================
// replyQueueName
// ============================================================================

func TestReplyQueueName_ReturnsExpectedFormat(t *testing.T) {
	tests := []struct {
		osInstanceID string
		expected     string
	}{
		{"bos-prod-1", "sorx.responses.bos-prod-1"},
		{"instance-42", "sorx.responses.instance-42"},
		{"", "sorx.responses."},
	}

	for _, tc := range tests {
		t.Run(tc.osInstanceID, func(t *testing.T) {
			got := replyQueueName(tc.osInstanceID)
			assert.Equal(t, tc.expected, got)
		})
	}
}
