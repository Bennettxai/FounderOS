package agents

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Cron is one scheduled agent job (founderos_agent_crons + its last run).
type Cron struct {
	ID          string     `json:"id"`
	AgentID     string     `json:"agentId"`
	Schedule    string     `json:"schedule"`
	Description string     `json:"description"`
	Enabled     bool       `json:"enabled"`
	LastRunAt   *time.Time `json:"lastRunAt"`
}

var cronField = regexp.MustCompile(`^(\*|[0-9*/,-]+)$`)

// ValidCron accepts 5 numeric cron fields (lib/cron.ts isValidCron).
func ValidCron(expr string) bool {
	f := strings.Fields(expr)
	if len(f) != 5 {
		return false
	}
	for _, x := range f {
		if !cronField.MatchString(x) {
			return false
		}
	}
	return true
}

func fieldMatches(field string, value int) bool {
	for _, part := range strings.Split(field, ",") {
		spec, stepRaw, hasStep := strings.Cut(part, "/")
		step := 1
		if hasStep {
			n, err := strconv.Atoi(stepRaw)
			if err != nil || n < 1 {
				continue
			}
			step = n
		}
		if spec == "*" {
			if value%step == 0 {
				return true
			}
			continue
		}
		if lo, hi, isRange := strings.Cut(spec, "-"); isRange {
			a, e1 := strconv.Atoi(lo)
			b, e2 := strconv.Atoi(hi)
			if e1 == nil && e2 == nil && value >= a && value <= b && (value-a)%step == 0 {
				return true
			}
			continue
		}
		if n, err := strconv.Atoi(spec); err == nil {
			if hasStep {
				if value >= n && (value-n)%step == 0 {
					return true
				}
			} else if value == n {
				return true
			}
		}
	}
	return false
}

// CronZone is the wall clock crons are written in: the operator's
// (America/Chicago), or FOUNDEROS_TZ. FounderOS v1 used process-local time, which
// is the same on the Macs under launchd; pinning it keeps the 09:00 digest at
// 09:00 under a UTC container too.
var CronZone = cronZone()

func cronZone() *time.Location {
	name := os.Getenv("FOUNDEROS_TZ")
	if name == "" {
		name = "America/Chicago"
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return time.Local
}

// MatchesCron reports whether expr fires during when's minute on CronZone's
// wall clock.
func MatchesCron(expr string, when time.Time) bool {
	if !ValidCron(expr) {
		return false
	}
	when = when.In(CronZone)
	f := strings.Fields(expr)
	return fieldMatches(f[0], when.Minute()) && fieldMatches(f[1], when.Hour()) &&
		fieldMatches(f[2], when.Day()) && fieldMatches(f[3], int(when.Month())) &&
		fieldMatches(f[4], int(when.Weekday()))
}

const lookMinutes = 8 * 24 * 60

// LastOccurrence is the latest minute at or before now that expr fires.
func LastOccurrence(expr string, now time.Time) (time.Time, bool) {
	if !ValidCron(expr) {
		return time.Time{}, false
	}
	c := now.Truncate(time.Minute)
	for i := 0; i <= lookMinutes; i++ {
		if MatchesCron(expr, c) {
			return c, true
		}
		c = c.Add(-time.Minute)
	}
	return time.Time{}, false
}

// NextOccurrence is the first minute after now that expr fires.
func NextOccurrence(expr string, now time.Time) (time.Time, bool) {
	if !ValidCron(expr) {
		return time.Time{}, false
	}
	c := now.Truncate(time.Minute)
	for i := 0; i < lookMinutes; i++ {
		c = c.Add(time.Minute)
		if MatchesCron(expr, c) {
			return c, true
		}
	}
	return time.Time{}, false
}

// DueCrons returns enabled crons whose latest window has not run yet.
func DueCrons(crons []Cron, now time.Time) []Cron {
	var due []Cron
	for _, c := range crons {
		if !c.Enabled {
			continue
		}
		occ, ok := LastOccurrence(c.Schedule, now)
		if !ok {
			continue
		}
		if c.LastRunAt == nil || c.LastRunAt.Before(occ) {
			due = append(due, c)
		}
	}
	return due
}

var dow = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func dowLabel(field string) string {
	if field == "*" {
		return "daily"
	}
	if a, b, ok := strings.Cut(field, "-"); ok {
		x, e1 := strconv.Atoi(a)
		y, e2 := strconv.Atoi(b)
		if e1 == nil && e2 == nil && x <= 6 && y <= 6 {
			return dow[x] + "-" + dow[y]
		}
	}
	if n, err := strconv.Atoi(field); err == nil && n <= 6 && len(field) == 1 {
		return dow[n]
	}
	return field
}

var everyN = regexp.MustCompile(`^\*/(\d+)$`)
var digits = regexp.MustCompile(`^\d+$`)

// DescribeCron is the human summary shown next to a schedule.
func DescribeCron(expr string) string {
	if !ValidCron(expr) {
		return ""
	}
	f := strings.Fields(expr)
	min, hour, wd := f[0], f[1], f[4]
	if m := everyN.FindStringSubmatch(min); m != nil && hour == "*" {
		return "every " + m[1] + " min"
	}
	pad := func(s string) string {
		if len(s) < 2 {
			return "0" + s
		}
		return s
	}
	if digits.MatchString(min) && hour == "*" {
		return "hourly at :" + pad(min)
	}
	if digits.MatchString(min) && digits.MatchString(hour) {
		return "at " + pad(hour) + ":" + pad(min) + ", " + dowLabel(wd)
	}
	return "cron " + expr
}
