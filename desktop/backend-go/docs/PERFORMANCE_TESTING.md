# Performance Testing Guide

**Version:** 1.0.0
**Last Updated:** 2026-01-26
**Status:** Production-Ready

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Test Suites](#test-suites)
4. [Running Tests](#running-tests)
5. [Performance Targets](#performance-targets)
6. [Interpreting Results](#interpreting-results)
7. [Baseline Management](#baseline-management)
8. [CI/CD Integration](#cicd-integration)
9. [Troubleshooting](#troubleshooting)

---

## Overview

BusinessOS uses **k6** by Grafana for comprehensive performance testing. Our test suite covers:

- **Load Testing:** OSA endpoints under varying load (100-1000 req/s)
- **Hybrid Architecture:** 70% direct path, 30% CoT path
- **Spike Testing:** Sudden traffic spikes (5000 req/s)
- **Endurance Testing:** 2-hour sustained load (500 req/s)

### Why k6?

- **Developer-friendly:** Write tests in JavaScript
- **Scalable:** Can simulate millions of VUs
- **Cloud-native:** Integrates with Prometheus, Grafana
- **CI/CD ready:** Easy GitHub Actions integration

---

## Installation

### Quick Install

```bash
cd desktop/backend-go/scripts/performance
./install_k6.sh
```

### Platform-Specific

#### Windows
```powershell
# Via Chocolatey
choco install k6

# Via winget
winget install k6
```

#### macOS
```bash
brew install k6
```

#### Linux (Debian/Ubuntu)
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 \
  --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### Verify Installation

```bash
k6 version
# Expected output: k6 v0.48.0 (or later)
```

---

## Test Suites

### 1. OSA Load Test

**File:** `load_test_osa.js`

Tests OSA endpoints under realistic load:
- `/api/osa/generate` (40% of traffic)
- `/api/osa/status/:id` (40% of traffic)
- `/api/osa/orchestrate` (20% of traffic)

**Scenarios:**
- **Low Load:** 100 req/s for 5 minutes
- **Medium Load:** 500 req/s for 5 minutes
- **High Load:** 1000 req/s for 5 minutes

**Key Metrics:**
- P50, P95, P99 latency per endpoint
- Error rate per scenario
- Success rate per endpoint type

### 2. Hybrid Architecture Test

**File:** `load_test_hybrid.js`

Tests the hybrid routing system:
- **70% Simple Requests:** Direct path (fast route)
- **30% Complex Requests:** CoT path (orchestration)

**Key Metrics:**
- Direct path latency (target: P95 < 200ms)
- CoT path latency (target: P95 < 3s)
- Routing overhead (target: P95 < 50ms)
- Success rates per path

### 3. Spike Test

**File:** `spike_test.js`

Simulates sudden traffic spikes:
1. **Baseline:** 100 req/s for 30s
2. **Spike Up:** Ramp to 5000 req/s over 30s
3. **Hold Spike:** 5000 req/s for 1 minute
4. **Recovery:** Drop to 100 req/s over 30s
5. **Post-Recovery:** 100 req/s for 1 minute

**Key Metrics:**
- Circuit breaker trips
- Error rate during spike
- System recovery time
- Resource exhaustion indicators

### 4. Endurance Test

**File:** `endurance_test.js`

Long-running stability test:
- **Duration:** 2 hours (configurable)
- **Load:** 500 req/s sustained
- **Monitoring:** Memory leaks, degradation

**Key Metrics:**
- Performance degradation over time
- Memory leak indicators
- Hourly performance breakdown
- Circuit breaker patterns

---

## Running Tests

### Prerequisites

1. **Start Backend:**
   ```bash
   cd desktop/backend-go
   go run cmd/server/main.go
   ```

2. **Set Environment Variables:**
   ```bash
   export BASE_URL=http://localhost:8001
   export AUTH_TOKEN=your_jwt_token_here
   export WORKSPACE_ID=your_workspace_id_here
   export TEST_APP_ID=test_app_id_here  # Optional
   ```

### Run Individual Tests

#### OSA Load Test
```bash
cd desktop/backend-go/scripts/performance
k6 run load_test_osa.js
```

#### Hybrid Architecture Test
```bash
k6 run load_test_hybrid.js
```

#### Spike Test
```bash
k6 run spike_test.js
```

#### Endurance Test
```bash
# Default: 2 hours
k6 run endurance_test.js

# Custom duration
k6 run -e DURATION_HOURS=4 endurance_test.js
```

### Run with JSON Output

```bash
k6 run --out json=results.json load_test_osa.js
```

### Run with HTML Report

```bash
k6 run --out json=results.json load_test_osa.js
k6 report results.json --out html-report.html
```

---

## Performance Targets

### Direct Path (Simple Requests)

| Metric | Target | Threshold |
|--------|--------|-----------|
| **P50 Latency** | < 50ms | < 100ms |
| **P95 Latency** | < 100ms | < 200ms |
| **P99 Latency** | < 200ms | < 500ms |
| **Success Rate** | > 99.9% | > 99.5% |

### CoT Path (Complex Requests)

| Metric | Target | Threshold |
|--------|--------|-----------|
| **P50 Latency** | < 500ms | < 1s |
| **P95 Latency** | < 2s | < 3s |
| **P99 Latency** | < 3s | < 5s |
| **Success Rate** | > 99.5% | > 99% |

### System-Wide

| Metric | Target | Threshold |
|--------|--------|-----------|
| **Overall Error Rate** | < 0.1% | < 1% |
| **Circuit Breaker Activation** | Rare | < 5 failures trigger |
| **Recovery Time** | < 3s | < 5s |
| **Routing Overhead** | < 20ms | < 50ms |

### Spike Test

| Metric | Target | Threshold |
|--------|--------|-----------|
| **Error Rate During Spike** | < 2% | < 5% |
| **Circuit Breaker Trips** | > 0 | Must activate |
| **Recovery Time** | < 3s | < 5s |
| **Post-Recovery P95** | < 200ms | < 500ms |

### Endurance Test

| Metric | Target | Threshold |
|--------|--------|-----------|
| **Performance Degradation** | < 10% | < 20% |
| **Memory Leak Indicators** | 0 | < 5 |
| **Hour-over-Hour Variance** | < 15% | < 25% |
| **Circuit Breaker Events** | < 5 in 2h | < 10 in 2h |

---

## Interpreting Results

### k6 Output

```
     ✓ http_req_duration...: avg=125.5ms  min=45ms  med=95ms  max=2.1s  p(90)=180ms p(95)=250ms
     ✓ http_req_failed.....: 0.08% ✓ 48 ✗ 59952
     ✓ http_reqs...........: 60000 (200 per second)
```

### Key Sections

1. **Checks:** Pass/fail ratio for assertions
2. **HTTP Metrics:** Duration, failure rate, request count
3. **Custom Metrics:** Domain-specific measurements
4. **Thresholds:** Whether targets were met

### What to Look For

#### Good Performance
- ✅ All thresholds passing
- ✅ P95 < target
- ✅ Error rate < 1%
- ✅ Consistent latency across scenarios

#### Performance Issues
- ❌ Thresholds failing
- ❌ High P99 (> 2x P95)
- ❌ Increasing latency over time
- ❌ Error rate spikes

#### Critical Issues
- 🔴 Error rate > 5%
- 🔴 Circuit breaker never activates
- 🔴 System crashes/OOM
- 🔴 Recovery time > 10s

---

## Baseline Management

### Creating Baselines

After initial performance optimization:

```bash
# Run all tests
k6 run --out json=baseline-osa.json load_test_osa.js
k6 run --out json=baseline-hybrid.json load_test_hybrid.js
k6 run --out json=baseline-spike.json spike_test.js

# Store baselines
mkdir -p baseline
mv baseline-*.json baseline/
git add baseline/
git commit -m "chore: add performance baselines"
```

### Comparing Against Baselines

```bash
# Run current test
k6 run --out json=current-osa.json load_test_osa.js

# Compare with analyze_results.sh
./analyze_results.sh baseline/baseline-osa.json current-osa.json
```

The analysis script will:
1. Compare P95 latencies
2. Calculate degradation percentage
3. Exit with code 1 if degradation > 20%

---

## CI/CD Integration

### GitHub Actions

Performance tests run weekly via `.github/workflows/performance-tests.yml`:

```yaml
# Automated weekly performance regression tests
on:
  schedule:
    - cron: '0 2 * * 0'  # Sundays at 2 AM
  workflow_dispatch:      # Manual trigger
```

### Manual Trigger

```bash
# Via GitHub CLI
gh workflow run performance-tests.yml

# Via GitHub UI
Actions → Performance Tests → Run workflow
```

### Alerts

Performance regression alerts trigger when:
- P95 latency degrades by > 20%
- Error rate increases by > 50%
- Any critical threshold fails

---

## Troubleshooting

### Common Issues

#### 1. Connection Refused

**Symptom:** `dial: connection refused`

**Solution:**
```bash
# Ensure backend is running
cd desktop/backend-go
go run cmd/server/main.go

# Check port
netstat -an | grep 8001
```

#### 2. Authentication Failures

**Symptom:** `401 Unauthorized`

**Solution:**
```bash
# Get fresh auth token
curl -X POST http://localhost:8001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpassword123"}'

# Set in environment
export AUTH_TOKEN="eyJhbGciOiJIUzI1NiIs..."
```

#### 3. High Error Rates

**Symptom:** Error rate > 10%

**Possible Causes:**
- Database connection pool exhausted
- Rate limiting active
- Circuit breaker triggering too early
- Insufficient backend resources

**Investigation:**
```bash
# Check backend logs
tail -f logs/server.log | grep ERROR

# Monitor database connections
psql -c "SELECT count(*) FROM pg_stat_activity;"

# Check system resources
htop  # CPU/Memory
```

#### 4. Inconsistent Results

**Symptom:** Results vary significantly between runs

**Possible Causes:**
- Background processes
- Shared test environment
- Cache warming effects
- Network instability

**Solution:**
- Run tests in isolated environment
- Warm up system first (discard first minute)
- Use dedicated test database
- Increase test duration for stability

---

## Best Practices

### 1. Test Environment

- **Isolated:** Dedicated test environment
- **Realistic:** Match production specs
- **Clean State:** Reset database between runs
- **Monitoring:** Enable Prometheus/Grafana

### 2. Test Execution

- **Warm-up:** Discard first 30-60s of data
- **Consistent Load:** Avoid other background tasks
- **Peak Hours:** Test during expected peak times
- **Regular Cadence:** Weekly automated runs

### 3. Result Analysis

- **Trends:** Track metrics over time
- **Baselines:** Maintain up-to-date baselines
- **Investigations:** Deep-dive on regressions
- **Documentation:** Record findings

### 4. Performance Budget

Set and enforce performance budgets:

```javascript
// In k6 test
thresholds: {
  'http_req_duration': ['p(95)<200'],  // Budget: 200ms P95
}
```

If tests fail, investigate before merging.

---

## Advanced Topics

### Custom Metrics

Add domain-specific metrics:

```javascript
import { Trend } from 'k6/metrics';

const dbQueryTime = new Trend('db_query_time');

// In test
dbQueryTime.add(queryDuration);
```

### Distributed Testing

Run tests across multiple machines:

```bash
# Master node
k6 run --execution-segment "0:1/4" load_test_osa.js

# Worker nodes
k6 run --execution-segment "1/4:2/4" load_test_osa.js
k6 run --execution-segment "2/4:3/4" load_test_osa.js
k6 run --execution-segment "3/4:4/4" load_test_osa.js
```

### Cloud Execution

Use k6 Cloud for large-scale tests:

```bash
k6 cloud load_test_osa.js
```

---

## Support

### Resources

- **k6 Documentation:** https://k6.io/docs/
- **k6 Community:** https://community.k6.io/
- **BusinessOS Slack:** #performance-testing

### Reporting Issues

```bash
# Create performance issue
gh issue create \
  --title "Performance regression in OSA endpoints" \
  --body "P95 latency increased by 40% in OSA load test" \
  --label "performance,critical"
```

---

**Last Updated:** 2026-01-26
**Maintained by:** DevOps Team
**Review Schedule:** Quarterly
