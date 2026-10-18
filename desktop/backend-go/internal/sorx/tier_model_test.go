package sorx

import "testing"

func TestTierToModel(t *testing.T) {
	tests := []struct {
		tier SkillTier
		want string
	}{
		{TierDeterministic, ""},
		{TierStructuredAI, "claude-haiku-4-5"},
		{TierReasoningAI, "claude-sonnet-5"},
		{TierGenerativeAI, "claude-opus-5"},
		{SkillTier(99), "claude-sonnet-5"},
	}
	for _, tt := range tests {
		if got := TierToModel(tt.tier); got != tt.want {
			t.Errorf("TierToModel(%d) = %q, want %q", tt.tier, got, tt.want)
		}
	}
}
