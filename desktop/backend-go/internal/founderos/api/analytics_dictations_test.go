package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

func wisprPush(device string, state connectors.State, meta map[string]any) devicepush.Payload {
	return devicepush.Payload{Device: device, Label: device, Statuses: []connectors.Status{
		{ID: "wispr", Name: "Wispr Flow", Kind: connectors.KindLocal, State: state, Detail: "d", Meta: meta},
	}}
}

// Prod's Dictations tile is the Wispr lifetime total (wispr.meta.dictations
// while connected). Here it comes from the host's fresh push, else another
// Mac's; anything less than a connected reading with a count is unknown.
func TestDictationsTileReadsThePushedWisprTotal(t *testing.T) {
	ctx := context.Background()
	if v := wisprDictations(ctx, nil); v != nil {
		t.Fatalf("no receiver = unknown, got %v", *v)
	}
	r := devicepush.NewReceiver(devicepush.NewMemStore())
	r.Host = "mini"
	if v := wisprDictations(ctx, r); v != nil {
		t.Fatalf("no push = unknown, got %v", *v)
	}
	must := func(p devicepush.Payload) {
		t.Helper()
		if err := r.Accept(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	must(wisprPush("macbook", connectors.StateConnected, map[string]any{"dictations": 4944}))
	if v := wisprDictations(ctx, r); v == nil || *v != 4944 {
		t.Fatalf("another Mac answers while the host has no reading: %v", v)
	}
	// a push that went through Postgres comes back as a JSON number
	var meta map[string]any
	_ = json.Unmarshal([]byte(`{"dictations": 5120}`), &meta)
	must(wisprPush("mini", connectors.StateConnected, meta))
	if v := wisprDictations(ctx, r); v == nil || *v != 5120 {
		t.Fatalf("the host's total wins: %v", v)
	}
	must(wisprPush("mini", connectors.StateConnected, nil))
	if v := wisprDictations(ctx, r); v != nil {
		t.Fatalf("a reading without a total is unknown, never 0: %v", *v)
	}
	must(wisprPush("mini", connectors.StateError, map[string]any{"dictations": 7}))
	if v := wisprDictations(ctx, r); v != nil {
		t.Fatalf("an errored reading is unknown: %v", *v)
	}
	must(wisprPush("mini", connectors.StateConnected, map[string]any{"dictations": 7}))
	r.StaleAfter = -time.Second
	if v := wisprDictations(ctx, r); v != nil {
		t.Fatalf("a stale push is unknown: %v", *v)
	}
}

func TestDictationsTileCarriesTheRead(t *testing.T) {
	tile := func(reads analyticsReads) (v *float64, source string) {
		for _, in := range operatingInputs(0, nil, 0, 0, reads, nil, "") {
			if in.ID == "dictations" {
				return in.Value, in.Source
			}
		}
		t.Fatal("no dictations tile")
		return nil, ""
	}
	if v, src := tile(analyticsReads{Dictations: fptr(4944)}); v == nil || *v != 4944 || src != "Wispr Flow" {
		t.Fatalf("tile = %v %q", v, src)
	}
	if v, _ := tile(analyticsReads{}); v != nil {
		t.Fatalf("an unread total stays pending: %v", *v)
	}
}
