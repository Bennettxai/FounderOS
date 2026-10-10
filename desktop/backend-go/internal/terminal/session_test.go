package terminal

import (
	"testing"
	"time"
)

func TestDefaultSessionSecurityConfig(t *testing.T) {
	config := DefaultSessionSecurityConfig()

	if config.MaxSessionDuration != 8*time.Hour {
		t.Errorf("Expected MaxSessionDuration=8h, got %v", config.MaxSessionDuration)
	}
	if config.IdleTimeout != 30*time.Minute {
		t.Errorf("Expected IdleTimeout=30m, got %v", config.IdleTimeout)
	}
	if !config.EnableIPBinding {
		t.Error("Expected EnableIPBinding=true")
	}
	if config.AllowIPMigration {
		t.Error("Expected AllowIPMigration=false")
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{"zero expiration", time.Time{}, false},
		{"future expiration", time.Now().Add(time.Hour), false},
		{"past expiration", time.Now().Add(-time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ExpiresAt: tt.expiresAt}
			if got := session.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSession_IsIdle(t *testing.T) {
	tests := []struct {
		name         string
		lastActivity time.Time
		timeout      time.Duration
		expected     bool
	}{
		{"recent activity", time.Now(), 30 * time.Minute, false},
		{"old activity", time.Now().Add(-time.Hour), 30 * time.Minute, true},
		{"just before timeout", time.Now().Add(-29 * time.Minute), 30 * time.Minute, false},
		{"past timeout", time.Now().Add(-31 * time.Minute), 30 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{LastActivity: tt.lastActivity}
			if got := session.IsIdle(tt.timeout); got != tt.expected {
				t.Errorf("IsIdle() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSession_ValidateIP(t *testing.T) {
	tests := []struct {
		name       string
		sessionIP  string
		clientIP   string
		config     *SessionSecurityConfig
		wantValid  bool
	}{
		{
			name:      "ip binding disabled",
			sessionIP: "192.168.1.100",
			clientIP:  "10.0.0.1",
			config:    &SessionSecurityConfig{EnableIPBinding: false},
			wantValid: true,
		},
		{
			name:      "exact ip match",
			sessionIP: "192.168.1.100",
			clientIP:  "192.168.1.100",
			config:    &SessionSecurityConfig{EnableIPBinding: true},
			wantValid: true,
		},
		{
			name:      "ip mismatch - strict mode",
			sessionIP: "192.168.1.100",
			clientIP:  "192.168.1.200",
			config:    &SessionSecurityConfig{EnableIPBinding: true, AllowIPMigration: false},
			wantValid: false,
		},
		{
			name:      "subnet match - migration enabled",
			sessionIP: "192.168.1.100",
			clientIP:  "192.168.1.200",
			config:    &SessionSecurityConfig{EnableIPBinding: true, AllowIPMigration: true},
			wantValid: true,
		},
		{
			name:      "different subnet - migration enabled",
			sessionIP: "192.168.1.100",
			clientIP:  "192.168.2.100",
			config:    &SessionSecurityConfig{EnableIPBinding: true, AllowIPMigration: true},
			wantValid: false,
		},
		{
			name:      "completely different network",
			sessionIP: "192.168.1.100",
			clientIP:  "10.0.0.1",
			config:    &SessionSecurityConfig{EnableIPBinding: true, AllowIPMigration: true},
			wantValid: false,
		},
		{
			name:      "empty session IP",
			sessionIP: "",
			clientIP:  "192.168.1.100",
			config:    &SessionSecurityConfig{EnableIPBinding: true},
			wantValid: true, // No IP to compare against
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				ClientIP:     tt.sessionIP,
				ClientSubnet: extractSubnet(tt.sessionIP),
			}
			got, _ := session.ValidateIP(tt.clientIP, tt.config)
			if got != tt.wantValid {
				t.Errorf("ValidateIP() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestExtractSubnet(t *testing.T) {
	tests := []struct {
		ip       string
		expected string
	}{
		{"192.168.1.100", "192.168.1"},
		{"10.0.0.1", "10.0.0"},
		{"172.16.0.1", "172.16.0"},
		{"127.0.0.1", "127.0.0"},
		{"2001:db8:85a3:0000:0000:8a2e:0370:7334", "2001:db8:85a3:0000:"},
		{"::1", "::1"},
		{"short", "short"},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			got := extractSubnet(tt.ip)
			if got != tt.expected {
				t.Errorf("extractSubnet(%s) = %s, want %s", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestSplitIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected int // Number of parts
	}{
		{"192.168.1.100", 4},
		{"10.0.0.1", 4},
		{"2001:db8::1", 1}, // No dots in IPv6
		{"localhost", 1},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			parts := splitIP(tt.ip)
			if len(parts) != tt.expected {
				t.Errorf("splitIP(%s) = %d parts, want %d", tt.ip, len(parts), tt.expected)
			}
		})
	}
}

func TestSession_IsContainerized(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		expected    bool
	}{
		{"with container", "abc123", true},
		{"without container", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{ContainerID: tt.containerID}
			if got := session.IsContainerized(); got != tt.expected {
				t.Errorf("IsContainerized() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSession_SecurityFields(t *testing.T) {
	// Test that security fields are properly initialized
	config := DefaultSessionSecurityConfig()
	now := time.Now()
	expiresAt := now.Add(config.MaxSessionDuration)

	session := &Session{
		ID:           "test-session-id",
		UserID:       "test-user-id",
		CreatedAt:    now,
		LastActivity: now,
		ClientIP:     "192.168.1.100",
		ClientSubnet: "192.168.1",
		ExpiresAt:    expiresAt,
		Status:       StatusActive,
	}

	// Verify fields
	if session.ClientIP != "192.168.1.100" {
		t.Errorf("ClientIP = %s, want 192.168.1.100", session.ClientIP)
	}
	if session.ClientSubnet != "192.168.1" {
		t.Errorf("ClientSubnet = %s, want 192.168.1", session.ClientSubnet)
	}
	if session.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}
	if !session.ExpiresAt.After(now) {
		t.Error("ExpiresAt should be in the future")
	}
}

func BenchmarkValidateIP(b *testing.B) {
	session := &Session{
		ClientIP:     "192.168.1.100",
		ClientSubnet: "192.168.1",
	}
	config := &SessionSecurityConfig{
		EnableIPBinding:  true,
		AllowIPMigration: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.ValidateIP("192.168.1.200", config)
	}
}

func BenchmarkExtractSubnet(b *testing.B) {
	ip := "192.168.1.100"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractSubnet(ip)
	}
}

func BenchmarkIsExpired(b *testing.B) {
	session := &Session{
		ExpiresAt: time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.IsExpired()
	}
}
