package sorx

// DefaultSchedules contains the proactive skill schedules provisioned for every
// new user or workspace. All entries are disabled by default — the user must
// explicitly enable each one to opt into proactive behaviour.
//
// These schedules are seeded via Scheduler.ProvisionDefaultSchedules and stored
// in the sorx_schedules table with enabled = false.
var DefaultSchedules = []Schedule{
	{
		// Morning intelligence brief delivered on weekday mornings.
		SkillID:  "daily.brief",
		CronExpr: "0 0 8 * * 1-5", // 08:00 Monday–Friday
		Enabled:  false,
		Params:   map[string]any{"auto": true},
	},
	{
		// Keep the inbox processed throughout the working day.
		SkillID:  "email.process_inbox",
		CronExpr: "0 */30 * * * *", // Every 30 minutes
		Enabled:  false,
	},
	{
		// Pull fresh contact data from the CRM on a regular cadence.
		SkillID:  "crm.sync_contacts",
		CronExpr: "0 0 */2 * * *", // Every 2 hours
		Enabled:  false,
	},
	{
		// Ensure the calendar is always reflected in the workspace.
		SkillID:  "calendar.sync_events",
		CronExpr: "0 0 7 * * *", // 07:00 daily
		Enabled:  false,
	},
	{
		// Weekly pipeline review ready first thing Monday morning.
		SkillID:  "analysis.pipeline",
		CronExpr: "0 0 9 * * 1", // 09:00 every Monday
		Enabled:  false,
	},
	{
		// Client health summary delivered every Friday to close the week.
		SkillID:  "analysis.client_health",
		CronExpr: "0 0 10 * * 5", // 10:00 every Friday
		Enabled:  false,
	},
}
