package vsl

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func TestWithoutAPoolIsAnErrorNotEmpty(t *testing.T) {
	s := &Store{}
	if _, err := s.Snapshots(context.Background()); err == nil {
		t.Fatal("no pool must fail")
	}
	if _, err := s.IGVideo(context.Background()); err == nil {
		t.Fatal("no pool must fail")
	}
}

func TestSnapshotsAndTheIGAssociation(t *testing.T) {
	pool := pgtest.ThrowawayDB(t)
	ctx := context.Background()
	ws := map[string]string{}
	for _, slug := range []string{"launchpad-cohort", "founderos"} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ($1,$1,'u1') RETURNING id::text`, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		ws[slug] = id
	}
	for _, row := range []struct{ id, from, payload string }{
		{"vidA", "2025-08-10", `{"videoId":"vidA","title":"A","plays":5}`},
		{"vidB", "2026-01-01", `{"videoId":"vidB","title":"B","plays":9}`},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO founderos_vsl_snapshots (video_id, date_from, date_to, workspace_id, captured_at, payload)
			VALUES ($1, $2, '2026-09-24', $3, now(), $4)`, row.id, row.from, ws["launchpad-cohort"], row.payload); err != nil {
			t.Fatal(err)
		}
	}
	s := &Store{Pool: pool, Workspaces: ws}
	snaps, err := s.Snapshots(ctx)
	if err != nil || len(snaps) != 2 {
		t.Fatalf("snaps = %v %v", snaps, err)
	}
	var first struct{ VideoID string }
	_ = json.Unmarshal(snaps[0], &first)
	if first.VideoID != "vidB" {
		t.Fatalf("newest period first, got %s", first.VideoID)
	}
	if v, err := s.IGVideo(ctx); err != nil || v != nil {
		t.Fatalf("unset association = %v %v", v, err)
	}
	if err := s.SetIGVideo(ctx, "nope"); !errors.Is(err, ErrUnknownVideo) {
		t.Fatalf("unknown video = %v", err)
	}
	if err := s.SetIGVideo(ctx, "vidA"); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.IGVideo(ctx); v == nil || *v != "vidA" {
		t.Fatalf("association = %v", v)
	}
	if err := s.SetIGVideo(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.IGVideo(ctx); v != nil {
		t.Fatalf("cleared association = %v", *v)
	}
}
