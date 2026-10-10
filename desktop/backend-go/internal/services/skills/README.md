# Skills Module

The Skills Module provides a registry system for executable capabilities that can be registered and invoked by different parts of the BusinessOS system, particularly by the Chain of Thought (CoT) agent.

## Overview

Skills are self-contained units of functionality that:
- Have a unique name and description
- Accept typed parameters (JSON-serializable)
- Return typed results
- Include JSON schema for validation
- Can be executed through the registry

## Architecture

```
┌─────────────────┐
│  CoT Agent      │
│  (Orchestrator) │
└────────┬────────┘
         │
         │ Execute skill
         v
┌─────────────────┐
│  Skill Registry │
│  - Register     │
│  - Get          │
│  - Execute      │
└────────┬────────┘
         │
         │ Dispatch
         v
┌─────────────────┐
│  Skill (OSA)    │
│  - Validate     │
│  - Execute      │
│  - Return       │
└─────────────────┘
```

## Components

### 1. Skill Interface (`types.go`)

The core interface that all skills must implement:

```go
type Skill interface {
    Name() string
    Description() string
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
    Schema() *SkillSchema
}
```

### 2. Registry (`registry.go`)

Manages the collection of skills:

```go
registry := skills.NewRegistry()

// Register a skill
err := registry.Register(osaSkill)

// Get a skill
skill, err := registry.Get("osa_orchestrate")

// List all skills
allSkills := registry.List()

// Execute a skill
result, err := registry.Execute(ctx, "osa_orchestrate", params)
```

### 3. OSA Skill (`osa_skill.go`)

The OSA orchestration skill implementation:

```go
skill := skills.NewOsaSkill(resilientClient)

params := map[string]interface{}{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "input": "Create a task management app",
    "workspace_id": "660e8400-e29b-41d4-a716-446655440001",
    "phase": "analysis",
    "context": map[string]interface{}{
        "key": "value",
    },
}

result, err := skill.Execute(ctx, params)
```

## Integration

### In `main.go` or setup:

```go
import (
    "businessos/internal/integrations/osa"
    "businessos/internal/services/skills"
)

func main() {
    // 1. Create OSA resilient client
    osaConfig := osa.DefaultResilientClientConfig()
    osaConfig.OSAConfig.BaseURL = os.Getenv("OSA_URL")
    osaConfig.OSAConfig.SharedSecret = os.Getenv("OSA_SECRET")

    osaClient, err := osa.NewResilientClient(osaConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer osaClient.Close()

    // 2. Initialize skill registry
    registry, err := skills.InitializeSkillRegistry(osaClient)
    if err != nil {
        log.Fatal(err)
    }

    // 3. Set up API routes (optional)
    router := gin.Default()
    api := router.Group("/api/v1")
    skills.SetupSkillRoutes(api, registry)

    // 4. Pass registry to CoT agent
    cotAgent := NewCoTAgent(registry)

    router.Run(":8080")
}
```

### In CoT Agent:

```go
type CoTAgent struct {
    skillRegistry *skills.Registry
}

func (a *CoTAgent) ProcessTask(ctx context.Context, task string) error {
    // Decide which skill to use
    skillName := "osa_orchestrate"

    // Prepare parameters
    params := map[string]interface{}{
        "user_id": a.userID.String(),
        "input": task,
    }

    // Execute skill
    result, err := a.skillRegistry.Execute(ctx, skillName, params)
    if err != nil {
        return fmt.Errorf("skill execution failed: %w", err)
    }

    // Process result
    return a.processResult(result)
}
```

## Creating New Skills

To create a new skill:

1. Implement the `Skill` interface:

```go
type MyCustomSkill struct {
    // dependencies
}

func (s *MyCustomSkill) Name() string {
    return "my_custom_skill"
}

func (s *MyCustomSkill) Description() string {
    return "Does something useful"
}

func (s *MyCustomSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Validate parameters
    input, ok := params["input"].(string)
    if !ok {
        return nil, fmt.Errorf("input is required")
    }

    // Execute logic
    result := s.doWork(ctx, input)

    return result, nil
}

func (s *MyCustomSkill) Schema() *SkillSchema {
    return &SkillSchema{
        InputSchema: json.RawMessage(`{
            "type": "object",
            "required": ["input"],
            "properties": {
                "input": {"type": "string"}
            }
        }`),
    }
}
```

2. Register it:

```go
customSkill := NewMyCustomSkill(deps)
err := registry.Register(customSkill)
```

## Testing

Run tests:

```bash
cd internal/services/skills
go test -v
go test -race -v  # Check for race conditions
go test -cover    # Check coverage
```

## API Endpoints

If using the example handler:

### List all skills
```
GET /api/v1/skills
```

Response:
```json
{
  "skills": [
    {
      "name": "osa_orchestrate",
      "description": "Triggers the full 21-agent OSA orchestration workflow",
      "schema": {...}
    }
  ],
  "count": 1
}
```

### Execute a skill
```
POST /api/v1/skills/execute
Content-Type: application/json

{
  "skill_name": "osa_orchestrate",
  "parameters": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "input": "Create a task management app"
  }
}
```

Response:
```json
{
  "skill_name": "osa_orchestrate",
  "success": true,
  "result": {
    "success": true,
    "output": "Generated task management app...",
    "agents_used": ["architect", "backend-specialist"],
    "execution_ms": 15000
  }
}
```

### Get skill schema
```
GET /api/v1/skills/osa_orchestrate/schema
```

## Error Handling

All skills follow BusinessOS error handling patterns:

- Context propagation for cancellation
- Wrapped errors with context: `fmt.Errorf("description: %w", err)`
- Structured logging with `slog`
- No panics in production code

## Performance Considerations

- Registry uses read-write locks for concurrent access
- Skills are long-lived (registered once at startup)
- Execution can be concurrent (context per execution)
- OSA skill uses ResilientClient with circuit breaker and retry

## Security

- Skills validate all parameters before execution
- User ID is required for all operations
- Context carries authentication/authorization
- OSA skill uses authenticated client with shared secret

## Future Enhancements

Potential additions:

1. **Skill Middleware**: Add hooks for logging, metrics, rate limiting
2. **Skill Versioning**: Support multiple versions of the same skill
3. **Skill Discovery**: Auto-discover and register skills
4. **Skill Composition**: Allow skills to call other skills
5. **Async Execution**: Support long-running skills with callbacks
6. **Skill Marketplace**: Share and import community skills

## Related Documentation

- [OSA Integration](../../../integrations/osa/README.md)
- [CoT Agent](../../orchestration/README.md)
- [Handler Patterns](../../handlers/README.md)
