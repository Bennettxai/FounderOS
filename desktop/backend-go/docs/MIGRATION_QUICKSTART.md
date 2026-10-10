# Migration Quick Start Guide

**For developers who need to test migrations 052-054, 088-089 quickly.**

---

## 🚀 Quick Commands

### Test Locally (Recommended First Step)

```bash
cd desktop/backend-go/scripts/migrations
./test_migrations.sh
```

This will:
- Create test database
- Apply all migrations
- Verify schema integrity
- Test CRUD operations
- Clean up automatically

### Test Specific Migration

```bash
./test_migrations.sh --specific 052
```

### Test Rollback

```bash
./test_migrations.sh --rollback
```

### Test on Staging

```bash
./test_migrations.sh --staging
```

**⚠️ WARNING**: Only run on staging if you have backup and approval!

---

## 📋 What These Migrations Do

| Migration | Purpose | Tables Created |
|-----------|---------|----------------|
| **052** | Workspace snapshots | `workspace_versions` |
| **053** | Onboarding AI system | `onboarding_sessions`, `onboarding_email_metadata`, `onboarding_conversation_history` |
| **054** | Custom modules marketplace | `custom_modules`, `custom_module_versions`, `custom_module_installations`, `custom_module_shares` |
| **089** | App generation system | `app_templates`, `user_generated_apps`, `app_generation_queue` |
| **088** | Seed app templates | Data insert (5 templates) |

**Important**: Apply in order: 052 → 053 → 054 → 089 → 088

---

## ✅ Pre-Flight Checklist

Before deploying:

- [ ] Ran `./test_migrations.sh` successfully
- [ ] Ran `sqlc generate` (regenerate queries)
- [ ] Ran `go build ./...` (verify compiles)
- [ ] Tested rollback locally
- [ ] Created database backup

---

## 🔧 Common Issues

### "relation does not exist"
**Solution**: Apply migrations in correct order (052 → 053 → 054 → 089 → 088)

### "column does not exist" (in Go code)
**Solution**: Run `sqlc generate` to regenerate database code

### "foreign key violation"
**Solution**: Ensure `workspaces` and `"user"` tables exist

### "permission denied"
**Solution**: Make script executable with `chmod +x test_migrations.sh`

---

## 📚 Full Documentation

- **Detailed Guide**: [MIGRATIONS.md](./MIGRATIONS.md)
- **Staging Plan**: [STAGING_MIGRATION_PLAN.md](./STAGING_MIGRATION_PLAN.md)
- **Test Script**: [scripts/migrations/test_migrations.sh](../scripts/migrations/test_migrations.sh)

---

## 🆘 Need Help?

1. Check logs: `tail -f /var/log/postgresql/*.log`
2. Review [MIGRATIONS.md](./MIGRATIONS.md) troubleshooting section
3. Contact database team: #database-migrations

---

**Last Updated**: 2026-01-26
