package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database"
	"github.com/rhl/businessos-backend/internal/database/schemahealth"
	"github.com/rhl/businessos-backend/internal/security"
)

type status string

const (
	statusOK   status = "ok"
	statusWarn status = "warn"
	statusFail status = "fail"
	statusSkip status = "skip"
)

type check struct {
	Name    string
	Status  status
	Message string
}

func main() {
	if strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT"))) != "production" {
		_ = godotenv.Load()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var checks []check
	fail := func(name string, format string, args ...any) {
		checks = append(checks, check{name, statusFail, fmt.Sprintf(format, args...)})
	}
	warn := func(name string, format string, args ...any) {
		checks = append(checks, check{name, statusWarn, fmt.Sprintf(format, args...)})
	}
	ok := func(name string, format string, args ...any) {
		checks = append(checks, check{name, statusOK, fmt.Sprintf(format, args...)})
	}
	skip := func(name string, format string, args ...any) {
		checks = append(checks, check{name, statusSkip, fmt.Sprintf(format, args...)})
	}

	cfg, err := config.Load()
	if err != nil {
		fail("config", "load failed: %v", err)
		printAndExit(checks)
	}
	ok("config", "loaded ENVIRONMENT=%s DEPLOYMENT_MODE=%s", cfg.Environment, cfg.DeploymentMode)

	if err := security.ValidateAndFail(cfg.Environment, cfg.SecretKey, cfg.TokenEncryptionKey, cfg.RedisKeyHMACSecret); err != nil {
		fail("security", "%v", err)
	} else {
		ok("security", "core secrets pass placeholder/length validation")
	}

	if cfg.IsProduction() {
		if err := cfg.Validate(); err != nil {
			fail("production config", "%v", err)
		} else {
			ok("production config", "required production settings are present")
		}
	} else {
		warn("production config", "ENVIRONMENT is not production; production-only checks were not enforced")
	}

	var pool *pgxpool.Pool
	if cfg.DatabaseRequired {
		p, err := database.Connect(cfg)
		if err != nil {
			fail("postgres", "connect failed: %v", err)
		} else {
			pool = p
			defer pool.Close()
			ok("postgres", "connected")
			checkDatabaseSchema(ctx, pool, ok, warn, fail)
		}
	} else {
		skip("postgres", "DATABASE_REQUIRED=false")
	}

	checkRedis(ctx, cfg, ok, warn, fail, skip)
	checkOptimalEngine(ctx, cfg, ok, warn, fail)
	checkOSA(ctx, cfg, ok, warn, fail, skip)

	printAndExit(checks)
}

func checkDatabaseSchema(
	ctx context.Context,
	pool *pgxpool.Pool,
	ok func(string, string, ...any),
	warn func(string, string, ...any),
	fail func(string, string, ...any),
) {
	report, err := schemahealth.Check(ctx, pool)
	if err != nil {
		fail("db schema", "%v", err)
		return
	}
	ok("db schema", "runtime schema contract passed (%d tables, %d column groups, %d functions)",
		len(schemahealth.RequiredTables),
		len(schemahealth.RequiredColumns),
		len(schemahealth.RequiredFunctions),
	)
	if !report.OK() {
		warn("db schema", "unexpected non-empty report without error: %s", report.Error())
	}
}

func checkRedis(
	ctx context.Context,
	cfg *config.Config,
	ok func(string, string, ...any),
	warn func(string, string, ...any),
	fail func(string, string, ...any),
	skip func(string, string, ...any),
) {
	if strings.TrimSpace(cfg.RedisURL) == "" {
		skip("redis", "REDIS_URL is blank")
		return
	}

	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		fail("redis", "invalid REDIS_URL: %v", err)
		return
	}
	if cfg.RedisPassword != "" {
		opts.Password = cfg.RedisPassword
	}
	if cfg.RedisTLSEnabled {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	client := redis.NewClient(opts)
	defer client.Close()

	start := time.Now()
	if err := client.Ping(ctx).Err(); err != nil {
		warn("redis", "ping failed: %v", err)
		return
	}
	ok("redis", "connected in %s", time.Since(start).Round(time.Millisecond))
}

func checkOptimalEngine(
	ctx context.Context,
	cfg *config.Config,
	ok func(string, string, ...any),
	warn func(string, string, ...any),
	fail func(string, string, ...any),
) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OPTIMAL_ENGINE_URL")), "/")
	if baseURL == "" {
		if cfg.IsProduction() {
			fail("optimal engine", "OPTIMAL_ENGINE_URL is required in production")
		} else {
			warn("optimal engine", "OPTIMAL_ENGINE_URL is blank; /api/optimal/* will return setup guidance")
		}
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/health", nil)
	if err != nil {
		fail("optimal engine", "invalid OPTIMAL_ENGINE_URL: %v", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		warn("optimal engine", "health check failed at %s: %v", baseURL, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		warn("optimal engine", "health check returned HTTP %d at %s", resp.StatusCode, baseURL)
		return
	}
	ok("optimal engine", "healthy at %s", baseURL)
}

func checkOSA(
	ctx context.Context,
	cfg *config.Config,
	ok func(string, string, ...any),
	warn func(string, string, ...any),
	fail func(string, string, ...any),
	skip func(string, string, ...any),
) {
	if !cfg.OSAEnabled {
		skip("osa runtime", "OSA_ENABLED=false")
		return
	}
	if cfg.OSA == nil {
		fail("osa runtime", "OSA_ENABLED=true but OSA config was not built")
		return
	}
	if err := cfg.OSA.Validate(); err != nil {
		fail("osa runtime", "%v", err)
		return
	}

	baseURL := strings.TrimRight(cfg.OSA.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		fail("osa runtime", "invalid OSA_BASE_URL: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.OSA.SharedSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		warn("osa runtime", "health check failed at %s: %v", baseURL, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		warn("osa runtime", "health check returned HTTP %d at %s", resp.StatusCode, baseURL)
		return
	}
	ok("osa runtime", "healthy at %s", baseURL)
}

func printAndExit(checks []check) {
	failed := 0
	warned := 0

	for _, c := range checks {
		switch c.Status {
		case statusOK:
			fmt.Printf("[ok]   %-20s %s\n", c.Name, c.Message)
		case statusWarn:
			warned++
			fmt.Printf("[warn] %-20s %s\n", c.Name, c.Message)
		case statusSkip:
			fmt.Printf("[skip] %-20s %s\n", c.Name, c.Message)
		case statusFail:
			failed++
			fmt.Printf("[fail] %-20s %s\n", c.Name, c.Message)
		}
	}

	fmt.Printf("\nsummary: %d failed, %d warnings\n", failed, warned)
	if failed > 0 {
		os.Exit(1)
	}
}
