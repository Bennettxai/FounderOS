package api

import (
	"context"
	"testing"
)

// A malformed engine URL (stray whitespace in OE_*_URL) is an error on the
// brain-pages read, never a nil request handed to client.Do (a panic).
func TestBrainPagesWithAMalformedEngineURLIsAnError(t *testing.T) {
	for _, env := range []string{"OE_MACBOOK_URL", "OE_MINI_URL", "OE_HUB_URL"} {
		t.Setenv(env, "http://127.0.0.1:4211 \n")
		t.Setenv(env[:len(env)-3]+"KEY", "k")
	}
	if _, err := brainPages(context.Background(), &Deps{}); err == nil {
		t.Fatal("want an error")
	}
}
