# Production Monitoring & Observability

## Overview

Comprehensive monitoring setup for BusinessOS using GCP Cloud Monitoring, structured logging, and health checks.

## Table of Contents

1. [Structured Logging](#structured-logging)
2. [Cloud Monitoring Dashboard](#cloud-monitoring-dashboard)
3. [Alert Policies](#alert-policies)
4. [Uptime Monitoring](#uptime-monitoring)
5. [Performance Metrics](#performance-metrics)

---

## Structured Logging

### Backend: slog Configuration

The backend already uses `slog` for structured logging. Here's the production configuration:

#### Update `cmd/server/main.go`:

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"cloud.google.com/go/logging"
)

func setupLogger(environment string) {
	var handler slog.Handler

	if environment == "production" {
		// Production: JSON format for Cloud Logging
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Add severity mapping for Cloud Logging
				if a.Key == slog.LevelKey {
					level := a.Value.Any().(slog.Level)
					switch level {
					case slog.LevelDebug:
						a.Value = slog.StringValue("DEBUG")
					case slog.LevelInfo:
						a.Value = slog.StringValue("INFO")
					case slog.LevelWarn:
						a.Value = slog.StringValue("WARNING")
					case slog.LevelError:
						a.Value = slog.StringValue("ERROR")
					}
				}
				return a
			},
		})
	} else {
		// Development: Human-readable format
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Logger initialized",
		"environment", environment,
		"level", handler.Enabled(context.Background(), slog.LevelDebug),
	)
}

func main() {
	environment := os.Getenv("ENVIRONMENT")
	setupLogger(environment)

	// Rest of initialization...
}
```

### Request Logging Middleware

Add request ID and user ID to all logs:

```go
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLogger logs all HTTP requests with structured data
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get user ID if authenticated
		userID := ""
		if user, exists := c.Get("user"); exists {
			if u, ok := user.(map[string]interface{}); ok {
				if id, ok := u["id"].(string); ok {
					userID = id
				}
			}
		}

		// Log request
		slog.Info("HTTP request",
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"user_id", userID,
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		)

		// Log errors separately
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				slog.Error("Request error",
					"request_id", requestID,
					"error", err.Error(),
				)
			}
		}
	}
}
```

### Log Critical Events

```go
// Database connection issues
slog.Error("Database connection failed",
	"error", err.Error(),
	"connection_string", redactedURL,
)

// Slow queries
slog.Warn("Slow database query",
	"query", "GetUserWorkspaces",
	"duration_ms", duration.Milliseconds(),
	"user_id", userID,
)

// Failed authentication attempts
slog.Warn("Authentication failed",
	"ip", ip,
	"email", sanitizedEmail,
	"reason", "invalid_password",
)

// Background job failures
slog.Error("Background job failed",
	"job_id", jobID,
	"job_type", jobType,
	"error", err.Error(),
	"retry_count", retryCount,
)
```

---

## Cloud Monitoring Dashboard

### 1. Create Dashboard via Terraform

Create file: `infrastructure/monitoring/dashboard.tf`

```hcl
resource "google_monitoring_dashboard" "businessos_backend" {
  dashboard_json = jsonencode({
    displayName = "BusinessOS Backend"
    mosaicLayout = {
      columns = 12
      tiles = [
        # Request Rate
        {
          width  = 6
          height = 4
          widget = {
            title = "HTTP Request Rate"
            xyChart = {
              dataSets = [{
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"cloud_run_revision\" AND metric.type=\"run.googleapis.com/request_count\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_RATE"
                      crossSeriesReducer = "REDUCE_SUM"
                    }
                  }
                }
              }]
            }
          }
        },
        # Error Rate
        {
          xPos   = 6
          width  = 6
          height = 4
          widget = {
            title = "HTTP Error Rate (5xx)"
            xyChart = {
              dataSets = [{
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"cloud_run_revision\" AND metric.type=\"run.googleapis.com/request_count\" AND metric.label.response_code_class=\"5xx\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_RATE"
                      crossSeriesReducer = "REDUCE_SUM"
                    }
                  }
                }
              }]
            }
          }
        },
        # Latency
        {
          yPos   = 4
          width  = 12
          height = 4
          widget = {
            title = "Request Latency (p50, p95, p99)"
            xyChart = {
              dataSets = [
                {
                  timeSeriesQuery = {
                    timeSeriesFilter = {
                      filter = "resource.type=\"cloud_run_revision\" AND metric.type=\"run.googleapis.com/request_latencies\""
                      aggregation = {
                        alignmentPeriod    = "60s"
                        perSeriesAligner   = "ALIGN_DELTA"
                        crossSeriesReducer = "REDUCE_PERCENTILE_50"
                      }
                    }
                  }
                  plotType = "LINE"
                },
                # Add p95 and p99 similarly
              ]
            }
          }
        },
        # Database Connections
        {
          yPos   = 8
          width  = 6
          height = 4
          widget = {
            title = "Database Connections"
            xyChart = {
              dataSets = [{
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"cloudsql_database\" AND metric.type=\"cloudsql.googleapis.com/database/network/connections\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_MEAN"
                    }
                  }
                }
              }]
            }
          }
        },
        # Redis Memory Usage
        {
          xPos   = 6
          yPos   = 8
          width  = 6
          height = 4
          widget = {
            title = "Redis Memory Usage"
            xyChart = {
              dataSets = [{
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"redis_instance\" AND metric.type=\"redis.googleapis.com/stats/memory/usage_ratio\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_MEAN"
                    }
                  }
                }
              }]
            }
          }
        }
      ]
    }
  })
}
```

### 2. Create Dashboard Manually (GCP Console)

1. Go to [Cloud Monitoring](https://console.cloud.google.com/monitoring)
2. Click **Dashboards** > **Create Dashboard**
3. Name: "BusinessOS Backend"
4. Add charts:

**Chart 1: Request Rate**
- Resource type: Cloud Run Revision
- Metric: `run.googleapis.com/request_count`
- Aggregation: Rate (1 minute)
- Reducer: Sum

**Chart 2: Error Rate**
- Resource type: Cloud Run Revision
- Metric: `run.googleapis.com/request_count`
- Filter: `response_code_class = 5xx`
- Aggregation: Rate (1 minute)

**Chart 3: Latency Percentiles**
- Resource type: Cloud Run Revision
- Metric: `run.googleapis.com/request_latencies`
- Aggregation: Percentiles (50th, 95th, 99th)

**Chart 4: Container Instance Count**
- Resource type: Cloud Run Revision
- Metric: `run.googleapis.com/container/instance_count`

**Chart 5: Memory Usage**
- Resource type: Cloud Run Revision
- Metric: `run.googleapis.com/container/memory/utilizations`

**Chart 6: CPU Usage**
- Resource type: Cloud Run Revision
- Metric: `run.googleapis.com/container/cpu/utilizations`

---

## Alert Policies

### 1. High Error Rate Alert

```yaml
# alert-high-error-rate.yaml
displayName: "High Error Rate (5xx)"
conditions:
  - displayName: "5xx errors > 5% of requests"
    conditionThreshold:
      filter: |
        resource.type = "cloud_run_revision"
        AND metric.type = "run.googleapis.com/request_count"
        AND metric.label.response_code_class = "5xx"
      aggregations:
        - alignmentPeriod: 60s
          perSeriesAligner: ALIGN_RATE
          crossSeriesReducer: REDUCE_SUM
      comparison: COMPARISON_GT
      thresholdValue: 5  # errors per minute
      duration: 300s  # sustained for 5 minutes

notificationChannels:
  - projects/[PROJECT_ID]/notificationChannels/[CHANNEL_ID]

alertStrategy:
  autoClose: 1800s  # 30 minutes
```

Apply via CLI:
```bash
gcloud alpha monitoring policies create --policy-from-file=alert-high-error-rate.yaml
```

### 2. High Latency Alert

```yaml
displayName: "High Request Latency (p95)"
conditions:
  - displayName: "p95 latency > 2 seconds"
    conditionThreshold:
      filter: |
        resource.type = "cloud_run_revision"
        AND metric.type = "run.googleapis.com/request_latencies"
      aggregations:
        - alignmentPeriod: 60s
          perSeriesAligner: ALIGN_DELTA
          crossSeriesReducer: REDUCE_PERCENTILE_95
      comparison: COMPARISON_GT
      thresholdValue: 2000  # milliseconds
      duration: 300s
```

### 3. Database Connection Alert

```yaml
displayName: "High Database Connection Usage"
conditions:
  - displayName: "DB connections > 80% of limit"
    conditionThreshold:
      filter: |
        resource.type = "cloudsql_database"
        AND metric.type = "cloudsql.googleapis.com/database/network/connections"
      comparison: COMPARISON_GT
      thresholdValue: 80  # depends on your tier
      duration: 600s
```

### 4. Memory Pressure Alert

```yaml
displayName: "High Memory Usage"
conditions:
  - displayName: "Memory utilization > 90%"
    conditionThreshold:
      filter: |
        resource.type = "cloud_run_revision"
        AND metric.type = "run.googleapis.com/container/memory/utilizations"
      comparison: COMPARISON_GT
      thresholdValue: 0.9  # 90%
      duration: 300s
```

### Create Alerts via Console

1. Go to **Monitoring** > **Alerting** > **Create Policy**
2. Add condition (metric threshold)
3. Configure notification channels (Email, Slack, PagerDuty)
4. Set documentation (runbook link)
5. Save policy

---

## Uptime Monitoring

### 1. Create Uptime Checks

**Backend Health Check:**

```yaml
# uptime-backend-health.yaml
displayName: "Backend Health Check"
monitoredResource:
  type: "uptime_url"
httpCheck:
  path: "/health"
  port: 443
  useSsl: true
  validateSsl: true
timeout: 10s
period: 60s  # Check every minute
selectedRegions:
  - USA
  - EUROPE
  - ASIA_PACIFIC
```

Apply:
```bash
gcloud monitoring uptime-checks create https://api.businessos.example.com/health \
  --display-name="Backend Health Check"
```

**Backend Ready Check:**

```bash
gcloud monitoring uptime-checks create https://api.businessos.example.com/ready \
  --display-name="Backend Ready Check"
```

**Frontend Homepage:**

```bash
gcloud monitoring uptime-checks create https://app.businessos.example.com \
  --display-name="Frontend Homepage Check"
```

### 2. Uptime Check Alerts

Create alert when uptime check fails:

```yaml
displayName: "Service Down - Health Check Failed"
conditions:
  - displayName: "Health check fails"
    conditionThreshold:
      filter: |
        resource.type = "uptime_url"
        AND metric.type = "monitoring.googleapis.com/uptime_check/check_passed"
      aggregations:
        - alignmentPeriod: 60s
          perSeriesAligner: ALIGN_FRACTION_TRUE
      comparison: COMPARISON_LT
      thresholdValue: 0.8  # Alert if < 80% success rate
      duration: 60s

notificationChannels:
  - [CRITICAL_CHANNEL_ID]  # Page on-call engineer
```

---

## Performance Metrics

### Custom Metrics Export

Add custom metrics from your application:

```go
package metrics

import (
	"context"
	"log/slog"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MetricsClient struct {
	client    *monitoring.MetricClient
	projectID string
}

func NewMetricsClient(ctx context.Context, projectID string) (*MetricsClient, error) {
	client, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return nil, err
	}

	return &MetricsClient{
		client:    client,
		projectID: projectID,
	}, nil
}

// RecordCacheHitRate records Redis cache hit rate
func (m *MetricsClient) RecordCacheHitRate(ctx context.Context, hitRate float64) error {
	req := &monitoringpb.CreateTimeSeriesRequest{
		Name: "projects/" + m.projectID,
		TimeSeries: []*monitoringpb.TimeSeries{{
			Metric: &monitoringpb.Metric{
				Type: "custom.googleapis.com/redis/hit_rate",
			},
			Resource: &monitoringpb.MonitoredResource{
				Type: "global",
			},
			Points: []*monitoringpb.Point{{
				Interval: &monitoringpb.TimeInterval{
					EndTime: timestamppb.Now(),
				},
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_DoubleValue{
						DoubleValue: hitRate,
					},
				},
			}},
		}},
	}

	err := m.client.CreateTimeSeries(ctx, req)
	if err != nil {
		slog.Error("Failed to record metric", "error", err)
	}
	return err
}
```

### Tracked Metrics

| Metric | Type | Description | Alert Threshold |
|--------|------|-------------|-----------------|
| HTTP request rate | Counter | Requests per second | N/A |
| HTTP error rate | Counter | 5xx errors per minute | > 5 |
| Request latency | Histogram | p50/p95/p99 latency | p95 > 2s |
| Database connections | Gauge | Active DB connections | > 80% limit |
| Redis hit rate | Gauge | Cache hit percentage | < 90% |
| Redis memory usage | Gauge | Memory utilization | > 80% |
| Active sessions | Gauge | Concurrent user sessions | N/A |
| Background jobs queued | Gauge | Pending jobs | > 100 |
| Background jobs failed | Counter | Failed job executions | > 5/min |

---

## Log-Based Metrics

Create metrics from log entries:

### 1. Failed Login Attempts

```bash
gcloud logging metrics create failed_logins \
  --description="Failed login attempts" \
  --log-filter='jsonPayload.message:"Authentication failed"'
```

### 2. Slow Database Queries

```bash
gcloud logging metrics create slow_queries \
  --description="Database queries > 1 second" \
  --log-filter='jsonPayload.message:"Slow database query"'
```

### 3. Background Job Failures

```bash
gcloud logging metrics create job_failures \
  --description="Background job failures" \
  --log-filter='jsonPayload.message:"Background job failed"'
```

---

## Notification Channels

### Setup Email Notifications

```bash
gcloud alpha monitoring channels create \
  --display-name="Ops Team Email" \
  --type=email \
  --channel-labels=email_address=ops@businessos.example.com
```

### Setup Slack Notifications

1. Create Slack webhook: https://api.slack.com/messaging/webhooks
2. Add notification channel:

```bash
gcloud alpha monitoring channels create \
  --display-name="Slack #alerts" \
  --type=slack \
  --channel-labels=url=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

---

## Monitoring Checklist

### Initial Setup

- [ ] Configure structured logging (slog + JSON)
- [ ] Create Cloud Monitoring dashboard
- [ ] Set up uptime checks (/health, /ready, homepage)
- [ ] Create alert policies (error rate, latency, DB)
- [ ] Configure notification channels (email, Slack)
- [ ] Enable Cloud Logging API
- [ ] Set log retention policies (30-90 days)

### Post-Deployment

- [ ] Verify logs appear in Cloud Logging
- [ ] Check dashboard shows live data
- [ ] Test uptime checks are running
- [ ] Trigger test alert to verify notifications
- [ ] Document alert response procedures
- [ ] Set up log-based metrics
- [ ] Configure log exclusions (reduce costs)

---

## Resources

- [Cloud Monitoring Documentation](https://cloud.google.com/monitoring/docs)
- [Cloud Logging Documentation](https://cloud.google.com/logging/docs)
- [Alert Policy Best Practices](https://cloud.google.com/monitoring/alerts/best-practices)

---

**Last Updated:** 2026-01-18
**Maintainer:** BusinessOS Team
