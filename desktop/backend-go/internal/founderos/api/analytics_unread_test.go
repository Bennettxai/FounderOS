package api

import (
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
)

// The unread tile counts only inboxes that answered, and its label says how
// many that was: a sum over 3 of 4 inboxes is not "all inboxes".
func TestUnreadTileSaysHowManyInboxesItCounted(t *testing.T) {
	n := func(v int) *int { return &v }
	cases := []struct {
		counts []email.InboxUnread
		value  *float64
		label  string
	}{
		{[]email.InboxUnread{{Unread: n(2)}, {Unread: n(3)}}, fp(5), "Unread · all inboxes"},
		{[]email.InboxUnread{{Unread: n(2)}, {Unread: nil}, {Unread: n(1)}}, fp(3), "Unread · 2 of 3 inboxes"},
		{[]email.InboxUnread{{Unread: nil}}, nil, "Unread · all inboxes"},
		{nil, nil, "Unread · all inboxes"},
	}
	for i, c := range cases {
		v, label := unreadTile(c.counts)
		if (v == nil) != (c.value == nil) || (v != nil && *v != *c.value) || label != c.label {
			t.Errorf("case %d: got %v %q, want %v %q", i, v, label, c.value, c.label)
		}
	}
}

func fp(v float64) *float64 { return &v }
