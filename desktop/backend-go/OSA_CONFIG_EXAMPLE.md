# OSA Configuration Example

Copy these environment variables to your `.env` file to enable OSA integration with onboarding:

```bash
# OSA Server Configuration
OSA_BASE_URL=http://localhost:8002
OSA_SHARED_SECRET=your-shared-secret-here
OSA_TIMEOUT=30s
OSA_MAX_RETRIES=3
OSA_RETRY_DELAY=1s

# Feature Flags
OSA_AUTO_GENERATE_ENABLED=true
```

## Testing Configurations

### Development (OSA Enabled)
```bash
OSA_BASE_URL=http://localhost:8002
OSA_SHARED_SECRET=dev-secret-123
OSA_AUTO_GENERATE_ENABLED=true
```

### Testing (OSA Disabled)
```bash
# Leave OSA_BASE_URL empty to disable
OSA_BASE_URL=
OSA_AUTO_GENERATE_ENABLED=false
```

### Production
```bash
OSA_BASE_URL=https://osa.yourdomain.com
OSA_SHARED_SECRET=prod-secret-from-vault
OSA_TIMEOUT=60s
OSA_MAX_RETRIES=5
OSA_RETRY_DELAY=2s
OSA_AUTO_GENERATE_ENABLED=true
```

## Verification

After adding to `.env`, restart the server and check logs:

```bash
# Should see OSA client initialization
grep "OSA" server.log

# Complete an onboarding and check for generation
grep "generateInitialWorkspaceApp" server.log
```

Success log should look like:
```
INFO Successfully triggered OSA app generation for new workspace
    user_id=550e8400-e29b-41d4-a716-446655440000
    workspace_id=660e8400-e29b-41d4-a716-446655440000
    workspace_name="Acme Corp"
    app_id="app_abc123"
    status="pending"
```
