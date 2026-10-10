package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rhl/businessos-backend/internal/database"
)

func main() {
	// Load .env for local development only.
	//
	// SECURITY: In production (ENVIRONMENT=production) the .env file must not exist
	// on the container filesystem. All secrets must be injected as real environment
	// variables (Cloud Run, Kubernetes secrets, etc.). We skip godotenv.Load()
	// entirely in production so there is no code path that could accidentally read
	// stale secrets from disk.
	//
	// In development: godotenv.Load() populates os.Getenv for keys not already set
	// in the shell, which is what config.Load() reads via viper.AutomaticEnv().
	if strings.ToLower(strings.TrimSpace(os.Getenv("ENVIRONMENT"))) != "production" {
		if err := godotenv.Load(); err != nil {
			slog.Debug("server: no .env file found; relying on environment variables",
				"error", err.Error(),
			)
		}
	}

	ctx := context.Background()

	// Bootstrap all services. Fatal on hard failures; degraded mode on soft ones.
	app, err := bootstrap(ctx)
	if err != nil {
		slog.Error("bootstrap failed", "error", err)
		os.Exit(1)
	}

	// Degraded-mode path: DB unavailable — only health + /api/status reachable.
	if !app.dbConnected || app.pool == nil {
		slog.Warn("server: running in degraded mode (no database); only /health, /ready, /api/status, and /uploads are available")
		go func() {
			slog.Info("server: starting in degraded mode", "port", app.cfg.ServerPort)
			srv := &http.Server{
				Addr:              os.Getenv("SERVER_HOST") + ":" + app.cfg.ServerPort, // SERVER_HOST=127.0.0.1 keeps it off the LAN
				Handler:           app.router,
				ReadHeaderTimeout: 10 * time.Second,
			}
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("server: failed to start (degraded mode)", "error", err)
				os.Exit(1)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("server: shutting down (degraded mode)")
		shutdownContainers(app)
		database.Close()
		slog.Info("server: stopped")
		return
	}

	// Full-mode: start HTTP server with graceful shutdown.
	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%s", os.Getenv("SERVER_HOST"), app.cfg.ServerPort),
		Handler:           app.router,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}
	go func() {
		slog.Info("server: starting", "port", app.cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server: listen failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("server: shutdown signal received")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("server: HTTP shutdown error", "error", err)
	}

	gracefulShutdown(app)

	slog.Info("server: stopped")
}

// shutdownContainers stops the container monitor and manager (used in both
// degraded-mode and full-mode shutdown paths).
func shutdownContainers(app *AppServices) {
	if app.containerMonitor != nil {
		slog.Info("server: stopping container monitor")
		if err := app.containerMonitor.StopMonitoring(); err != nil {
			slog.Warn("server: container monitor stop error", "error", err)
		}
	}
	if app.containerMgr != nil {
		slog.Info("server: shutting down container manager")
		app.containerMgr.Shutdown()
	}
}

// gracefulShutdown tears down all services in the correct order.
func gracefulShutdown(app *AppServices) {
	if app.osaQueueWorker != nil {
		slog.Info("server: stopping OSA queue worker")
		app.osaQueueWorker.Stop()
	}

	if app.batchWorker != nil {
		slog.Info("server: stopping notification batch worker")
		app.batchWorker.Stop()
	}

	if app.imageWarmerService != nil {
		slog.Info("server: stopping image warmer service")
		app.imageWarmerService.Stop()
	}

	if app.sandboxCleanupService != nil {
		slog.Info("server: stopping sandbox cleanup service")
		app.sandboxCleanupService.Stop()
	}

	if app.sandboxHealthMonitor != nil {
		slog.Info("server: stopping sandbox health monitor")
		app.sandboxHealthMonitor.Stop()
	}

	shutdownContainers(app)

	if app.jobScheduler != nil {
		slog.Info("server: stopping job scheduler")
		if err := app.jobScheduler.Stop(); err != nil {
			slog.Warn("server: job scheduler stop error", "error", err)
		}
	}

	for i, worker := range app.jobWorkers {
		if worker != nil && worker.IsRunning() {
			slog.Info("server: stopping job worker", "index", i+1)
			if err := worker.Stop(); err != nil {
				slog.Warn("server: job worker stop error", "index", i+1, "error", err)
			}
		}
	}

	if app.jobsHandler != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if count, err := app.jobsHandler.GetService().ReleaseStuckJobs(cleanupCtx); err == nil && count > 0 {
			slog.Info("server: released stuck jobs", "count", count)
		}
	}

	if app.sorxScheduler != nil {
		slog.Info("server: stopping SORX scheduler")
		app.sorxScheduler.Stop()
	}

	if app.proactiveConsumer != nil {
		slog.Info("server: stopping Optimal proactive consumer")
		if err := app.proactiveConsumer.Stop(); err != nil {
			slog.Warn("server: Optimal proactive consumer stop error", "error", err)
		}
	}

	if app.carrierClient != nil {
		slog.Info("server: closing CARRIER connection")
		if err := app.carrierClient.Close(); err != nil {
			slog.Warn("server: CARRIER close error", "error", err)
		}
	}

	database.Close()

	if app.sqlDB != nil {
		if err := app.sqlDB.Close(); err != nil {
			slog.Warn("server: sql.DB close error", "error", err)
		}
	}
}
