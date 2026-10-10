package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/rhl/businessos-backend/internal/services"
)

// conversationSummaryJobConfig holds all parameters for the conversation summary background job.
type conversationSummaryJobConfig struct {
	interval    time.Duration
	batchSize   int
	maxMessages int
}

// startConversationSummaryJob starts the optional background job that keeps
// conversation_summaries fresh for context + semantic search.
// All closure variables are passed as explicit parameters.
func startConversationSummaryJob(ctx context.Context, svc *services.ConversationIntelligenceService, cfg conversationSummaryJobConfig) {
	go func() {
		t := time.NewTicker(cfg.interval)
		defer t.Stop()
		slog.Info("Conversation summary job enabled",
			"interval", cfg.interval, "batch", cfg.batchSize, "max_messages", cfg.maxMessages)
		for {
			select {
			case <-t.C:
				count, err := svc.BackfillStaleSummaries(ctx, cfg.batchSize, cfg.maxMessages, false)
				if err != nil {
					slog.Error("Conversation summary job error", "error", err)
					continue
				}
				if count > 0 {
					slog.Info("Conversation summary job updated conversations", "count", count)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// behaviorPatternsJobConfig holds all parameters for the behavior patterns background job.
type behaviorPatternsJobConfig struct {
	interval  time.Duration
	batchSize int
}

// startBehaviorPatternsJob starts the optional background job that detects behavior
// patterns and stores them as user_facts for explicit confirmation.
// All closure variables are passed as explicit parameters.
func startBehaviorPatternsJob(ctx context.Context, svc *services.LearningService, cfg behaviorPatternsJobConfig) {
	go func() {
		t := time.NewTicker(cfg.interval)
		defer t.Stop()
		slog.Info("Behavior patterns job enabled",
			"interval", cfg.interval, "user_batch", cfg.batchSize)
		for {
			select {
			case <-t.C:
				usersProcessed, factsUpserted, err := svc.BackfillRecentUsersBehaviorPatterns(ctx, cfg.batchSize)
				if err != nil {
					slog.Error("Behavior patterns job error", "error", err)
					continue
				}
				if usersProcessed > 0 || factsUpserted > 0 {
					slog.Info("Behavior patterns job completed",
						"users_processed", usersProcessed, "facts_upserted", factsUpserted)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// appProfilerJobConfig holds all parameters for the app profiler sync background job.
type appProfilerJobConfig struct {
	interval  time.Duration
	batchSize int
}

// startAppProfilerSyncJob starts the optional background job that auto-syncs
// application profiles based on git HEAD or filesystem changes.
// All closure variables are passed as explicit parameters.
func startAppProfilerSyncJob(ctx context.Context, svc *services.AppProfilerService, cfg appProfilerJobConfig) {
	go func() {
		t := time.NewTicker(cfg.interval)
		defer t.Stop()
		slog.Info("App profiler auto-sync job enabled",
			"interval", cfg.interval, "batch", cfg.batchSize)
		for {
			select {
			case <-t.C:
				checked, refreshed, err := svc.SyncAutoProfiles(ctx, cfg.batchSize)
				if err != nil {
					slog.Error("App profiler auto-sync job error", "error", err)
					continue
				}
				if checked > 0 || refreshed > 0 {
					slog.Info("App profiler auto-sync completed",
						"profiles_checked", checked, "profiles_refreshed", refreshed)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
