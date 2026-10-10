package email

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Outbound mail guard, ported from FounderOS v1 lib/mail-guard.mjs.
//
// an earlier incident (incident 2026-08-21): agent-generated mail reached a paying customer
// and three invented addresses without anyone approving a recipient. Mail to
// the operator's own inboxes stays allowed so internal alerting keeps working;
// mail to anyone else is refused. The overrides are env-only and per call,
// never defaults:
//
//	MAIL_ALLOW_EXTERNAL=1        allow recipients outside the operator's inboxes
//	MAIL_ALLOW_SYSTEM_FROM=1 allow sending as the OS system address (os@founderos.local)
//
// This is independent of, and checked before, the bridge's FOUNDEROS_WRITES guard.

// InternalRecipients are the operator's own mailboxes.
var InternalRecipients = []string{
	"os@founderos.local",
	"alex@founderos.local",
	"alex@personal.example",
	"alex@launchpadcohort.example",
	"alex@vantage.example",
	"admin@founderos.local",
}

// BannedFrom: the operator, 2026-08-18: "i dont want it sent from the OS
// system address going forward." Client mail goes out from a venture inbox.
var BannedFrom = []string{"os@founderos.local"}

// OutboundMail is a proposed send. To, Cc and Bcc accept "a@x, b@y" and
// "Name <addr>" forms.
type OutboundMail struct {
	From, To, Cc, Bcc string
}

func norm(a string) string { return strings.ToLower(strings.TrimSpace(a)) }

var angle = regexp.MustCompile(`<([^>]+)>`)

// ParseAddresses splits a comma list into normalized addresses. A display
// name can never smuggle a recipient past the check: "Name <addr>" yields addr.
func ParseAddresses(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		a := norm(part)
		if a == "" {
			continue
		}
		if m := angle.FindStringSubmatch(a); m != nil {
			a = strings.TrimSpace(m[1])
		}
		out = append(out, a)
	}
	return out
}

// IsInternal reports whether address is one of the operator's own mailboxes.
func IsInternal(address string) bool { return slices.Contains(InternalRecipients, norm(address)) }

// CheckOutboundMail returns the normalized recipient list when the send is
// allowed, or the reason it is refused. env reads the per-call overrides.
func CheckOutboundMail(m OutboundMail, env func(string) string) ([]string, error) {
	var recipients []string
	recipients = append(recipients, ParseAddresses(m.To)...)
	recipients = append(recipients, ParseAddresses(m.Cc)...)
	recipients = append(recipients, ParseAddresses(m.Bcc)...)
	if len(recipients) == 0 {
		return nil, errors.New("no recipient address")
	}

	if slices.Contains(BannedFrom, norm(m.From)) && env("MAIL_ALLOW_SYSTEM_FROM") != "1" {
		return nil, fmt.Errorf("blocked by mail-guard: sending as %s was disallowed by the operator on 2026-08-18. "+
			"Use --from alex@vantage.example (client mail) or set MAIL_ALLOW_SYSTEM_FROM=1 for this one call.", norm(m.From))
	}

	var external []string
	for _, a := range recipients {
		if !IsInternal(a) {
			external = append(external, a)
		}
	}
	if len(external) > 0 && env("MAIL_ALLOW_EXTERNAL") != "1" {
		return nil, fmt.Errorf("blocked by mail-guard: %d external recipient(s) — %s. "+
			"Outbound mail to addresses outside the operator's own inboxes needs a human to approve the recipient. "+
			"Set MAIL_ALLOW_EXTERNAL=1 for this one call if that approval exists.", len(external), strings.Join(external, ", "))
	}
	return recipients, nil
}
