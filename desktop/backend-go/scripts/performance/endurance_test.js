/**
 * k6 Endurance Test - OSA System
 *
 * Long-running stability test:
 * - 500 req/s sustained for 2 hours
 * - Detect memory leaks
 * - Monitor circuit breaker patterns
 * - Track performance degradation over time
 *
 * Run this overnight or during non-peak hours
 */

/**
 * Required environment variables:
 *   BASE_URL            - Target server URL (default: http://localhost:8001)
 *   AUTH_TOKEN          - Bearer token for authenticated requests (preferred)
 *   TEST_LOGIN_EMAIL    - Login email used when AUTH_TOKEN is not set
 *   TEST_LOGIN_PASSWORD - Login password used when AUTH_TOKEN is not set
 *   WORKSPACE_ID        - Workspace ID for generation payloads
 *   TEST_APP_ID         - App ID for status polling (default: zero UUID)
 *   DURATION_HOURS      - Test duration in hours (default: 2)
 *
 * Example:
 *   k6 run -e AUTH_TOKEN=<token> -e BASE_URL=http://localhost:8001 -e DURATION_HOURS=2 endurance_test.js
 *   k6 run -e TEST_LOGIN_EMAIL=user@example.com -e TEST_LOGIN_PASSWORD=secret endurance_test.js
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';

// Custom metrics
const memoryLeakIndicator = new Gauge('memory_leak_indicator');
const performanceDegradation = new Trend('performance_degradation_percent');
const circuitBreakerEvents = new Counter('circuit_breaker_events');
const errorRate = new Rate('error_rate');
const responseTimePerHour = new Trend('response_time_per_hour', true);

// Baseline tracking
let baselineP95 = null;
let hourlyMetrics = [];

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8001';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';
const DURATION_HOURS = __ENV.DURATION_HOURS || 2;

export const options = {
  scenarios: {
    endurance: {
      executor: 'constant-arrival-rate',
      rate: 500,           // 500 req/s
      timeUnit: '1s',
      duration: `${DURATION_HOURS}h`,  // Default: 2 hours
      preAllocatedVUs: 250,
      maxVUs: 500,
    },
  },

  thresholds: {
    // Performance should remain stable
    'http_req_duration': [
      'p(50)<200',
      'p(95)<500',
      'p(99)<1000',
    ],

    // Error rate should stay low
    'error_rate': ['rate<0.01'],  // < 1%

    // Performance degradation threshold
    'performance_degradation_percent': ['p(95)<20'],  // < 20% degradation

    // Minimal circuit breaker events
    'circuit_breaker_events': ['count<10'],  // < 10 events in 2 hours
  },

  // Summary settings
  summaryTrendStats: ['min', 'avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max', 'count'],
};

// Setup authentication
export function setup() {
  console.log('========================================================');
  console.log(`  🕐 Starting ${DURATION_HOURS}-hour Endurance Test`);
  console.log('========================================================');
  console.log('');
  console.log('⏰ Duration:', `${DURATION_HOURS} hours`);
  console.log('📊 Load:', '500 req/s sustained');
  console.log('');
  console.log('🔍 Monitoring for:');
  console.log('  - Memory leaks');
  console.log('  - Performance degradation');
  console.log('  - Circuit breaker patterns');
  console.log('  - Connection pool exhaustion');
  console.log('');

  if (!AUTH_TOKEN) {
    const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
      email: __ENV.TEST_LOGIN_EMAIL,
      password: __ENV.TEST_LOGIN_PASSWORD,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });

    if (loginRes.status === 200) {
      return { token: loginRes.json('token'), startTime: Date.now() };
    }
  }

  return { token: AUTH_TOKEN, startTime: Date.now() };
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

  const now = Date.now();
  const elapsedHours = (now - data.startTime) / (1000 * 60 * 60);
  const currentHour = Math.floor(elapsedHours);

  // Distribute load across different endpoints
  const rand = Math.random();

  if (rand < 0.3) {
    testHealthEndpoint(headers, currentHour);
  } else if (rand < 0.6) {
    testWorkspacesEndpoint(headers, currentHour);
  } else if (rand < 0.85) {
    testStatusEndpoint(headers, currentHour);
  } else {
    testGenerateEndpoint(headers, currentHour);
  }

  sleep(0.1); // 100ms think time
}

// Test health endpoint
function testHealthEndpoint(headers, hour) {
  group('Health Check', () => {
    const response = http.get(`${BASE_URL}/api/osa/health`, { headers });

    const success = check(response, {
      'health: status 200': (r) => r.status === 200,
      'health: has status': (r) => r.json('status') !== undefined,
    });

    trackMetrics(response, hour, success);
  });
}

// Test workspaces endpoint
function testWorkspacesEndpoint(headers, hour) {
  group('List Workspaces', () => {
    const response = http.get(`${BASE_URL}/api/osa/workspaces`, { headers });

    const success = check(response, {
      'workspaces: status 200': (r) => r.status === 200,
      'workspaces: has data': (r) => r.json('workspaces') !== undefined,
    });

    trackMetrics(response, hour, success);
  });
}

// Test status endpoint
function testStatusEndpoint(headers, hour) {
  group('App Status', () => {
    const appId = __ENV.TEST_APP_ID || '00000000-0000-0000-0000-000000000001';
    const response = http.get(`${BASE_URL}/api/osa/status/${appId}`, { headers });

    const success = check(response, {
      'status: response OK': (r) => r.status === 200 || r.status === 404,
    });

    trackMetrics(response, hour, success);
  });
}

// Test generate endpoint (lighter load)
function testGenerateEndpoint(headers, hour) {
  group('Generate App', () => {
    const payload = JSON.stringify({
      name: `EnduranceApp-${__VU}-${Date.now()}`,
      description: 'Endurance test app generation',
      type: 'module',
      workspace_id: __ENV.WORKSPACE_ID || '',
    });

    const response = http.post(`${BASE_URL}/api/osa/generate`, payload, { headers });

    const success = check(response, {
      'generate: status 200': (r) => r.status === 200,
    });

    trackMetrics(response, hour, success);
  });
}

// Track metrics and detect anomalies
function trackMetrics(response, hour, success) {
  const duration = response.timings.duration;

  // Record baseline in first 5 minutes
  if (!baselineP95 && __ITER > 100) {
    baselineP95 = duration;
    console.log(`📊 Baseline P95 established: ${baselineP95.toFixed(2)}ms`);
  }

  // Record hourly metrics
  if (!hourlyMetrics[hour]) {
    hourlyMetrics[hour] = {
      count: 0,
      totalDuration: 0,
      errors: 0,
    };
  }

  hourlyMetrics[hour].count++;
  hourlyMetrics[hour].totalDuration += duration;
  if (!success) {
    hourlyMetrics[hour].errors++;
  }

  // Calculate performance degradation
  if (baselineP95 && duration > baselineP95) {
    const degradation = ((duration - baselineP95) / baselineP95) * 100;
    performanceDegradation.add(degradation);

    // Memory leak indicator (if degradation is consistent)
    if (degradation > 50) {
      memoryLeakIndicator.add(1);
    }
  }

  // Track error rate
  errorRate.add(!success);

  // Detect circuit breaker
  if (response.status === 503) {
    circuitBreakerEvents.add(1);
    console.log(`⚡ Circuit breaker at ${new Date().toISOString()}`);
  }

  // Record response time per hour
  responseTimePerHour.add(duration, { hour: hour.toString() });

  // Periodic status update (every 15 minutes)
  if (__ITER % 45000 === 0) {
    console.log(`⏰ ${Math.floor((Date.now() - data.startTime) / 60000)} minutes elapsed`);
    console.log(`   Avg response time: ${duration.toFixed(2)}ms`);
    console.log(`   Error rate: ${(errorRate.rate * 100).toFixed(3)}%`);
  }
}

// Teardown: Generate detailed analysis
export function teardown(data) {
  const totalHours = (Date.now() - data.startTime) / (1000 * 60 * 60);

  console.log('========================================================');
  console.log('  🏁 Endurance Test Complete');
  console.log('========================================================');
  console.log('');
  console.log(`⏰ Total Duration: ${totalHours.toFixed(2)} hours`);
  console.log('');

  console.log('📊 Hourly Breakdown:');
  hourlyMetrics.forEach((hourData, hour) => {
    if (hourData) {
      const avgDuration = hourData.totalDuration / hourData.count;
      const errorRate = (hourData.errors / hourData.count) * 100;
      console.log(`  Hour ${hour}:`);
      console.log(`    - Avg Response Time: ${avgDuration.toFixed(2)}ms`);
      console.log(`    - Error Rate: ${errorRate.toFixed(2)}%`);
      console.log(`    - Total Requests: ${hourData.count}`);
    }
  });

  console.log('');
  console.log('🔍 Leak Detection:');
  console.log(`  - Baseline P95: ${baselineP95 ? baselineP95.toFixed(2) : 'N/A'}ms`);
  console.log(`  - Memory leak indicators: ${memoryLeakIndicator.value || 0}`);
  console.log(`  - Circuit breaker events: ${circuitBreakerEvents.value || 0}`);
  console.log('');

  console.log('✅ Stability Checklist:');
  console.log('  □ Response times remained stable');
  console.log('  □ Error rate stayed below 1%');
  console.log('  □ No memory leaks detected');
  console.log('  □ Circuit breaker behaved correctly');
  console.log('  □ Database connections stable');
  console.log('  □ No resource exhaustion');
  console.log('');

  console.log('💡 Analysis Tips:');
  console.log('  1. Compare first hour vs last hour metrics');
  console.log('  2. Check backend memory usage (Prometheus/Grafana)');
  console.log('  3. Review database connection pool stats');
  console.log('  4. Analyze GC patterns (Go runtime metrics)');
  console.log('');

  console.log('📈 Export results:');
  console.log('   k6 run --out json=endurance-results.json endurance_test.js');
  console.log('');
}
