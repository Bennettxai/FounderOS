package sorx

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Helpers
// ============================================================================

// cronParser returns a second-resolution parser matching the one used
// internally by NewScheduler / registerCronJob.
func cronParser() cron.Parser {
	return cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
}

// mustParseTime is a small test helper that panics rather than returning an
// error so table-driven tests stay compact.
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// ============================================================================
// Schedule struct — JSON serialization
// ============================================================================

func TestSchedule_JSONRoundTrip_FullyPopulated(t *testing.T) {
	// Arrange
	now := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)
	lastRun := now.Add(-time.Hour)
	nextRun := now.Add(time.Hour)
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	original := Schedule{
		ID:          id,
		SkillID:     "daily.brief",
		UserID:      "user-abc",
		WorkspaceID: "ws-xyz",
		CronExpr:    "0 0 8 * * 1-5",
		Params:      map[string]any{"auto": true, "limit": float64(10)},
		Enabled:     true,
		LastRunAt:   &lastRun,
		LastStatus:  StatusComplete,
		LastError:   "",
		NextRunAt:   &nextRun,
		RunCount:    7,
		FailCount:   1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Act — marshal then unmarshal
	data, err := json.Marshal(original)
	require.NoError(t, err, "Schedule should marshal to JSON without error")

	var decoded Schedule
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err, "Schedule should unmarshal from JSON without error")

	// Assert — every exported field survives the round-trip
	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.SkillID, decoded.SkillID)
	assert.Equal(t, original.UserID, decoded.UserID)
	assert.Equal(t, original.WorkspaceID, decoded.WorkspaceID)
	assert.Equal(t, original.CronExpr, decoded.CronExpr)
	assert.Equal(t, original.Enabled, decoded.Enabled)
	assert.Equal(t, original.LastStatus, decoded.LastStatus)
	assert.Equal(t, original.RunCount, decoded.RunCount)
	assert.Equal(t, original.FailCount, decoded.FailCount)
	assert.Equal(t, original.Params["auto"], decoded.Params["auto"])
	assert.Equal(t, original.Params["limit"], decoded.Params["limit"])
	require.NotNil(t, decoded.LastRunAt)
	assert.True(t, original.LastRunAt.Equal(*decoded.LastRunAt))
	require.NotNil(t, decoded.NextRunAt)
	assert.True(t, original.NextRunAt.Equal(*decoded.NextRunAt))
}

func TestSchedule_JSONOmitEmptyFields(t *testing.T) {
	// Arrange — minimal schedule with optional fields left at zero value
	sc := Schedule{
		ID:        uuid.New(),
		SkillID:   "email.process_inbox",
		UserID:    "user-001",
		CronExpr:  "0 */30 * * * *",
		Enabled:   false,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Act
	data, err := json.Marshal(sc)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	// Assert — omitempty fields must be absent when zero
	assert.NotContains(t, raw, "workspace_id", "workspace_id should be omitted when empty")
	assert.NotContains(t, raw, "params", "params should be omitted when nil")
	assert.NotContains(t, raw, "last_run_at", "last_run_at should be omitted when nil")
	assert.NotContains(t, raw, "last_status", "last_status should be omitted when empty")
	assert.NotContains(t, raw, "last_error", "last_error should be omitted when empty")
	assert.NotContains(t, raw, "next_run_at", "next_run_at should be omitted when nil")
}

func TestSchedule_JSONFieldNames(t *testing.T) {
	// Verify the JSON field name mapping is stable — changing struct tags would
	// be a breaking API change.
	sc := Schedule{
		ID:        uuid.New(),
		SkillID:   "crm.sync_contacts",
		UserID:    "u1",
		CronExpr:  "0 0 */2 * * *",
		Enabled:   true,
		RunCount:  3,
		FailCount: 0,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(sc)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	expectedKeys := []string{
		"id", "skill_id", "user_id", "cron_expr",
		"enabled", "run_count", "fail_count", "created_at", "updated_at",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, raw, key, "JSON output should contain field %q", key)
	}
}

// ============================================================================
// nullableString helper
// ============================================================================

func TestNullableString_EmptyStringReturnsNil(t *testing.T) {
	result := nullableString("")
	assert.Nil(t, result, "empty string should produce a nil pointer")
}

func TestNullableString_NonEmptyStringReturnsPointer(t *testing.T) {
	input := "workspace-123"
	result := nullableString(input)
	require.NotNil(t, result, "non-empty string should produce a non-nil pointer")
	assert.Equal(t, input, *result, "pointer should dereference to the original string")
}

func TestNullableString_WhitespaceOnlyIsNonEmpty(t *testing.T) {
	// A string containing only whitespace is not an empty string — the caller
	// is responsible for trimming. nullableString is purely length-based.
	result := nullableString("   ")
	require.NotNil(t, result, "whitespace-only string is not empty, should return pointer")
	assert.Equal(t, "   ", *result)
}

func TestNullableString_SingleCharacterReturnsPointer(t *testing.T) {
	result := nullableString("x")
	require.NotNil(t, result)
	assert.Equal(t, "x", *result)
}

// ============================================================================
// DefaultSchedules — existence and shape
// ============================================================================

func TestDefaultSchedules_ExactlySixEntries(t *testing.T) {
	assert.Len(t, DefaultSchedules, 6,
		"there should be exactly 6 default schedules")
}

func TestDefaultSchedules_AllDisabledByDefault(t *testing.T) {
	for i, sc := range DefaultSchedules {
		assert.False(t, sc.Enabled,
			"DefaultSchedules[%d] (skill %q) must be disabled by default so users explicitly opt in",
			i, sc.SkillID)
	}
}

func TestDefaultSchedules_SkillIDsMatchExpected(t *testing.T) {
	expectedSkillIDs := []string{
		"daily.brief",
		"email.process_inbox",
		"crm.sync_contacts",
		"calendar.sync_events",
		"analysis.pipeline",
		"analysis.client_health",
	}

	actualSkillIDs := make([]string, len(DefaultSchedules))
	for i, sc := range DefaultSchedules {
		actualSkillIDs[i] = sc.SkillID
	}

	assert.Equal(t, expectedSkillIDs, actualSkillIDs,
		"default schedule skill IDs must match expected list in declaration order")
}

func TestDefaultSchedules_NoDuplicateSkillIDs(t *testing.T) {
	seen := make(map[string]int)
	for i, sc := range DefaultSchedules {
		if prev, exists := seen[sc.SkillID]; exists {
			t.Errorf("skill_id %q appears at both index %d and %d", sc.SkillID, prev, i)
		}
		seen[sc.SkillID] = i
	}
}

func TestDefaultSchedules_AllHaveNonEmptyCronExpression(t *testing.T) {
	for i, sc := range DefaultSchedules {
		assert.NotEmpty(t, sc.CronExpr,
			"DefaultSchedules[%d] (skill %q) must have a non-empty cron expression", i, sc.SkillID)
	}
}

// ============================================================================
// DefaultSchedules — cron expression validity (6-field, second-resolution)
// ============================================================================

func TestDefaultSchedules_AllCronExpressionsParseWithSeconds(t *testing.T) {
	parser := cronParser()

	for i, sc := range DefaultSchedules {
		sc := sc // capture loop variable
		t.Run(sc.SkillID, func(t *testing.T) {
			_, err := parser.Parse(sc.CronExpr)
			assert.NoError(t, err,
				"DefaultSchedules[%d] skill %q has invalid cron expression %q",
				i, sc.SkillID, sc.CronExpr)
		})
	}
}

func TestDefaultSchedules_CronExpressionsHaveSixFields(t *testing.T) {
	// The scheduler uses cron.WithSeconds(), meaning all expressions must have
	// 6 space-separated fields: second minute hour dom month dow.
	for i, sc := range DefaultSchedules {
		fields := splitFields(sc.CronExpr)
		assert.Equal(t, 6, len(fields),
			"DefaultSchedules[%d] skill %q expression %q should have 6 fields (sec min hr dom mon dow)",
			i, sc.SkillID, sc.CronExpr)
	}
}

// splitFields counts whitespace-separated tokens in a cron expression.
func splitFields(expr string) []string {
	var fields []string
	inField := false
	start := 0
	for i, ch := range expr {
		if ch == ' ' || ch == '\t' {
			if inField {
				fields = append(fields, expr[start:i])
				inField = false
			}
		} else {
			if !inField {
				start = i
				inField = true
			}
		}
	}
	if inField {
		fields = append(fields, expr[start:])
	}
	return fields
}

// ============================================================================
// DefaultSchedules — individual cron expression semantics
// ============================================================================

func TestDefaultSchedules_DailyBrief_RunsWeekdayMornings(t *testing.T) {
	// "0 0 8 * * 1-5" → 08:00 Mon–Fri
	var sc *Schedule
	for i := range DefaultSchedules {
		if DefaultSchedules[i].SkillID == "daily.brief" {
			sc = &DefaultSchedules[i]
			break
		}
	}
	require.NotNil(t, sc)
	assert.Equal(t, "0 0 8 * * 1-5", sc.CronExpr)

	parser := cronParser()
	sched, err := parser.Parse(sc.CronExpr)
	require.NoError(t, err)

	// Sunday 2025-06-15 → next fire is Monday 2025-06-16 at 08:00
	sunday := mustParseTime("2025-06-15T10:00:00Z")
	next := sched.Next(sunday)
	assert.Equal(t, 1, int(next.Weekday()), "next run after Sunday should be Monday (weekday=1)")
	assert.Equal(t, 8, next.Hour(), "daily.brief should fire at 08:00")
	assert.Equal(t, 0, next.Minute())
	assert.Equal(t, 0, next.Second())
}

func TestDefaultSchedules_DailyBrief_HasAutoParam(t *testing.T) {
	for _, sc := range DefaultSchedules {
		if sc.SkillID == "daily.brief" {
			require.NotNil(t, sc.Params, "daily.brief should have params")
			v, ok := sc.Params["auto"]
			require.True(t, ok, "daily.brief params should contain 'auto'")
			assert.Equal(t, true, v, "daily.brief 'auto' param should be true")
			return
		}
	}
	t.Fatal("daily.brief not found in DefaultSchedules")
}

func TestDefaultSchedules_ProcessInbox_RunsEvery30Minutes(t *testing.T) {
	// "0 */30 * * * *" → fires at :00 and :30 of every hour
	var sc *Schedule
	for i := range DefaultSchedules {
		if DefaultSchedules[i].SkillID == "email.process_inbox" {
			sc = &DefaultSchedules[i]
			break
		}
	}
	require.NotNil(t, sc)
	assert.Equal(t, "0 */30 * * * *", sc.CronExpr)

	parser := cronParser()
	sched, err := parser.Parse(sc.CronExpr)
	require.NoError(t, err)

	// Seed at :15 → next fire should be :30 of the same hour
	seed := mustParseTime("2025-06-15T14:15:00Z")
	next := sched.Next(seed)
	assert.Equal(t, 30, next.Minute(), "email.process_inbox next run should be at minute :30")
	assert.Equal(t, 0, next.Second())

	// From :30 → next should be :00 of the following hour
	next2 := sched.Next(next)
	assert.Equal(t, 0, next2.Minute(), "email.process_inbox fires at minute :00 of the next hour too")
	assert.Equal(t, 15, next2.Hour(), "should advance one hour")
}

func TestDefaultSchedules_CRMSync_RunsEvery2Hours(t *testing.T) {
	// "0 0 */2 * * *" → fires at 00:00, 02:00, 04:00, …
	var sc *Schedule
	for i := range DefaultSchedules {
		if DefaultSchedules[i].SkillID == "crm.sync_contacts" {
			sc = &DefaultSchedules[i]
			break
		}
	}
	require.NotNil(t, sc)
	assert.Equal(t, "0 0 */2 * * *", sc.CronExpr)

	parser := cronParser()
	sched, err := parser.Parse(sc.CronExpr)
	require.NoError(t, err)

	// Seed at 01:30 → next should be 02:00
	seed := mustParseTime("2025-06-15T01:30:00Z")
	next := sched.Next(seed)
	assert.Equal(t, 2, next.Hour(), "crm.sync_contacts next run should be at 02:00")
	assert.Equal(t, 0, next.Minute())
}

func TestDefaultSchedules_CalendarSync_RunsAt0700Daily(t *testing.T) {
	// "0 0 7 * * *"
	var sc *Schedule
	for i := range DefaultSchedules {
		if DefaultSchedules[i].SkillID == "calendar.sync_events" {
			sc = &DefaultSchedules[i]
			break
		}
	}
	require.NotNil(t, sc)
	assert.Equal(t, "0 0 7 * * *", sc.CronExpr)

	parser := cronParser()
	sched, err := parser.Parse(sc.CronExpr)
	require.NoError(t, err)

	// Seed late in the day → fires at 07:00 next day
	seed := mustParseTime("2025-06-15T20:00:00Z")
	next := sched.Next(seed)
	assert.Equal(t, 7, next.Hour())
	assert.Equal(t, 0, next.Minute())
	assert.Equal(t, 0, next.Second())
	assert.Equal(t, 16, next.Day(), "should advance to the 16th")
}

func TestDefaultSchedules_PipelineAnalysis_RunsMondayAt0900(t *testing.T) {
	// "0 0 9 * * 1"
	var sc *Schedule
	for i := range DefaultSchedules {
		if DefaultSchedules[i].SkillID == "analysis.pipeline" {
			sc = &DefaultSchedules[i]
			break
		}
	}
	require.NotNil(t, sc)
	assert.Equal(t, "0 0 9 * * 1", sc.CronExpr)

	parser := cronParser()
	sched, err := parser.Parse(sc.CronExpr)
	require.NoError(t, err)

	// Wednesday 2025-06-18 → next Monday is 2025-06-23
	seed := mustParseTime("2025-06-18T09:01:00Z")
	next := sched.Next(seed)
	assert.Equal(t, 1, int(next.Weekday()), "analysis.pipeline should fire on Monday (weekday=1)")
	assert.Equal(t, 9, next.Hour())
	assert.Equal(t, 0, next.Minute())
}

func TestDefaultSchedules_ClientHealth_RunsFridayAt1000(t *testing.T) {
	// "0 0 10 * * 5"
	var sc *Schedule
	for i := range DefaultSchedules {
		if DefaultSchedules[i].SkillID == "analysis.client_health" {
			sc = &DefaultSchedules[i]
			break
		}
	}
	require.NotNil(t, sc)
	assert.Equal(t, "0 0 10 * * 5", sc.CronExpr)

	parser := cronParser()
	sched, err := parser.Parse(sc.CronExpr)
	require.NoError(t, err)

	// Monday 2025-06-16 → next Friday is 2025-06-20
	seed := mustParseTime("2025-06-16T08:00:00Z")
	next := sched.Next(seed)
	assert.Equal(t, 5, int(next.Weekday()), "analysis.client_health should fire on Friday (weekday=5)")
	assert.Equal(t, 10, next.Hour())
	assert.Equal(t, 0, next.Minute())
}

// ============================================================================
// Cron expression parsing — valid / invalid cases (unit-level, no DB needed)
// ============================================================================

func TestCronParsing_ValidSixFieldExpressions(t *testing.T) {
	parser := cronParser()

	validExprs := []struct {
		expr        string
		description string
	}{
		{"0 0 8 * * 1-5", "weekday 08:00"},
		{"0 */30 * * * *", "every 30 minutes"},
		{"0 0 */2 * * *", "every 2 hours"},
		{"0 0 7 * * *", "daily at 07:00"},
		{"0 0 9 * * 1", "Monday 09:00"},
		{"0 0 10 * * 5", "Friday 10:00"},
		{"0 0 0 * * *", "midnight daily"},
		{"0 0 12 1 * *", "noon on the 1st of every month"},
		{"*/10 * * * * *", "every 10 seconds"},
		{"0 0 0 1 1 *", "midnight on Jan 1st"},
	}

	for _, tc := range validExprs {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			_, err := parser.Parse(tc.expr)
			assert.NoError(t, err, "expression %q should be valid", tc.expr)
		})
	}
}

func TestCronParsing_InvalidExpressionsAreRejected(t *testing.T) {
	parser := cronParser()

	invalidExprs := []struct {
		expr        string
		description string
	}{
		{"* * * * *", "5-field expression (missing seconds)"},
		{"@hourly", "named schedule — not supported with WithSeconds"},
		{"", "empty string"},
		{"0 0 25 * * *", "hour out of range (25)"},
		{"0 60 * * * *", "minute out of range (60)"},
		{"70 * * * * *", "second out of range (70)"},
		{"0 0 8 * * 8", "day-of-week out of range (8)"},
		{"not a cron", "gibberish"},
	}

	for _, tc := range invalidExprs {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			_, err := parser.Parse(tc.expr)
			assert.Error(t, err,
				"expression %q should be rejected as invalid", tc.expr)
		})
	}
}

// ============================================================================
// Schedule JSON — edge cases for Params field
// ============================================================================

func TestSchedule_NilParamsOmittedFromJSON(t *testing.T) {
	sc := Schedule{
		ID:        uuid.New(),
		SkillID:   "calendar.sync_events",
		UserID:    "u2",
		CronExpr:  "0 0 7 * * *",
		Params:    nil,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(sc)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.NotContains(t, raw, "params", "nil Params should be omitted (omitempty)")
}

func TestSchedule_EmptyParamsOmittedFromJSON(t *testing.T) {
	// Go's encoding/json treats an empty map as a zero value under omitempty,
	// so map[string]any{} (non-nil but empty) is elided just like a nil map.
	// This test documents that behaviour so callers are not surprised.
	sc := Schedule{
		ID:        uuid.New(),
		SkillID:   "crm.sync_contacts",
		UserID:    "u3",
		CronExpr:  "0 0 */2 * * *",
		Params:    map[string]any{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(sc)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))
	// encoding/json omitempty elides an empty map — it is absent from the output.
	assert.NotContains(t, raw, "params",
		"an empty (non-nil) params map is omitted by omitempty in encoding/json")
}

func TestSchedule_ComplexParamsPreservedThroughJSON(t *testing.T) {
	sc := Schedule{
		ID:      uuid.New(),
		SkillID: "daily.brief",
		UserID:  "u4",
		Params: map[string]any{
			"auto":       true,
			"max_emails": float64(20),
			"filters":    []any{"label:INBOX", "label:UNREAD"},
		},
		CronExpr:  "0 0 8 * * 1-5",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(sc)
	require.NoError(t, err)

	var decoded Schedule
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, true, decoded.Params["auto"])
	assert.Equal(t, float64(20), decoded.Params["max_emails"])

	filters, ok := decoded.Params["filters"].([]any)
	require.True(t, ok, "filters should decode as []any")
	assert.Len(t, filters, 2)
	assert.Equal(t, "label:INBOX", filters[0])
}

// ============================================================================
// Schedule JSON — time pointer fields
// ============================================================================

func TestSchedule_TimePtrFieldsRoundTripWithNanosecondPrecision(t *testing.T) {
	now := time.Date(2025, 3, 10, 8, 0, 0, 123456789, time.UTC)
	sc := Schedule{
		ID:        uuid.New(),
		SkillID:   "analysis.pipeline",
		UserID:    "u5",
		CronExpr:  "0 0 9 * * 1",
		LastRunAt: &now,
		NextRunAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(sc)
	require.NoError(t, err)

	var decoded Schedule
	require.NoError(t, json.Unmarshal(data, &decoded))

	require.NotNil(t, decoded.LastRunAt)
	require.NotNil(t, decoded.NextRunAt)
	// time.Time JSON marshaling uses RFC3339Nano — sub-second precision is kept.
	assert.True(t, now.Equal(*decoded.LastRunAt),
		"LastRunAt should survive JSON round-trip")
	assert.True(t, now.Equal(*decoded.NextRunAt),
		"NextRunAt should survive JSON round-trip")
}

// ============================================================================
// StatusConstants sanity check
// ============================================================================

func TestStatusConstants_HaveExpectedValues(t *testing.T) {
	// These values are persisted to the database so their string form must never
	// change without a migration.
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "running", StatusRunning)
	assert.Equal(t, "waiting_callback", StatusWaitingCallback)
	assert.Equal(t, "waiting_decision", StatusWaitingDecision)
	assert.Equal(t, "complete", StatusComplete)
	assert.Equal(t, "failed", StatusFailed)
	assert.Equal(t, "cancelled", StatusCancelled)
}

func TestStatusConstants_LastStatusValues_UsedByScheduler(t *testing.T) {
	// persistScheduleOutcome writes StatusComplete or StatusFailed to last_status.
	// Confirm those two constants are non-empty and distinct.
	assert.NotEmpty(t, StatusComplete)
	assert.NotEmpty(t, StatusFailed)
	assert.NotEqual(t, StatusComplete, StatusFailed)
}
