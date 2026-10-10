// Package social is the bridge port of FounderOS v1's /social pages: the
// growth math (lib/growth.ts), the dashboard, audience and DM derivations
// (lib/social.ts, lib/email-list.ts), the slab view models
// (lib/social-volume.ts, lib/newsletter-volume.ts) and the Postgres reads
// they run on (founderos_social_*, founderos_email_list_*, personal).
package social

import (
	"sort"
	"time"
)

// GrowthPoint is one dated value; series are sorted oldest → newest.
type GrowthPoint struct {
	CapturedAt string  `json:"capturedAt"`
	Value      float64 `json:"value"`
}

// Growth is SocialGrowth: nil means history too short, never a fake zero.
type Growth struct {
	D7      *float64 `json:"d7"`
	D30     *float64 `json:"d30"`
	D60     *float64 `json:"d60"`
	AllTime *float64 `json:"allTime"`
}

// Delta is windowDelta's current value and baseline.
type Delta struct{ Current, Baseline float64 }

const dayMS = 86_400_000

func dateMS(day string) (int64, bool) {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return 0, false
	}
	return t.UnixMilli(), true
}

// WindowDelta is the latest value and the baseline at/before the trailing
// window start, or nil when no snapshot reaches back that far (or the
// baseline is zero). maxLagDays > 0 refuses a baseline further than that
// before the window start (FOS-659).
func WindowDelta(points []GrowthPoint, days int, maxLagDays int) *Delta {
	if len(points) < 2 {
		return nil
	}
	latest := points[len(points)-1]
	end, ok := dateMS(latest.CapturedAt)
	if !ok {
		return nil
	}
	start := end - int64(days)*dayMS
	for i := len(points) - 1; i >= 0; i-- {
		at, ok := dateMS(points[i].CapturedAt)
		if !ok || at > start {
			continue
		}
		if points[i].Value == 0 {
			return nil
		}
		if maxLagDays > 0 && start-at > int64(maxLagDays)*dayMS {
			return nil
		}
		return &Delta{Current: latest.Value, Baseline: points[i].Value}
	}
	return nil
}

func pct(d *Delta) *float64 {
	if d == nil {
		return nil
	}
	v := (d.Current - d.Baseline) / d.Baseline * 100
	return &v
}

// GrowthOver is the percentage growth over the trailing window.
func GrowthOver(points []GrowthPoint, days int, maxLagDays int) *float64 {
	return pct(WindowDelta(points, days, maxLagDays))
}

// GrowthAllTime is first-to-last growth.
func GrowthAllTime(points []GrowthPoint) *float64 {
	if len(points) < 2 || points[0].Value == 0 {
		return nil
	}
	return pct(&Delta{Current: points[len(points)-1].Value, Baseline: points[0].Value})
}

// MergeSeriesSum sums several series over the union of their dates, each
// channel's latest value carried forward; a channel adds nothing before its
// first point.
func MergeSeriesSum(list [][]GrowthPoint) []GrowthPoint {
	seen := map[string]bool{}
	var dates []string
	for _, s := range list {
		for _, p := range s {
			if !seen[p.CapturedAt] {
				seen[p.CapturedAt] = true
				dates = append(dates, p.CapturedAt)
			}
		}
	}
	sort.Strings(dates)
	out := []GrowthPoint{}
	for _, d := range dates {
		sum, any := 0.0, false
		for _, s := range list {
			var carried *float64
			for i := range s {
				if s[i].CapturedAt <= d {
					carried = &s[i].Value
				} else {
					break
				}
			}
			if carried != nil {
				sum += *carried
				any = true
			}
		}
		if any {
			out = append(out, GrowthPoint{CapturedAt: d, Value: sum})
		}
	}
	return out
}
