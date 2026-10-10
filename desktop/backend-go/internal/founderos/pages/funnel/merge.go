package funnel

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
)

// ---- Trakyo attribution (lib/funnel-trakyo.ts) -----------------------------

var spaces = regexp.MustCompile(`\s+`)

func normSpace(s string) string {
	return spaces.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), " ")
}

// TrakyoAcquisition is the rim wedge a Trakyo first touch justifies.
func TrakyoAcquisition(e trakyo.Event) string {
	switch e.SourceType {
	case "youtube":
		return "youtube"
	case "meta_ad":
		return "instagram"
	}
	src := ""
	if e.SourceName != nil {
		src = *e.SourceName
	}
	if m := MatchAcquisition(src + " " + e.Label); m != "" {
		return m
	}
	return "word_of_mouth"
}

// MergeTrakyoTouches swaps a journey's first touch for the Trakyo-attributed
// content touch, matched by any normalized name or email the two share.
func MergeTrakyoTouches(journeys []Journey, events []trakyo.Event) []Journey {
	if len(events) == 0 {
		return journeys
	}
	byID := map[string]trakyo.Event{}
	for _, e := range events {
		ids := append(append([]string{e.Lead}, e.Names...), e.Emails...)
		for _, id := range ids {
			if k := normSpace(id); k != "" {
				if _, ok := byID[k]; !ok {
					byID[k] = e
				}
			}
		}
	}
	out := make([]Journey, 0, len(journeys))
	for _, j := range journeys {
		var ev *trakyo.Event
		for _, v := range []*string{&j.Name, j.Person, j.Email} {
			if v == nil || *v == "" {
				continue
			}
			if e, ok := byID[normSpace(*v)]; ok {
				ev = &e
				break
			}
		}
		if ev == nil || len(j.Touches) == 0 {
			out = append(out, j)
			continue
		}
		touches := append([]Touch(nil), j.Touches...)
		first := touches[0]
		first.Channel, first.Label, first.Source, first.At, first.Acquisition = ev.Channel, ev.Label, "trakyo", ev.At, TrakyoAcquisition(*ev)
		touches[0] = first
		j.Touches = touches
		out = append(out, j)
	}
	return out
}

// ---- Stripe wins (lib/funnel-stripe.ts) -------------------------------------

// enUS formats like Number.toLocaleString('en-US', {maximumFractionDigits}).
func enUS(v float64, maxFrac int) string {
	p := math.Pow(10, float64(maxFrac))
	v = math.Round(v*p) / p
	neg := v < 0
	s := strconv.FormatFloat(math.Abs(v), 'f', -1, 64)
	intPart, frac, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if frac != "" {
		out += "." + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}

func winSource(w stripe.Win) string {
	if w.Source == "paykit" {
		return "paykit"
	}
	return "stripe"
}

func winLabel(w stripe.Win) string {
	amount := "$" + enUS(w.AmountUSD, 2)
	if winSource(w) == "paykit" {
		// AmountUSD is lifetime spend; say so whenever it spans several payments
		if w.Payments > 1 {
			return "PayKit " + amount + " lifetime · " + strconv.Itoa(w.Payments) + " payments"
		}
		return "PayKit payment " + amount
	}
	l := "Stripe payment " + amount
	if w.Product != nil {
		l += " — " + *w.Product
	}
	return l
}

// MergeStripeWins folds settled payments into the funnel: a win matched by
// email or name converts that journey (amount = the larger of deal value and
// money collected); unmatched wins become standalone converted journeys,
// grouped per customer email.
func MergeStripeWins(journeys []Journey, wins []stripe.Win) []Journey {
	if len(wins) == 0 {
		return journeys
	}
	byID := map[string]string{}
	for _, j := range journeys {
		for _, v := range []*string{j.Email, &j.Name, j.Person} {
			if v == nil {
				continue
			}
			if k := norm(*v); k != "" {
				if _, ok := byID[k]; !ok {
					byID[k] = j.ID
				}
			}
		}
	}
	matched := map[string][]stripe.Win{}
	var orphanOrder []string
	orphans := map[string][]stripe.Win{}
	for _, w := range wins {
		jid := ""
		for _, v := range []*string{w.Email, w.Name} {
			if v != nil && *v != "" {
				if id, ok := byID[norm(*v)]; ok {
					jid = id
					break
				}
			}
		}
		if jid != "" {
			matched[jid] = append(matched[jid], w)
			continue
		}
		key := w.ID
		if w.Email != nil {
			key = w.Venture + ":" + norm(*w.Email)
		}
		if _, ok := orphans[key]; !ok {
			orphanOrder = append(orphanOrder, key)
		}
		orphans[key] = append(orphans[key], w)
	}
	byDate := func(ws []stripe.Win) {
		sort.SliceStable(ws, func(a, b int) bool { return ws[a].At < ws[b].At })
	}
	out := make([]Journey, 0, len(journeys)+len(orphans))
	for _, j := range journeys {
		pays := matched[j.ID]
		if len(pays) == 0 {
			out = append(out, j)
			continue
		}
		byDate(pays)
		paid := 0.0
		touches := append([]Touch(nil), j.Touches...)
		n := len(j.Touches)
		for i, w := range pays {
			paid += w.AmountUSD
			// Stripe touch ids predate PayKit and stay unchanged.
			touches = append(touches, Touch{ID: j.ID + "-" + winSource(w) + "-" + w.ID, ContactID: j.ID, Seq: n + i + 1, Stage: "converted", Channel: "checkout", Label: winLabel(w), Source: winSource(w), At: w.At})
		}
		if j.Product == nil {
			for _, w := range pays {
				if w.Product != nil {
					j.Product = w.Product
					break
				}
			}
		}
		amount := paid
		if j.AmountUSD != nil && *j.AmountUSD > amount {
			amount = *j.AmountUSD
		}
		j.Status, j.Relationship, j.Likelihood, j.AmountUSD, j.Touches = "converted", "hot", 100, &amount, touches
		out = append(out, j)
	}
	for _, key := range orphanOrder {
		group := orphans[key]
		byDate(group)
		first := group[0]
		id := winSource(first) + "-" + first.ID
		name := "Stripe customer"
		if winSource(first) == "paykit" {
			name = "PayKit customer"
		}
		if first.Email != nil {
			name = *first.Email
		}
		var product, phone *string
		named := false
		total := 0.0
		touches := []Touch{}
		for i, w := range group {
			total += w.AmountUSD
			if product == nil && w.Product != nil {
				product = w.Product
			}
			if phone == nil && w.Phone != nil {
				phone = w.Phone
			}
			if !named && w.Name != nil {
				name, named = *w.Name, true
			}
			touches = append(touches, Touch{ID: id + "-t" + strconv.Itoa(i+1), ContactID: id, Seq: i + 1, Stage: "converted", Channel: "checkout", Label: winLabel(w), Source: winSource(w), At: w.At})
		}
		out = append(out, Journey{ID: id, Name: name, Venture: first.Venture, Status: "converted", Product: product, AmountUSD: &total,
			Relationship: "hot", Likelihood: 100, Email: first.Email, Phone: phone, CreatedAt: first.At, Touches: touches})
	}
	return out
}
