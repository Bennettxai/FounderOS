# OSA Onboarding Integration

## Overview

This document describes the integration between the BusinessOS onboarding system and OSA (Orchestration System for Applications) to automatically generate a welcome workspace app when users complete onboarding.

## Architecture

```
User Completes Onboarding
    ↓
CompleteOnboarding() in OnboardingService
    ↓
Workspace Created Successfully
    ↓
[ASYNC] generateInitialWorkspaceApp() launched in goroutine
    ↓
buildWelcomeAppPrompt() creates customized prompt
    ↓
OSA Client generates app (30s timeout)
    ↓
Success: Log app_id and status
Failure: Log error (doesn't fail onboarding)
```

## Implementation Details

### 1. Auto-Trigger Logic

**File:** `internal/services/onboarding_service.go`
**Method:** `CompleteOnboarding()`
**Line:** ~701-710

When a user completes onboarding:
1. Workspace is created and committed to database
2. If `osaClient` is not nil, trigger app generation
3. Run in background goroutine (doesn't block response)
4. Parse userID to UUID
5. Call `generateInitialWorkspaceApp()` asynchronously

```go
// Trigger OSA app generation based on user profile (async, don't block onboarding)
if s.osaClient != nil {
    userUUID, uuidErr := uuid.Parse(userID)
    if uuidErr == nil {
        // Run in background goroutine so onboarding completes immediately
        go s.generateInitialWorkspaceApp(context.Background(), userUUID, workspace.ID, workspace.Name, extractedData, integrations)
    } else {
        slog.Warn("Failed to parse user ID for OSA generation", "user_id", userID, "error", uuidErr)
    }
}
```

### 2. App Generation Method

**Method:** `generateInitialWorkspaceApp()`
**Lines:** ~827-890

**Purpose:** Creates a welcome workspace app via OSA after onboarding completes.

**Key Features:**
- Runs asynchronously in background goroutine
- Has 30-second timeout
- Logs errors but doesn't fail onboarding flow
- Uses structured logging with `slog`
- Customizes app based on user preferences

**Parameters:**
- `ctx context.Context` - Context for cancellation
- `userID uuid.UUID` - User who completed onboarding
- `workspaceID uuid.UUID` - Newly created workspace
- `workspaceName string` - Workspace display name
- `extractedData ExtractedOnboardingData` - User preferences from onboarding
- `integrations []string` - Selected integrations

**OSA Request Structure:**
```go
&osa.AppGenerationRequest{
    UserID:      userID,
    WorkspaceID: workspaceID,
    Name:        "Welcome Workspace",
    Description: appDescription,
    Type:        "full-stack",
    Parameters: map[string]interface{}{
        "workspace_name": workspaceName,
        "business_type":  extractedData.BusinessType,
        "team_size":      extractedData.TeamSize,
        "role":           extractedData.Role,
        "challenge":      extractedData.Challenge,
        "integrations":   integrations,
        "prompt":         prompt,
    },
}
```

### 3. Prompt Builder

**Method:** `buildWelcomeAppPrompt()`
**Lines:** ~892-946

**Purpose:** Constructs a detailed, customized prompt for OSA based on onboarding preferences.

**Prompt Structure:**

1. **Header**
   ```
   Create a welcome workspace application for [WorkspaceName].
   ```

2. **Business Context**
   - Business Type (agency, startup, freelance, etc.)
   - Team Size (solo, 2-5, 6-10, etc.)
   - User Role (founder, developer, etc.)
   - Main Challenge (from onboarding)

3. **Requirements (Customized by Business Type)**
   - **Agency/Consulting:**
     - Client management features
     - Project tracking capabilities

   - **Startup:**
     - Product roadmap view
     - Team collaboration features

   - **Freelance:**
     - Time tracking features
     - Invoice/payment tracking

   - **Other:**
     - Task management features
     - Team overview

4. **Integrations**
   - Highlights selected integrations (Slack, Linear, etc.)

5. **Technical Stack**
   ```
   - Backend: Go with PostgreSQL
   - Frontend: Modern web UI with responsive design
   - Follow BusinessOS architecture patterns
   ```

**Example Prompt:**
```
Create a welcome workspace application for Acme Agency.

Business Context:
- Business Type: agency
- Team Size: 2-5
- User Role: founder
- Main Challenge: managing multiple client projects

Requirements:
- Create a simple, user-friendly dashboard
- Include getting started guide and quick actions
- Add client management features
- Include project tracking capabilities

Integrations to highlight: hubspot, slack, notion

Technical Stack:
- Backend: Go with PostgreSQL
- Frontend: Modern web UI with responsive design
- Follow BusinessOS architecture patterns
```

## Error Handling

### Non-Blocking Design

The OSA integration is designed to **never fail the onboarding flow**:

1. **OSA Client Disabled** (`osaClient == nil`)
   - Onboarding completes normally
   - No OSA calls attempted
   - No errors logged

2. **Invalid User ID** (UUID parse fails)
   - Warning logged with `slog.Warn`
   - Onboarding completes successfully
   - User can manually trigger app generation later

3. **OSA Generation Fails**
   - Error logged with `slog.Error` including:
     - Error message
     - User ID
     - Workspace ID
     - Workspace name
   - Onboarding completes successfully
   - User can retry via UI

4. **OSA Timeout** (>30 seconds)
   - Context timeout triggers
   - Error logged
   - Onboarding completes successfully

### Logging Standards

All logging uses **structured logging with `slog`**:

```go
// Success
slog.Info("Successfully triggered OSA app generation for new workspace",
    "user_id", userID,
    "workspace_id", workspaceID,
    "workspace_name", workspaceName,
    "app_id", resp.AppID,
    "status", resp.Status,
)

// Error
slog.Error("Failed to generate initial workspace app via OSA",
    "error", err,
    "user_id", userID,
    "workspace_id", workspaceID,
    "workspace_name", workspaceName,
)

// Warning
slog.Warn("Failed to parse user ID for OSA generation",
    "user_id", userID,
    "error", uuidErr,
)
```

## Configuration

### Environment Variables

```bash
# OSA Configuration (see internal/integrations/osa/config.go)
OSA_BASE_URL=http://localhost:8002          # OSA server URL
OSA_SHARED_SECRET=your-secret-here          # Shared secret for auth
OSA_TIMEOUT=30s                             # Request timeout
OSA_MAX_RETRIES=3                           # Retry attempts
OSA_RETRY_DELAY=1s                          # Delay between retries

# Feature Flag (optional)
OSA_AUTO_GENERATE_ENABLED=true              # Enable/disable auto-generation
```

### Service Initialization

**File:** `cmd/server/main.go` (or wherever services are initialized)

```go
// Create OSA client
osaConfig := &osa.Config{
    BaseURL:      os.Getenv("OSA_BASE_URL"),
    SharedSecret: os.Getenv("OSA_SHARED_SECRET"),
    Timeout:      30 * time.Second,
    MaxRetries:   3,
    RetryDelay:   1 * time.Second,
}

var osaClient *osa.ResilientClient
if osaConfig.BaseURL != "" {
    client, err := osa.NewClient(osaConfig)
    if err != nil {
        slog.Warn("Failed to create OSA client", "error", err)
    } else {
        osaClient = osa.NewResilientClient(client)
    }
}

// Pass to OnboardingService
onboardingService := services.NewOnboardingService(pool, aiService, osaClient)
```

## Testing

### Unit Tests

**File:** `tests/onboarding_osa_test.go`

Test coverage includes:
1. Prompt building for different business types
2. OSA request parameter validation
3. Success case handling
4. Error case handling
5. Async execution behavior
6. Timeout handling
7. Disabled OSA client behavior

### Mock OSA Client

A mock OSA client is provided for testing:

```go
type MockOSAClient struct {
    mock.Mock
}

func (m *MockOSAClient) GenerateApp(ctx context.Context, req *osa.AppGenerationRequest) (*osa.AppGenerationResponse, error) {
    args := m.Called(ctx, req)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*osa.AppGenerationResponse), args.Error(1)
}
```

### Integration Testing

**Manual Test:**
1. Complete onboarding flow
2. Check logs for OSA generation
3. Verify workspace has generated app
4. Test with OSA disabled
5. Test with invalid OSA URL

**Expected Logs:**
```
INFO Successfully triggered OSA app generation for new workspace
    user_id=uuid
    workspace_id=uuid
    workspace_name="Test Workspace"
    app_id="app_123"
    status="pending"
```

## Performance Considerations

### Async Execution

- App generation runs in background goroutine
- Onboarding API response returns immediately
- User doesn't wait for OSA to complete

### Timeout

- 30-second timeout prevents indefinite blocking
- Context cancellation ensures cleanup
- Timeout is configurable via `osaConfig.Timeout`

### Resource Usage

- Each onboarding spawns 1 goroutine
- Goroutine exits after OSA call (success or failure)
- No memory leaks or goroutine leaks

## Future Enhancements

### 1. Status Polling UI
Show real-time app generation progress in UI

### 2. Retry Mechanism
Auto-retry failed generations with exponential backoff

### 3. Template Variations
Multiple prompt templates based on use case

### 4. Analytics
Track success rate, generation time, user engagement

### 5. Webhooks
OSA sends completion webhook instead of polling

## Troubleshooting

### OSA Not Triggering

**Check:**
1. `osaClient` is initialized (not nil)
2. User ID is valid UUID format
3. OSA server is running and accessible
4. Logs show no errors during onboarding

**Logs to Search:**
```bash
grep "OSA" server.log
grep "generateInitialWorkspaceApp" server.log
```

### OSA Timeout

**Check:**
1. OSA server response time
2. Network latency
3. Timeout configuration (default: 30s)

**Solution:**
Increase timeout in `osaConfig.Timeout`

### Invalid Prompt

**Check:**
1. Onboarding data is complete
2. Business type is valid
3. Prompt builder logic matches expectations

**Debug:**
Add logging in `buildWelcomeAppPrompt()` to see generated prompt

## Related Documentation

- [OSA Client Documentation](../internal/integrations/osa/README.md)
- [Onboarding Flow](./ONBOARDING_FLOW.md)
- [Nick's Implementation Roadmap](../../NICK_IMPLEMENTATION_ROADMAP.md)

## Changelog

### 2026-01-25 - Initial Implementation
- Added `generateInitialWorkspaceApp()` method
- Added `buildWelcomeAppPrompt()` method
- Integrated with `CompleteOnboarding()`
- Created unit tests
- Added documentation

## Authors

- Nick's OSA Implementation (Task #57)
- BusinessOS Backend Team
