// Package testutil provides testing utilities for integration tests.
//
// This package includes helpers for setting up test databases and Redis instances
// using PostgreSQL (via environment variable) and miniredis for Redis.
//
// # Database Testing
//
// Use SetupTestDatabase to connect to a test PostgreSQL database with migrations:
//
//	func TestMyFeature(t *testing.T) {
//		db := testutil.RequireTestDatabase(t)
//		defer db.Close()
//
//		// Use db.Pool for database operations
//		service := NewMyService(db.Pool)
//		result, err := service.DoSomething(ctx)
//		require.NoError(t, err)
//	}
//
// # Redis Testing
//
// Use SetupTestRedis to create an in-memory Redis instance:
//
//	func TestCaching(t *testing.T) {
//		redis := testutil.RequireTestRedis(t)
//		defer redis.Close()
//
//		// Use redis.Client for Redis operations
//		cache := NewCacheService(redis.Client, nil) // nil logger uses slog.Default()
//		err := cache.Set(ctx, "key", "value")
//		require.NoError(t, err)
//	}
//
// # Running Integration Tests
//
// Integration tests are automatically skipped when running with -short flag:
//
//	go test -short ./...     # Skip integration tests
//	go test ./...            # Run all tests including integration
//
// # Environment Configuration
//
// Set TEST_DATABASE_URL to point to your test database:
//
//	export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/test_db?sslmode=disable"
//
// If not set, defaults to: postgres://postgres:postgres@localhost:5432/businessos_test?sslmode=disable
//
// Redis tests use in-memory miniredis and don't require any configuration.
//
// # Migration Management
//
// The SetupTestDatabase function automatically applies all migrations from
// internal/database/migrations/ in alphabetical order. Ensure your migration
// files are properly numbered (e.g., 001_initial.sql, 002_add_users.sql).
//
// # CI/CD Integration
//
// In CI/CD pipelines, you can use a PostgreSQL service container:
//
//	services:
//	  postgres:
//	    image: postgres:16-alpine
//	    env:
//	      POSTGRES_PASSWORD: postgres
//	      POSTGRES_DB: businessos_test
//
// Then set TEST_DATABASE_URL in your CI configuration.
package testutil
