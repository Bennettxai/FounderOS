package paperclip

// The shape of the "FounderOS v1 Cockpit" issue (lib/cockpit-issue.ts).
//
// The cockpit is a standing chat thread between the operator and the Conductor,
// not a task. `backlog` is the one status that both the board's handoff and
// its stranded-issue recovery skip while a new comment still wakes the agent
// assignee, and the board allows exactly one assignee, so the Conductor must
// be it.

const CockpitTitle = "FounderOS v1 Cockpit"

const CockpitDescription = "Standing thread: the operator talks to the Conductor from the FounderOS v1 panel. " +
	"Conductor: treat new comments here as direct messages from the operator. Reply in this thread, concisely. " +
	"Delegate real work to the departments/Hermes Workers as separate tasks rather than doing it inline. " +
	"This issue is a chat lane, not a task: leave it in backlog and never move it to in_progress or blocked, " +
	"so the handoff and recovery automations skip it while a new comment still wakes you."

// CockpitSafeStatus is the one status neither automation touches.
const CockpitSafeStatus = "backlog"

func cockpitCreateBody(conductorID string) map[string]any {
	body := map[string]any{"title": CockpitTitle, "description": CockpitDescription, "status": CockpitSafeStatus}
	if conductorID != "" {
		body["assigneeAgentId"] = conductorID
	}
	return body
}

// CockpitRepairPatch is the PATCH that returns an open cockpit issue to the
// safe shape, or nil when nothing needs writing. A closed issue is left
// alone. When the Conductor's id is known and it is not the assignee, the
// patch hands the issue back to it and clears any human assignee.
func CockpitRepairPatch(status, assigneeAgentID, assigneeUserID, conductorID string) map[string]any {
	if status == "done" || status == "cancelled" {
		return nil
	}
	patch := map[string]any{}
	if status != CockpitSafeStatus {
		patch["status"] = CockpitSafeStatus
	}
	if conductorID != "" && assigneeAgentID != conductorID {
		patch["assigneeAgentId"] = conductorID
		if assigneeUserID != "" {
			patch["assigneeUserId"] = nil
		}
	}
	if len(patch) == 0 {
		return nil
	}
	return patch
}
