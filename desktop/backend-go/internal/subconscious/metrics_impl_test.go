package subconscious

import (
	"context"
	"testing"
	"time"
)

func TestSlidingWindowBasicRate(t *testing.T) {
	sw := newSlidingWindow(5 * time.Minute)
	// Empty window returns 0
	if r := sw.rate(); r != 0 {
		t.Fatalf("expected 0 for empty window, got %f", r)
	}

	// 3 successes, 1 failure = 0.75
	sw.record(true)
	sw.record(true)
	sw.record(true)
	sw.record(false)
	if r := sw.rate(); r != 0.75 {
		t.Fatalf("expected 0.75, got %f", r)
	}
}

func TestSlidingWindowPrune(t *testing.T) {
	sw := newSlidingWindow(1 * time.Millisecond)

	sw.record(true)
	sw.record(true)

	// Wait for window to expire
	time.Sleep(5 * time.Millisecond)

	// New record triggers prune
	sw.record(false)

	// Only the new record should remain
	if c := sw.count(); c != 1 {
		t.Fatalf("expected 1 entry after prune, got %d", c)
	}
	if r := sw.rate(); r != 0 {
		t.Fatalf("expected 0 rate (only failure), got %f", r)
	}
}

func TestInMemoryReEncoding(t *testing.T) {
	m := NewInMemoryReEncoding(30 * time.Minute)

	if m.Name() != "re_encoding_frequency" {
		t.Fatalf("unexpected name: %s", m.Name())
	}

	// Record 2 re-encodings and 8 normal
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := m.Increment(ctx, nil, "rephrase"); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 8; i++ {
		m.RecordNormal()
	}

	freq, err := m.Frequency(ctx, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// 2 re-encodings out of 10 total = 0.2
	if freq != 0.2 {
		t.Fatalf("expected 0.2 frequency, got %f", freq)
	}

	mv, err := m.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mv.Name != "re_encoding_frequency" {
		t.Fatalf("unexpected metric name: %s", mv.Name)
	}
}

func TestInMemorySignalBounce(t *testing.T) {
	m := NewInMemorySignalBounce(30 * time.Minute)

	if m.Name() != "signal_bounce_rate" {
		t.Fatalf("unexpected name: %s", m.Name())
	}

	ctx := context.Background()
	// 1 bounce, 9 normal
	if err := m.RecordBounce(ctx, nil, "", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		m.RecordNoBounce()
	}

	rate, err := m.Rate(ctx, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 0.1 {
		t.Fatalf("expected 0.1 bounce rate, got %f", rate)
	}
}

func TestInMemoryGenreRecognition(t *testing.T) {
	m := NewInMemoryGenreRecognition(30 * time.Minute)

	if m.Name() != "genre_recognition_rate" {
		t.Fatalf("unexpected name: %s", m.Name())
	}

	ctx := context.Background()
	// 8 correct, 2 incorrect
	for i := 0; i < 8; i++ {
		if err := m.RecordRecognition(ctx, nil, true); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := m.RecordRecognition(ctx, nil, false); err != nil {
			t.Fatal(err)
		}
	}

	rate, err := m.Rate(ctx, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 0.8 {
		t.Fatalf("expected 0.8 recognition rate, got %f", rate)
	}
}

func TestInMemoryFeedbackClosure(t *testing.T) {
	m := NewInMemoryFeedbackClosure(30 * time.Minute)

	if m.Name() != "feedback_closure_rate" {
		t.Fatalf("unexpected name: %s", m.Name())
	}

	ctx := context.Background()
	// 7 closed, 3 open
	for i := 0; i < 7; i++ {
		if err := m.RecordFeedback(ctx, nil, true); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := m.RecordFeedback(ctx, nil, false); err != nil {
			t.Fatal(err)
		}
	}

	rate, err := m.ClosureRate(ctx, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 0.7 {
		t.Fatalf("expected 0.7 closure rate, got %f", rate)
	}
}
