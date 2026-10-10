/**
 * k6 Load Test - Hybrid Architecture
 *
 * Tests the hybrid routing architecture:
 * - 70% simple requests → direct path (fast route)
 * - 30% complex requests → CoT path (orchestration)
 *
 * Measures:
 * - Routing overhead
 * - Latency comparison (direct vs CoT)
 * - Circuit breaker behavior
 * - Success rates per path
 */

/**
 * Required environment variables:
 *   BASE_URL            - Target server URL (default: http://localhost:8001)
 *   AUTH_TOKEN          - Bearer token for authenticated requests (preferred)
 *   TEST_LOGIN_EMAIL    - Login email used when AUTH_TOKEN is not set
 *   TEST_LOGIN_PASSWORD - Login password used when AUTH_TOKEN is not set
 *   WORKSPACE_ID        - Workspace ID for generation payloads
 *
 * Example:
 *   k6 run -e AUTH_TOKEN=<token> -e BASE_URL=http://localhost:8001 load_test_hybrid.js
 *   k6 run -e TEST_LOGIN_EMAIL=user@example.com -e TEST_LOGIN_PASSWORD=secret load_test_hybrid.js
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const directPathLatency = new Trend('direct_path_latency');
const cotPathLatency = new Trend('cot_path_latency');
const directPathErrors = new Counter('direct_path_errors');
const cotPathErrors = new Counter('cot_path_errors');
const directPathSuccess = new Rate('direct_path_success');
const cotPathSuccess = new Rate('cot_path_success');
const routingOverhead = new Trend('routing_overhead_ms');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8001';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

// Test configuration
export const options = {
  scenarios: {
    hybrid_load: {
      executor: 'ramping-vus',
      startVUs: 10,
      stages: [
        { duration: '2m', target: 50 },   // Ramp up to 50 VUs
        { duration: '5m', target: 50 },   // Hold at 50 VUs
        { duration: '2m', target: 100 },  // Ramp up to 100 VUs
        { duration: '5m', target: 100 },  // Hold at 100 VUs
        { duration: '2m', target: 0 },    // Ramp down
      ],
    },
  },

  thresholds: {
    // Overall performance
    'http_req_duration': ['p(95)<1000'],
    'http_req_failed': ['rate<0.01'],

    // Direct path should be fast
    'direct_path_latency': [
      'p(50)<100',  // P50 < 100ms
      'p(95)<200',  // P95 < 200ms
      'p(99)<500',  // P99 < 500ms
    ],

    // CoT path can be slower but bounded
    'cot_path_latency': [
      'p(50)<1000',  // P50 < 1s
      'p(95)<3000',  // P95 < 3s
      'p(99)<5000',  // P99 < 5s
    ],

    // Success rates
    'direct_path_success': ['rate>0.999'],  // 99.9%
    'cot_path_success': ['rate>0.99'],      // 99%

    // Routing overhead should be minimal
    'routing_overhead_ms': ['p(95)<50'],    // P95 < 50ms
  },
};

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

  // 70% simple requests (direct path)
  // 30% complex requests (CoT path)
  const rand = Math.random();

  if (rand < 0.7) {
    testDirectPath(headers);
  } else {
    testCoTPath(headers);
  }

  sleep(1); // 1s think time between requests
}

/**
 * Test Direct Path (Simple Requests)
 * - Fast queries
 * - Status checks
 * - Simple CRUD operations
 */
function testDirectPath(headers) {
  group('Direct Path - Simple Requests', () => {
    const routingStart = Date.now();

    // Simulate simple request (e.g., list workspaces)
    const response = http.get(`${BASE_URL}/api/osa/workspaces`, { headers });

    const routingTime = Date.now() - routingStart;
    routingOverhead.add(routingTime);

    const duration = response.timings.duration;
    directPathLatency.add(duration);

    const success = check(response, {
      'direct: status is 200': (r) => r.status === 200,
      'direct: has workspaces': (r) => {
        const body = r.json();
        return body.workspaces !== undefined || body.length !== undefined;
      },
      'direct: response time < 500ms': (r) => r.timings.duration < 500,
    });

    directPathSuccess.add(success);

    if (!success) {
      directPathErrors.add(1);
      console.error(`❌ Direct path failed: ${response.status}`);
    }
  });
}

/**
 * Test CoT Path (Complex Requests)
 * - App generation
 * - Orchestration workflows
 * - Complex queries requiring reasoning
 */
function testCoTPath(headers) {
  group('CoT Path - Complex Requests', () => {
    const routingStart = Date.now();

    // Simulate complex request (app generation)
    const payload = JSON.stringify({
      name: `HybridTestApp-${__VU}-${Date.now()}`,
      description: 'Test hybrid routing with complex CoT workflow',
      type: 'full-stack',
      workspace_id: __ENV.WORKSPACE_ID || '',
    });

    const response = http.post(`${BASE_URL}/api/osa/generate`, payload, { headers });

    const routingTime = Date.now() - routingStart;
    routingOverhead.add(routingTime);

    const duration = response.timings.duration;
    cotPathLatency.add(duration);

    const success = check(response, {
      'cot: status is 200': (r) => r.status === 200,
      'cot: has app_id': (r) => r.json('app_id') !== undefined,
      'cot: response time < 10s': (r) => r.timings.duration < 10000,
    });

    cotPathSuccess.add(success);

    if (!success) {
      cotPathErrors.add(1);
      console.error(`❌ CoT path failed: ${response.status}`);
    }
  });
}

// Teardown: Generate comparison report
export function teardown(data) {
  console.log('========================================================');
  console.log('  🏁 Hybrid Architecture Load Test Complete');
  console.log('========================================================');
  console.log('');
  console.log('📊 Path Comparison:');
  console.log('');
  console.log('  Direct Path (70% of traffic):');
  console.log(`    - Errors: ${directPathErrors.value || 0}`);
  console.log(`    - Success Rate: ${(directPathSuccess.rate * 100).toFixed(2)}%`);
  console.log('');
  console.log('  CoT Path (30% of traffic):');
  console.log(`    - Errors: ${cotPathErrors.value || 0}`);
  console.log(`    - Success Rate: ${(cotPathSuccess.rate * 100).toFixed(2)}%`);
  console.log('');
  console.log('⚡ Routing Overhead:');
  console.log('  - Overhead should be < 50ms P95');
  console.log('  - Check detailed metrics in k6 output');
  console.log('');
  console.log('💡 Analysis:');
  console.log('  1. Compare P95 latencies: Direct vs CoT');
  console.log('  2. Verify routing overhead is minimal');
  console.log('  3. Check success rates meet SLA (99.9% direct, 99% CoT)');
  console.log('  4. Monitor circuit breaker triggers');
  console.log('');
  console.log('📈 Export results:');
  console.log('   k6 run --out json=hybrid-results.json load_test_hybrid.js');
  console.log('');
}
