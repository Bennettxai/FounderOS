package signal

import "testing"

func TestFastClassifier_GenreDirect(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("Create a proposal for the new project", "", true, false)
	if env.Genre != GenreDirect {
		t.Errorf("expected DIRECT, got %s", env.Genre)
	}
	if env.DocType != "proposal" {
		t.Errorf("expected proposal doctype, got %q", env.DocType)
	}
}

func TestFastClassifier_GenreInform(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("What is the status of project Alpha?", "", true, false)
	if env.Genre != GenreInform {
		t.Errorf("expected INFORM, got %s", env.Genre)
	}
}

func TestFastClassifier_GenreDecide(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("Should we choose between vendor A or vendor B?", "", false, false)
	if env.Genre != GenreDecide {
		t.Errorf("expected DECIDE, got %s", env.Genre)
	}
}

func TestFastClassifier_GenreCommit(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("I will deliver this by Friday, let's plan the timeline", "", false, false)
	if env.Genre != GenreCommit {
		t.Errorf("expected COMMIT, got %s", env.Genre)
	}
}

func TestFastClassifier_GenreExpress(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("I feel frustrated with the slow progress", "", false, false)
	if env.Genre != GenreExpress {
		t.Errorf("expected EXPRESS, got %s", env.Genre)
	}
}

func TestFastClassifier_FocusModeOverride(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("Help me with something", "write", false, false)
	if env.Mode != ModeBuild {
		t.Errorf("expected BUILD for write focus mode, got %s", env.Mode)
	}
}

func TestFastClassifier_DocTypeProposal(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("Draft a proposal for Acme Corp integration", "", false, true)
	if env.DocType != "proposal" {
		t.Errorf("expected proposal, got %q", env.DocType)
	}
	if env.Confidence < 0.6 {
		t.Errorf("expected confidence >= 0.6 with client, got %.2f", env.Confidence)
	}
}

func TestFastClassifier_DocTypeSOP(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("Create an SOP for the onboarding process", "", false, false)
	if env.DocType != "sop" {
		t.Errorf("expected sop, got %q", env.DocType)
	}
}

func TestFastClassifier_DocTypeReport(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("Generate a quarterly report on sales performance", "", false, false)
	if env.DocType != "report" {
		t.Errorf("expected report, got %q", env.DocType)
	}
}

func TestFastClassifier_WeightShortMessage(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("ok", "", false, false)
	if env.Weight > 0.3 {
		t.Errorf("expected low weight for short message, got %.2f", env.Weight)
	}
}

func TestFastClassifier_WeightDetailedMessage(t *testing.T) {
	fc := NewFastClassifier()
	msg := "I need a comprehensive proposal for the Q3 expansion project that covers the market analysis, competitive landscape, implementation timeline, resource requirements, and ROI projections for the board presentation next week"
	env := fc.Classify(msg, "", true, true)
	if env.Weight < 0.7 {
		t.Errorf("expected high weight for detailed message, got %.2f", env.Weight)
	}
}

func TestFastClassifier_ConfidenceWithContext(t *testing.T) {
	fc := NewFastClassifier()
	// With project + client + doctype = high confidence
	env := fc.Classify("Create a proposal for the client", "", true, true)
	if env.Confidence < 0.8 {
		t.Errorf("expected high confidence with full context, got %.2f", env.Confidence)
	}
}

func TestFastClassifier_ModeAnalyze(t *testing.T) {
	fc := NewFastClassifier()
	env := fc.Classify("Analyze the performance metrics for Q2", "analyze", false, false)
	if env.Mode != ModeAnalyze {
		t.Errorf("expected ANALYZE, got %s", env.Mode)
	}
}

func TestFastClassifier_DefaultGenre(t *testing.T) {
	fc := NewFastClassifier()
	// Ambiguous message with no strong pattern
	env := fc.Classify("hello there", "", false, false)
	if env.Genre != GenreInform {
		t.Errorf("expected INFORM as default, got %s", env.Genre)
	}
}
