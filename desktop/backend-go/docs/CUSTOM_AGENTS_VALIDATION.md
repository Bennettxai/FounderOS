# Custom Agents Input Validation

## Overview

Comprehensive input validation has been added to the Custom Agents handlers to prevent abuse, ensure data quality, and improve security.

## Validation Rules

### 1. Rate Limiting

**Rule:** Maximum 100 custom agents per user

**Location:** `CreateCustomAgent` handler (before agent creation)

**Implementation:**
```go
count, err := queries.CountUserAgents(ctx, user.ID)
if count >= 100 {
    return 400 Bad Request: "Maximum 100 custom agents allowed per user. Please delete unused agents."
}
```

**Rationale:** Prevents resource abuse and ensures database performance.

---

### 2. Suggested Prompts Validation

**Rules:**
- Maximum 10 prompts per agent
- Each prompt: max 500 characters
- No empty/whitespace-only prompts

**Location:** Both `CreateCustomAgent` and `UpdateCustomAgent`

**Error Messages:**
- `"Maximum 10 suggested prompts allowed"`
- `"Suggested prompt {N} cannot be empty"`
- `"Suggested prompt {N} exceeds 500 characters (has {count})"`

**Implementation:**
```go
if len(req.SuggestedPrompts) > 10 {
    return 400 Bad Request
}
for i, prompt := range req.SuggestedPrompts {
    if len(strings.TrimSpace(prompt)) == 0 {
        return 400 Bad Request
    }
    if len(prompt) > 500 {
        return 400 Bad Request
    }
}
```

**Rationale:**
- Prevents UI clutter
- Ensures prompts are meaningful
- Protects database storage

---

### 3. Welcome Message Validation

**Rule:** Maximum 2000 characters

**Location:** Both `CreateCustomAgent` and `UpdateCustomAgent`

**Error Message:**
- `"Welcome message exceeds 2000 characters (has {count})"`

**Implementation:**
```go
if len(req.WelcomeMessage) > 2000 {
    return 400 Bad Request
}
```

**Rationale:**
- Welcome messages display in UI
- Prevents excessively long messages
- Ensures good UX

---

### 4. Category Validation

**Rule:** Must be one of the allowed categories

**Allowed Categories:**
- `general`
- `coding`
- `writing`
- `analysis`
- `research`
- `support`
- `sales`
- `marketing`

**Location:** Both `CreateCustomAgent` and `UpdateCustomAgent`

**Error Message:**
- `"Invalid category. Allowed: general, coding, writing, analysis, research, support, sales, marketing"`

**Implementation:**
```go
allowedCategories := map[string]bool{
    "general":   true,
    "coding":    true,
    "writing":   true,
    "analysis":  true,
    "research":  true,
    "support":   true,
    "sales":     true,
    "marketing": true,
}
if req.Category != "" && !allowedCategories[req.Category] {
    return 400 Bad Request
}
```

**Rationale:**
- Ensures consistent categorization
- Enables category-based filtering
- Prevents invalid data in database

---

### 5. Agent Name Validation (Pre-existing)

**Rules:**
- Lowercase letters only (a-z)
- Numbers (0-9)
- Hyphens (-)
- No spaces or special characters

**Location:** Both `CreateCustomAgent` and `UpdateCustomAgent`

**Error Message:**
- `"Agent name can only contain lowercase letters, numbers, and hyphens"`

**Rationale:**
- URL-safe names
- Consistent naming convention

---

### 6. Temperature Validation (Pre-existing)

**Rule:** Must be between 0.0 and 2.0

**Location:** Both `CreateCustomAgent` and `UpdateCustomAgent`

**Error Message:**
- `"Temperature must be between 0.0 and 2.0"`

**Rationale:**
- LLM API constraints
- Ensures predictable model behavior

---

## Database Changes

### New SQLC Query

**File:** `desktop/backend-go/internal/database/queries/custom_agents.sql`

**Query:**
```sql
-- name: CountUserAgents :one
SELECT COUNT(*) FROM custom_agents WHERE user_id = $1;
```

**Purpose:** Enables rate limiting by counting user's existing agents.

**Regeneration:** Run `sqlc generate` after adding this query.

---

## API Behavior

### HTTP Status Codes

| Scenario | Status | Response |
|----------|--------|----------|
| Validation error | 400 Bad Request | `{"error": "validation message"}` |
| Rate limit exceeded | 400 Bad Request | `{"error": "Maximum 100 custom agents..."}` |
| Successful create | 201 Created | `{"agent": {...}}` |
| Successful update | 200 OK | `{"agent": {...}}` |

### Example Validation Error Response

```json
{
  "error": "Suggested prompt 2 exceeds 500 characters (has 523)"
}
```

---

## Testing

### Automated Test Script

**Location:** `desktop/backend-go/scripts/tests/test-agent-validation.sh`

**Coverage:**
- Too many suggested prompts (>10)
- Suggested prompt too long (>500 chars)
- Empty suggested prompt
- Welcome message too long (>2000 chars)
- Invalid category
- Valid agent creation
- Valid agent update
- Invalid category update
- Too many prompts update

**Usage:**
```bash
# Update AUTH_HEADER in script first
bash desktop/backend-go/scripts/tests/test-agent-validation.sh
```

### Manual Testing

#### Test 1: Create Agent with Too Many Prompts
```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test",
    "display_name": "Test",
    "system_prompt": "You are helpful",
    "suggested_prompts": ["1","2","3","4","5","6","7","8","9","10","11"]
  }'

# Expected: 400 Bad Request
# {"error": "Maximum 10 suggested prompts allowed"}
```

#### Test 2: Create Agent with Invalid Category
```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test",
    "display_name": "Test",
    "system_prompt": "You are helpful",
    "category": "invalid"
  }'

# Expected: 400 Bad Request
# {"error": "Invalid category. Allowed: general, coding, ..."}
```

#### Test 3: Valid Agent Creation
```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "my-agent",
    "display_name": "My Agent",
    "system_prompt": "You are a helpful coding assistant",
    "category": "coding",
    "welcome_message": "Hello! Ready to code?",
    "suggested_prompts": [
      "Help me debug",
      "Review my code",
      "Explain this pattern"
    ]
  }'

# Expected: 201 Created
# {"agent": {...}}
```

---

## Frontend Integration

### Validation on Client Side (Recommended)

Add matching validation in frontend forms to provide immediate feedback:

```typescript
// frontend/src/lib/components/agents/AgentForm.svelte

const validateSuggestedPrompts = (prompts: string[]) => {
  if (prompts.length > 10) {
    return "Maximum 10 suggested prompts allowed";
  }
  for (let i = 0; i < prompts.length; i++) {
    if (prompts[i].trim().length === 0) {
      return `Prompt ${i + 1} cannot be empty`;
    }
    if (prompts[i].length > 500) {
      return `Prompt ${i + 1} exceeds 500 characters (has ${prompts[i].length})`;
    }
  }
  return null;
};

const validateWelcomeMessage = (message: string) => {
  if (message.length > 2000) {
    return `Welcome message exceeds 2000 characters (has ${message.length})`;
  }
  return null;
};

const allowedCategories = [
  "general", "coding", "writing", "analysis",
  "research", "support", "sales", "marketing"
];

const validateCategory = (category: string) => {
  if (category && !allowedCategories.includes(category)) {
    return "Invalid category";
  }
  return null;
};
```

### Character Counters (UX Enhancement)

```svelte
<!-- Suggested Prompt Input -->
<div>
  <input
    bind:value={prompt}
    maxlength="500"
    placeholder="Enter suggested prompt"
  />
  <span class="text-sm text-gray-500">
    {prompt.length}/500 characters
  </span>
</div>

<!-- Welcome Message Textarea -->
<div>
  <textarea
    bind:value={welcomeMessage}
    maxlength="2000"
    placeholder="Enter welcome message"
  ></textarea>
  <span class="text-sm text-gray-500">
    {welcomeMessage.length}/2000 characters
  </span>
</div>
```

---

## Security Considerations

### Why These Limits?

1. **Rate Limiting (100 agents):**
   - Prevents account abuse
   - Protects database from bloat
   - Encourages users to maintain quality over quantity

2. **Suggested Prompts (10 max, 500 chars each):**
   - Prevents UI overflow
   - Ensures prompts are concise and useful
   - Protects against injection attacks

3. **Welcome Message (2000 chars):**
   - Balances flexibility with UX
   - Prevents excessively long messages
   - Limits potential for abuse

4. **Category Whitelist:**
   - Ensures data integrity
   - Enables reliable filtering/search
   - Prevents arbitrary categorization

### What's NOT Validated (Consider Adding)

1. **System Prompt Length:** Currently unlimited
   - Recommendation: Add 10,000 character limit
   - Rationale: Prevents token waste, improves performance

2. **Display Name Length:** Currently unlimited
   - Recommendation: Add 100 character limit
   - Rationale: Ensures UI compatibility

3. **Description Length:** Currently unlimited
   - Recommendation: Add 1000 character limit
   - Rationale: Reasonable for descriptions

4. **Array Fields:** `capabilities`, `tools_enabled`, `context_sources`
   - Recommendation: Add array length limits (e.g., 20 items max)
   - Rationale: Prevents abuse, ensures performance

---

## Migration Path

### Adding New Validation Rules

1. **Update Handler:**
   ```go
   // In CreateCustomAgent or UpdateCustomAgent
   if len(req.SystemPrompt) > 10000 {
       c.JSON(http.StatusBadRequest, gin.H{
           "error": "System prompt exceeds 10000 characters",
       })
       return
   }
   ```

2. **Update Documentation:** Add to this file

3. **Update Tests:** Add test case to `test-agent-validation.sh`

4. **Update Frontend:** Add client-side validation

5. **Announce Changes:** Update API changelog

---

## Files Modified

| File | Change |
|------|--------|
| `internal/handlers/agents.go` | Added validation to CreateCustomAgent and UpdateCustomAgent |
| `internal/database/queries/custom_agents.sql` | Added CountUserAgents query |
| `internal/database/sqlc/*.go` | Regenerated after query addition |
| `scripts/tests/test-agent-validation.sh` | NEW - Validation test suite |
| `docs/CUSTOM_AGENTS_VALIDATION.md` | NEW - This documentation |

---

## Changelog

### 2026-01-11 - Initial Validation Implementation

**Added:**
- Rate limiting: 100 agents per user
- Suggested prompts: max 10, 500 chars each, no empty
- Welcome message: max 2000 chars
- Category: whitelist validation
- CountUserAgents SQLC query
- Comprehensive test suite
- Documentation

**Modified:**
- `CreateCustomAgent` handler
- `UpdateCustomAgent` handler
- SQLC queries and generated code

**Testing:**
- Automated test script created
- Manual test examples provided

---

## References

- **Handler Implementation:** `desktop/backend-go/internal/handlers/agents.go`
- **SQLC Queries:** `desktop/backend-go/internal/database/queries/custom_agents.sql`
- **Test Script:** `desktop/backend-go/scripts/tests/test-agent-validation.sh`
- **API Routes:** Defined in `desktop/backend-go/cmd/server/main.go` (or router file)

---

## Support

For questions or issues with validation:
1. Check error messages for specific validation failures
2. Review this documentation for validation rules
3. Run test script to verify expected behavior
4. Check server logs for detailed error context
