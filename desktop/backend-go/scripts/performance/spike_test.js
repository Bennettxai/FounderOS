/**
 * k6 Spike Test - OSA System
 *
 * Sudden traffic spike to test system resilience:
 * - Ramp from 100 → 5000 req/s over 30 seconds
 * - Hold at 5000 req/s for 1 minute
 * - Ramp down to 100 req/s
 *
 * Measures:
 * - Circuit breaker behavior
 * - System recovery time
 * - Error rate during spike
 * - Resource exhaustion handling
 */

/**
 * Required environment variables:
 *   BASE_URL            - Target server URL (default: http://localhost:8001)
 *   AUTH_TOKEN          - Bearer token for authenticated requests (preferred)
 *   TEST_LOGIN_EMAIL    - Login email used when AUTH_TOKEN is not set
 *   TEST_LOGIN_PASSWORD - Login password used when AUTH_TOKEN is not set
 *
 * Example:
 *   k6 run -e AUTH_TOKEN=<token> -e BASE_URL=http://localhost:8001 spike_test.js
 *   k6 run -e TEST_LOGIN_EMAIL=user@example.com -e TEST_LOGIN_PASSWORD=secret spike_test.js
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';

// Custom metrics
const spikeErrors = new Counter('spike_errors');
const circuitBreakerTrips = new Counter('circuit_breaker_trips');
const recoveryTime = new Trend('recovery_time_ms');
const activeRequests = new Gauge('active_requests');
const errorRate = new Rate('error_rate');
const responseTime = new Trend('response_time');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8001';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

export const options = {
  scenarios: {
    spike: {
      executor: 'ramping-arrival-rate',
      startRate: 100,
      timeUnit: '1s',
      preAllocatedVUs: 500,
      maxVUs: 10000,

      stages: [
        // Baseline
        { duration: '30s', target: 100 },   // 100 req/s baseline

        // SPIKE UP (30 seconds)
        { duration: '30s', target: 5000 },  // Rapid spike to 5000 req/s

        // HOLD SPIKE (1 minute)
        { duration: '1m', target: 5000 },   // Sustain high load

        // RECOVERY (30 seconds)
        { duration: '30s', target: 100 },   // Drop back to baseline

        // POST-RECOVERY (1 minute)
        { duration: '1m', target: 100 },    // Verify system recovered
      ],
    },
  },

  thresholds: {
    // System should stay responsive
    'http_req_duration': [
      'p(50)<500',   // P50 < 500ms (may degrade during spike)
      'p(95)<3000',  // P95 < 3s (acceptable during spike)
    ],

    // Error rate threshold
    'error_rate': ['rate<0.05'],  // < 5% errors acceptable during spike

    // Circuit breaker should trigger
    'circuit_breaker_trips': ['count>0'],  // Should trip at least once

    // Recovery should be fast
    'recovery_time_ms': ['p(95)<5000'],  // < 5s to recover

    // Requests should complete
    'http_req_failed': ['rate<0.1'],  // < 10% failure rate
  },
};

// State tracking for recovery measurement
let spikeStartTime = null;
let spikeEndTime = null;
let isInSpike = false;
let recoveryStartTime = null;

// Setup authentication
export function setup() {
  if (!AUTH_TOKEN) {
    const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
      email: __ENV.TEST_LOGIN_EMAIL,
      password: __ENV.TEST_LOGIN_PASSWORD,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });

    if (loginRes.status === 200) {
      console.log('✅ Authentication successful');
      return { token: loginRes.json('token') };
    }
  }

  return { token: AUTH_TOKEN };
}

// Main test function
export default function(data) {
  const token = data.token;

  if (!token) {
    console.error('❌ No auth token available');
    return;
  }

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };

  // Track spike phases
  const now = Date.now();
  const executionTime = (now - __ITER) / 1000;

  // Detect spike phase
  if (executionTime >= 60 && executionTime < 150) {
    if (!isInSpike) {
      isInSpike = true;
      spikeStartTime = now;
      console.log('🔥 SPIKE STARTED');
    }
  } else if (isInSpike && executionTime >= 150) {
    isInSpike = false;
    spikeEndTime = now;
    recoveryStartTime = now;
    console.log('📉 SPIKE ENDED - Recovery phase started');
  }

  // Increment active requests
  activeRequests.add(1);

  // Execute test
  group('Spike Test - OSA Health Check', () => {
    const startTime = Date.now();
    const response = http.get(`${BASE_URL}/api/osa/health`, { headers });
    const duration = Date.now() - startTime;

    responseTime.add(duration);

    const success = check(response, {
      'status is 200 or 503': (r) => r.status === 200 || r.status === 503,
      'response received': (r) => r.body.length > 0,
    });

    errorRate.add(!success);

    if (!success) {
      spikeErrors.add(1);

      // Detect circuit breaker trip
      if (response.status === 503 || response.body.includes('circuit breaker')) {
        circuitBreakerTrips.add(1);
        console.log('⚡ Circuit breaker tripped');
      }
    }

    // Measure recovery time
    if (recoveryStartTime && response.status === 200 && duration < 1000) {
      const timeSinceRecoveryStart = now - recoveryStartTime;
      if (timeSinceRecoveryStart < 30000) { // Within 30s of recovery
        recoveryTime.add(timeSinceRecoveryStart);
      }
    }
  });

  // Decrement active requests
  activeRequests.add(-1);

  // Minimal think time during spike
  sleep(0.01); // 10ms
}

// Teardown: Generate spike analysis
export function teardown(data) {
  console.log('========================================================');
  console.log('  🏁 Spike Test Complete');
  console.log('========================================================');
  console.log('');
  console.log('📊 Spike Metrics:');
  console.log(`  - Total Errors: ${spikeErrors.value || 0}`);
  console.log(`  - Circuit Breaker Trips: ${circuitBreakerTrips.value || 0}`);
  console.log(`  - Overall Error Rate: ${(errorRate.rate * 100).toFixed(2)}%`);
  console.log('');

  if (spikeStartTime && spikeEndTime) {
    const spikeDuration = (spikeEndTime - spikeStartTime) / 1000;
    console.log(`⚡ Spike Duration: ${spikeDuration.toFixed(1)}s`);
  }

  console.log('');
  console.log('✅ System Resilience Checklist:');
  console.log('  □ Circuit breaker triggered during spike');
  console.log('  □ Error rate stayed below 10%');
  console.log('  □ System recovered within 5 seconds');
  console.log('  □ No resource exhaustion (OOM, CPU throttling)');
  console.log('  □ Database connections managed properly');
  console.log('');
  console.log('💡 Next Steps:');
  console.log('  1. Review k6 HTML report for detailed timings');
  console.log('  2. Check backend logs for circuit breaker events');
  console.log('  3. Analyze Prometheus metrics (if available)');
  console.log('  4. Verify no memory leaks after recovery');
  console.log('');
  console.log('📈 Export results:');
  console.log('   k6 run --out json=spike-results.json spike_test.js');
  console.log('');
}
