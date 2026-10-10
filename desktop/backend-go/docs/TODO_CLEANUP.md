# TODO Cleanup Tracker

This document tracks TODOs and debug statements in the backend codebase that should be addressed.

## Completed (This Session)

### ✅ Comments Count in Mobile API
- **File:** `internal/handlers/mobile_handlers.go`
- **Change:** Added `CountCommentsByEntity` query call in `GetTask` handler
- **Details:** Mobile API now returns actual comment counts for tasks

### ✅ Notification Preferences in Mobile API
- **File:** `internal/handlers/mobile_handlers.go`
- **Change:** Wired up `GetNotificationPreferencesByUser` query in `GetMe` handler
- **Details:** Returns user's actual notification settings including quiet hours

### ✅ Debug Print in Comment Service
- **File:** `internal/services/comment_service.go:123`
- **Change:** Replaced `fmt.Printf` with `log.Printf("[CommentService]...")`
- **Details:** Proper production logging for mention storage errors

---

## Remaining Debug Prints (Low Priority)

These are debug statements that should be reviewed and potentially converted to proper logging:

### LLM Service
- **File:** `internal/services/llm.go`
- **Lines:** 142, 160
- **Statement:** `fmt.Printf("[LLM]...)`
- **Recommendation:** Convert to `log.Printf` or structured logging

### Whisper Service
- **File:** `internal/services/whisper.go`
- **Line:** 35
- **Statement:** `fmt.Printf("Whisper service init...)`
- **Recommendation:** Convert to `log.Printf` or remove (init message)

### Groq Service
- **File:** `internal/services/groq.go`
- **Line:** 168
- **Statement:** `fmt.Printf("[Groq]...)`
- **Recommendation:** Convert to `log.Printf` or structured logging

---

## Remaining Feature TODOs (Future Work)

### Tags Feature
- **File:** `internal/handlers/mobile_types.go:174`
- **Current:** `Tags: []string{}, // Tags feature not yet implemented`
- **Requires:** Implementation of `task_tags` table and queries
- **Priority:** Medium

### Attachments Feature
- **File:** `internal/handlers/mobile_types.go:177`
- **Current:** `AttachmentsCount: 0, // Attachments feature not yet implemented`
- **Requires:** Implementation of attachments table and file storage
- **Priority:** Medium

### User Timezone Settings
- **File:** `internal/handlers/mobile_handlers.go:56`
- **Current:** `userResp.Timezone = "UTC" // TODO: Implement user settings lookup`
- **Requires:** User settings table or profile field
- **Priority:** Low

---

## Search Commands

To find all remaining TODOs:
```bash
grep -rn "TODO" internal/ --include="*.go"
```

To find all debug prints:
```bash
grep -rn "fmt.Printf" internal/ --include="*.go"
```

---

*Last Updated: Session implementing comments and mentions system*
