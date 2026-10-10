package security

import (
	"strings"
	"testing"
)

func TestValidateProductionSecretsRejectsPlaceholders(t *testing.T) {
	valid := strings.Repeat("s", 32)

	err := ValidateProductionSecrets("CHANGE_ME_CHANGE_ME_CHANGE_ME_1234", valid, valid)
	if err == nil {
		t.Fatal("expected placeholder secret to fail validation")
	}
	if !strings.Contains(err.Error(), "change_me") {
		t.Fatalf("expected change_me placeholder error, got %v", err)
	}
}
