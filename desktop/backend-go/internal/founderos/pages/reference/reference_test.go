package reference

import (
	"context"
	"errors"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/pages/catalogdb"
)

const insert = `INSERT INTO founderos_domains (id, workspace_id, number, title, color, items) VALUES ($1, $2, $3, $4, '#fafafa', $5)`

func TestLoadReturnsTheWorkspaceDomainsInNumberOrder(t *testing.T) {
	pool := catalogdb.TestDB(t)
	ctx := context.Background()
	ws := catalogdb.Workspace(t, pool, "founderos")
	other := catalogdb.Workspace(t, pool, "vantage")
	for _, r := range []struct {
		id, ws string
		n      int
		title  string
		items  string
	}{
		{"brm-2", ws, 2, "Email Operations", `["Four IMAP inboxes","Digest (planned)"]`},
		{"brm-1", ws, 1, "Command & Memory", `["G-Brain (gbrain CLI)","brain-store markdown"]`},
		{"brm-x", other, 0, "Elsewhere", `["x"]`},
	} {
		if _, err := pool.Exec(ctx, insert, r.id, r.ws, r.n, r.title, r.items); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Load(ctx, pool, "founderos")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "brm-1" || got[1].ID != "brm-2" {
		t.Fatalf("got %+v, want brm-1 then brm-2 from founderos only", got)
	}
	d := got[0]
	if d.Number != 1 || d.Title != "Command & Memory" || d.Color != "#fafafa" {
		t.Fatalf("fields not mapped: %+v", d)
	}
	// G-Brain is gone: the knowledge core is the Optimal Engine in v2.
	if len(d.Items) != 2 || d.Items[0] != "Optimal Engine" || d.Items[1] != "brain-store markdown" {
		t.Fatalf("items = %v", d.Items)
	}
}

func TestLoadWithoutTheWorkspaceIsAnError(t *testing.T) {
	pool := catalogdb.TestDB(t)
	if _, err := Load(context.Background(), pool, "founderos"); !errors.Is(err, ErrNoWorkspace) {
		t.Fatalf("err = %v, want ErrNoWorkspace", err)
	}
}

func TestValidateMirrorsDomainSchema(t *testing.T) {
	ok := Domain{ID: "a", Number: 1, Title: "t", Color: "#fff", Items: []string{}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*Domain){
		"no title": func(d *Domain) { d.Title = "" },
		"no color": func(d *Domain) { d.Color = "" },
		"no id":    func(d *Domain) { d.ID = "" },
	} {
		d := ok
		mut(&d)
		if d.Validate() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
