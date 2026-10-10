# OSA-5 Integration Documentation Index

> **Complete documentation suite for the BusinessOS ↔ OSA-5 integration**
> Last Updated: January 9, 2026

---

## 📋 Quick Start

**New to OSA Integration?** Start here:

1. Read the [Setup Guide](#setup-guide) → Set up your environment
2. Review the [Architecture Overview](#architecture-overview) → Understand how it works
3. Follow the [API Testing Guide](#api-testing) → Test the endpoints
4. Check [Best Practices](#best-practices) → Implement production patterns

---

## 📚 Documentation Suite

### 🚀 Setup Guide

**File**: [`OSA_SETUP_GUIDE.md`](./OSA_SETUP_GUIDE.md) (25KB)

**What's Inside**:
- Prerequisites (Go, PostgreSQL, Redis, OSA-5)
- Database migration instructions
- Environment configuration
- Service startup procedures
- 8 comprehensive test scenarios
- Troubleshooting common errors
- Production deployment checklist

**When to Use**: First-time setup, onboarding new developers, deployment

---

### 🔌 API Testing Documentation

**File**: [`OSA_API_TESTS.md`](./OSA_API_TESTS.md) (26KB)

**What's Inside**:
- Authentication flows (JWT, OAuth)
- 9 complete endpoint specifications
- Request/response examples
- cURL command templates
- Automated test script
- Error scenarios and handling
- Integration testing checklist

**When to Use**: API integration, testing, debugging endpoint issues

**Quick Reference**: [`OSA_API_QUICK_REFERENCE.md`](./OSA_API_QUICK_REFERENCE.md) (3KB)

---

### 🏗️ Architecture Overview

**File**: [`OSA_ARCHITECTURE_FLOW.md`](./OSA_ARCHITECTURE_FLOW.md) (Generated)

**What's Inside**:
- Complete technical flow diagrams
- Database schema relationships
- Component interaction patterns
- File sync discovery process
- Webhook notification flow
- Module installation workflow

**When to Use**: Architectural decisions, code reviews, system understanding

---

### 📖 Best Practices Research

**File**: [`OSA_BEST_PRACTICES.md`](./OSA_BEST_PRACTICES.md) (Generated)

**What's Inside**:
- AI agent orchestration patterns
- Webhook security (HMAC, timing attacks, replay prevention)
- Real-time sync strategies (polling, event-driven, hybrid)
- Production error handling patterns
- File organization conventions
- Module installation patterns

**When to Use**: Code reviews, security audits, performance optimization

---

## 🎯 Use Case Guide

### Scenario: Setting Up OSA Integration (First Time)

1. **Prerequisites** → [Setup Guide](./OSA_SETUP_GUIDE.md#prerequisites)
2. **Database Setup** → [Setup Guide](./OSA_SETUP_GUIDE.md#database-setup)
3. **Configuration** → [Setup Guide](./OSA_SETUP_GUIDE.md#configuration)
4. **Start Services** → [Setup Guide](./OSA_SETUP_GUIDE.md#starting-services)
5. **Verify Setup** → [API Testing](./OSA_API_TESTS.md#health-check)

---

### Scenario: Testing API Endpoints

1. **Get Auth Token** → [API Testing](./OSA_API_TESTS.md#authentication)
2. **List Workflows** → [API Testing](./OSA_API_TESTS.md#get-apiosaworkflows)
3. **Get Files** → [API Testing](./OSA_API_TESTS.md#get-apiosaworkflowsidfiles)
4. **Install Module** → [API Testing](./OSA_API_TESTS.md#post-apiosamodulesinstall)

---

### Scenario: Debugging Issues

1. **Check Logs** → [Setup Guide](./OSA_SETUP_GUIDE.md#debugging-tips)
2. **Database State** → [API Testing](./OSA_API_TESTS.md#database-inspection)
3. **Common Errors** → [Setup Guide](./OSA_SETUP_GUIDE.md#troubleshooting)
4. **Error Patterns** → [Best Practices](./OSA_BEST_PRACTICES.md#error-handling)

---

### Scenario: Code Review / Architecture Decision

1. **Component Flow** → [Architecture](./OSA_ARCHITECTURE_FLOW.md#component-interaction)
2. **Security Patterns** → [Best Practices](./OSA_BEST_PRACTICES.md#webhook-security)
3. **Error Handling** → [Best Practices](./OSA_BEST_PRACTICES.md#production-error-handling)
4. **Performance** → [Architecture](./OSA_ARCHITECTURE_FLOW.md#performance-considerations)

---

## 📂 File Structure Reference

### Backend Go Code
```
desktop/backend-go/
├── cmd/
│   └── server/main.go               # Server initialization
├── internal/
│   ├── database/
│   │   └── migrations/
│   │       └── 042_osa_integration.sql  # Database schema
│   ├── handlers/
│   │   ├── osa_workflows.go         # API handlers (9 endpoints)
│   │   └── osa_webhooks.go          # Webhook receivers (2 endpoints)
│   └── services/
│       └── osa_file_sync.go         # Background polling service
└── docs/
    ├── OSA_INTEGRATION_INDEX.md     # This file (master index)
    ├── OSA_SETUP_GUIDE.md           # Setup and testing guide
    ├── OSA_API_TESTS.md             # API testing documentation
    ├── OSA_API_QUICK_REFERENCE.md   # Quick API reference
    ├── OSA_ARCHITECTURE_FLOW.md     # Technical architecture
    └── OSA_BEST_PRACTICES.md        # Best practices research
```

### Frontend Svelte Code
```
frontend/src/routes/
└── window/
    └── osa/
        ├── +page.svelte          # Workflow list UI
        └── [id]/
            └── +page.svelte      # File explorer UI
```

### OSA-5 Workspace
```
/Users/ososerious/OSA-5/miosa-backend/generated/
├── analysis/
│   └── analysis_<workflow_id>.md
├── architecture/
│   └── architecture_<workflow_id>.md
├── code/
│   └── code_<workflow_id>.go
├── quality/
│   └── quality_<workflow_id>.md
├── deployment/
│   └── deployment_<workflow_id>.md
├── monitoring/
│   └── monitoring_<workflow_id>.md
├── strategy/
│   └── strategy_<workflow_id>.md
└── recommendations/
    └── recommendations_<workflow_id>.md
```

---

## 🔄 Data Flow Summary

### 1. Workflow Generation
```
User → OSA-5 CLI
  ↓
OSA-5 generates files
  ↓
Files written to /generated/ directory
  ↓
OSAFileSyncService polls (30s)
  ↓
Files parsed and grouped by workflow_id
  ↓
PostgreSQL: osa_generated_apps table
  ↓
Frontend: /window/osa displays workflows
```

### 2. Workflow Query
```
Frontend → GET /api/osa/workflows
  ↓
Auth middleware validates JWT
  ↓
Handler queries osa_generated_apps
  ↓
Joins with osa_workspaces
  ↓
Returns JSON with workflow list
```

### 3. Module Installation
```
Frontend → POST /api/osa/modules/install
  ↓
Handler extracts metadata JSONB
  ↓
Creates entry in osa_modules table
  ↓
Links module_id back to osa_generated_apps
  ↓
Updates status to 'deployed'
  ↓
Returns success with module_id
```

---

## 🗄️ Database Schema Overview

### Core Tables
```
user (BetterAuth)
  ↓ user_id FK (VARCHAR)
osa_workspaces
  ↓ workspace_id FK
osa_generated_apps ★ (Main table)
  ├─ id (UUID)
  ├─ workspace_id (FK)
  ├─ module_id (FK to osa_modules)
  ├─ osa_workflow_id (8-char string)
  ├─ metadata (JSONB - all file contents)
  └─ status (generating|generated|deployed|failed)
```

### Supporting Tables
```
osa_modules         - Installed BusinessOS modules
osa_build_events    - Real-time build progress
osa_sync_status     - Sync tracking
osa_webhooks        - Webhook subscriptions
osa_execution_history - Workflow execution logs
```

**Key**: ★ = Primary data storage

---

## 🔐 Security Considerations

### Authentication
- **User Endpoints**: JWT Bearer tokens (from sessions table)
- **Webhook Endpoints**: HMAC-SHA256 signature verification
- **Internal Endpoints**: X-User-ID header (container context)

### HMAC Verification
```bash
# Generate HMAC signature
echo -n '{"event_type":"workflow.completed"}' | \
  openssl dgst -sha256 -hmac "your-webhook-secret" | \
  awk '{print $2}'
```

### Environment Variables
```bash
# Required
DATABASE_URL=postgresql://...
SECRET_KEY=your-jwt-secret

# OSA-specific
OSA_ENABLED=true
OSA_BASE_URL=http://localhost:3003
OSA_WORKSPACE_PATH=/Users/ososerious/OSA-5/miosa-backend/generated
OSA_WEBHOOK_SECRET=your-webhook-hmac-secret
```

---

## 📊 API Endpoints Quick Reference

### Workflow Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/osa/workflows` | List all workflows |
| GET | `/api/osa/workflows/:id` | Get workflow details |
| GET | `/api/osa/workflows/:id/files` | List workflow files |
| GET | `/api/osa/workflows/:id/files/:type` | Get file by type |
| GET | `/api/osa/files/:id/content` | Get file by ID |

### Module & Sync
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/osa/modules/install` | Install as module |
| POST | `/api/osa/sync/trigger` | Manual sync |

### Webhooks
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/osa/webhooks` | List webhooks |
| POST | `/api/osa/webhooks/register` | Register webhook |
| POST | `/api/osa/webhooks/workflow-complete` | OSA-5 callback |
| POST | `/api/osa/webhooks/build-event` | OSA-5 build event |

---

## 🧪 Testing Checklist

### Initial Setup
- [ ] PostgreSQL running and accessible
- [ ] Migration 042 applied successfully
- [ ] Default workspace created for test user
- [ ] Environment variables configured

### Backend Testing
- [ ] Server starts without errors
- [ ] Health check endpoint returns 200
- [ ] File sync service polling logs visible
- [ ] Database tables populated correctly

### API Testing
- [ ] Authentication successful (JWT token obtained)
- [ ] List workflows returns 200
- [ ] Get workflow details returns 200
- [ ] List files returns deterministic UUIDs
- [ ] Get file content returns markdown/code
- [ ] Module installation succeeds

### Frontend Testing
- [ ] `/window/osa` displays workflow cards
- [ ] Workflow cards show correct status
- [ ] Click into workflow shows file explorer
- [ ] File preview renders markdown
- [ ] Install button clickable
- [ ] No console errors

### Integration Testing
- [ ] Generate workflow in OSA-5
- [ ] Wait 30-60 seconds for sync
- [ ] Refresh frontend, see new workflow
- [ ] Click workflow, see all 8 files
- [ ] Install as module
- [ ] Verify osa_modules table entry

---

## 🐛 Common Issues & Solutions

### Issue: "No workflows found"
**Solution**:
1. Check OSA workspace path is correct
2. Verify files exist in `/generated/` directory
3. Check file naming matches pattern (e.g., `analysis_11af0132.md`)
4. Review file sync service logs for errors

### Issue: "Webhook signature invalid"
**Solution**:
1. Verify `OSA_WEBHOOK_SECRET` matches on both systems
2. Check signature is HMAC-SHA256 hex encoded
3. Ensure body is identical to what was signed
4. Set `OSA_WEBHOOK_DEV_MODE=true` to bypass (dev only)

### Issue: "Module installation fails"
**Solution**:
1. Verify workflow status is `generated` or `deployed`
2. Check metadata JSONB has required keys
3. Ensure user has permission to workspace
4. Review database logs for constraint violations

---

## 🚀 Deployment Checklist

### Development
- [ ] Use `.env` file for configuration
- [ ] Enable debug logging
- [ ] Allow unsigned webhooks (dev mode)
- [ ] Use short polling interval (5-10s)

### Staging
- [ ] Move secrets to environment variables
- [ ] Enable HMAC webhook verification
- [ ] Use moderate polling interval (30s)
- [ ] Test with production-like data

### Production
- [ ] All secrets in secure vault (AWS Secrets Manager)
- [ ] Strict HMAC verification enabled
- [ ] Adaptive polling (5-60s based on activity)
- [ ] Metrics and alerting configured
- [ ] HTTPS enforced
- [ ] Rate limiting enabled
- [ ] Circuit breakers configured
- [ ] Backup and recovery tested

---

## 📈 Performance Benchmarks

### Expected Response Times
| Endpoint | p50 | p95 | p99 |
|----------|-----|-----|-----|
| List workflows | 50ms | 150ms | 300ms |
| Get workflow | 30ms | 100ms | 200ms |
| List files | 40ms | 120ms | 250ms |
| Get file content | 20ms | 80ms | 150ms |
| Install module | 200ms | 500ms | 1000ms |

### File Sync Performance
- **Polling interval**: 30 seconds (default)
- **Scan time**: <100ms for 100 workflows
- **Memory overhead**: ~5MB per 1000 workflows

---

## 🔍 Debugging Commands

### Check Database State
```sql
-- List all workflows
SELECT id, name, osa_workflow_id, status, files_created
FROM osa_generated_apps
ORDER BY created_at DESC;

-- Check sync status
SELECT * FROM osa_sync_status
WHERE entity_type = 'app'
ORDER BY last_sync_at DESC;

-- View build events
SELECT app_id, event_type, phase, progress_percent, status_message
FROM osa_build_events
ORDER BY created_at DESC;
```

### View Logs
```bash
# Backend logs (structured JSON)
tail -f /var/log/businessos/backend.log

# File sync logs
grep "OSAFileSync" /var/log/businessos/backend.log

# Webhook logs
grep "OSAWebhooks" /var/log/businessos/backend.log
```

### Test Connectivity
```bash
# Health check
curl http://localhost:8080/api/osa/health

# Database connection
psql $DATABASE_URL -c "SELECT 1"

# OSA-5 connectivity
curl http://localhost:3003/health
```

---

## 📚 Additional Resources

### Official Documentation
- Go Documentation: https://go.dev/doc/
- PostgreSQL: https://www.postgresql.org/docs/
- Gin Framework: https://gin-gonic.com/docs/

### Related Projects
- BetterAuth: https://www.better-auth.com/
- SQLC: https://docs.sqlc.dev/
- pgx: https://github.com/jackc/pgx

### Internal Documentation
- BusinessOS Architecture: `../ARCHITECTURE.md`
- Database Schema: `../internal/database/schema.sql`
- API Changelog: `../CHANGELOG.md`

---

## 🤝 Contributing

### Making Changes
1. Read relevant documentation above
2. Make code changes
3. Update affected documentation
4. Run tests: `go test ./internal/handlers/ -v`
5. Update this index if adding new docs

### Adding New Endpoints
1. Add handler to `internal/handlers/osa_workflows.go`
2. Register route in `internal/handlers/handlers.go`
3. Document in `OSA_API_TESTS.md`
4. Add test case
5. Update this index

### Reporting Issues
Include:
- What you expected to happen
- What actually happened
- Relevant logs (redact secrets)
- Steps to reproduce
- Environment (dev/staging/production)

---

## 📞 Support

### Questions?
- Check [Setup Guide](./OSA_SETUP_GUIDE.md) troubleshooting section
- Review [API Testing](./OSA_API_TESTS.md) error scenarios
- Search [Best Practices](./OSA_BEST_PRACTICES.md) for patterns

### Need Help?
1. Check logs for error messages
2. Verify environment configuration
3. Review database state
4. Test with curl commands
5. Contact team if still stuck

---

## ✅ Document Status

| Document | Status | Last Updated | Size |
|----------|--------|--------------|------|
| OSA_INTEGRATION_INDEX.md | ✅ Complete | 2026-01-09 | 15KB |
| OSA_SETUP_GUIDE.md | ✅ Complete | 2026-01-09 | 25KB |
| OSA_API_TESTS.md | ✅ Complete | 2026-01-09 | 26KB |
| OSA_API_QUICK_REFERENCE.md | ✅ Complete | 2026-01-09 | 3KB |
| OSA_ARCHITECTURE_FLOW.md | 📝 Generated | 2026-01-09 | N/A |
| OSA_BEST_PRACTICES.md | 📝 Generated | 2026-01-09 | N/A |

**Total Documentation**: ~70KB of comprehensive guides

---

## 🎯 Quick Commands

### Start Everything
```bash
# Terminal 1: Backend
cd ~/BusinessOS-1/desktop/backend-go
go run ./cmd/server

# Terminal 2: Frontend
cd ~/BusinessOS-1/frontend
npm run dev

# Terminal 3: OSA-5
cd ~/OSA-5/miosa-backend
npm run dev
```

### Test End-to-End
```bash
# Generate workflow
osa generate todo app

# Wait 30 seconds

# List workflows
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/osa/workflows

# View in browser
open http://localhost:5173/window/osa
```

---

**Last Updated**: January 9, 2026
**Version**: 1.0.0
**Maintainers**: BusinessOS Team
