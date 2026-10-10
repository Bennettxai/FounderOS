/**
 * k6 Load Test - OSA Endpoints
 *
 * Tests OSA integration endpoints under various load scenarios:
 * - /api/osa/generate (app generation)
 * - /api/osa/status/:id (status polling)
 * - /api/osa/orchestrate (orchestration workflow)
 *
 * Scenarios: 100, 500, 1000 req/s
 * Duration: 5 minutes each
 * Metrics: P50, P95, P99 latency, error rate
 */

/**
 * Required environment variables:
 *   BASE_URL            - Target server URL (default: http://localhost:8001)
 *   AUTH_TOKEN          - Bearer token for authenticated requests (preferred)
 *   TEST_LOGIN_EMAIL    - Login email used when AUTH_TOKEN is not set
 *   TEST_LOGIN_PASSWORD - Login password used when AUTH_TOKEN is not set
 *   WORKSPACE_ID        - Workspace ID for generation/orchestration payloads
 *   TEST_APP_ID         - App ID for status polling (default: zero UUID)
 *
 * Example:
 *   k6 run -e AUTH_TOKEN=<token> -e BASE_URL=http://localhost:8001 load_test_osa.js
 *   k6 run -e TEST_LOGIN_EMAIL=user@example.com -e TEST_LOGIN_PASSWORD=secret load_test_osa.js
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const osaGenerateErrors = new Counter('osa_generate_errors');
const osaStatusErrors = new Counter('osa_status_errors');
const osaOrchestrateErrors = new Counter('osa_orchestrate_errors');
const osaGenerateLatency = new Trend('osa_generate_latency');
const osaStatusLatency = new Trend('osa_status_latency');
const osaOrchestrateLatency = new Trend('osa_orchestrate_latency');
const successRate = new Rate('success_rate');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8001';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

// Test scenarios
export const options = {
  scenarios: {
    // Scenario 1: Low load (100 req/s)
    low_load: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 50,
      maxVUs: 200,
      tags: { scenario: 'low_load' },
      exec: 'testOSAEndpoints',
    },

    // Scenario 2: Medium load (500 req/s)
    medium_load: {
      executor: 'constant-arrival-rate',
      rate: 500,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 250,
      maxVUs: 1000,
      tags: { scenario: 'medium_load' },
      exec: 'testOSAEndpoints',
      startTime: '6m', // Start after low_load finishes
    },

    // Scenario 3: High load (1000 req/s)
    high_load: {
      executor: 'constant-arrival-rate',
      rate: 1000,
      timeUnit: '1s',
      duration: '5m',
      preAllocatedVUs: 500,
      maxVUs: 2000,
      tags: { scenario: 'high_load' },
      exec: 'testOSAEndpoints',
      startTime: '12m', // Start after medium_load finishes
    },
  },

  // Performance thresholds
  thresholds: {
    // Overall HTTP metrics
    'http_req_duration': [
      'p(50)<200',  // P50 latency < 200ms
      'p(95)<500',  // P95 latency < 500ms
      'p(99)<1000', // P99 latency < 1s
    ],
    'http_req_failed': ['rate<0.01'], // Error rate < 1%

    // OSA-specific metrics
    'osa_generate_latency': ['p(95)<2000'], // P95 < 2s for generation
    'osa_status_latency': ['p(95)<200'],    // P95 < 200ms for status
    'osa_orchestrate_latency': ['p(95)<3000'], // P95 < 3s for orchestration

    'success_rate': ['rate>0.999'], // 99.9% success rate

    // Scenario-specific thresholds
    'http_req_duration{scenario:low_load}': ['p(95)<300'],
    'http_req_duration{scenario:medium_load}': ['p(95)<500'],
    'http_req_duration{scenario:high_load}': ['p(95)<1000'],
  },
};

// Setup: Authenticate and get token
export function setup() {
  if (!AUTH_TOKEN) {
    console.log('⚠️  No AUTH_TOKEN provided, using test credentials');

    // Login to get auth token
    const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
      email: __ENV.TEST_LOGIN_EMAIL,
      password: __ENV.TEST_LOGIN_PASSWORD,
    }), {
      headers: { 'Content-Type': 'application/json' },
    });

    if (loginRes.status === 200) {
      const token = loginRes.json('token');
      console.log('✅ Authentication successful');
      return { token: token };
    } else {
      console.error('❌ Authentication failed:', loginRes.status, loginRes.body);
      return { token: null };
    }
  }

  return { token: AUTH_TOKEN };
}

// Test OSA endpoints
export function testOSAEndpoints(data) {
  const token = data.token;

  if (!token) {
    console.error('❌ No auth token available, skipping test');
    return;
  }

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };

  // Distribute load across endpoints
  const rand = Math.random();

  if (rand < 0.4) {
    // 40% - Test app generation endpoint
    testGenerateApp(headers);
  } else if (rand < 0.8) {
    // 40% - Test status polling
    testGetStatus(headers);
  } else {
    // 20% - Test orchestration
    testOrchestrate(headers);
  }

  sleep(0.1); // 100ms think time
}

// Test /api/osa/generate
function testGenerateApp(headers) {
  group('POST /api/osa/generate', () => {
    const payload = JSON.stringify({
      name: `TestApp-${__VU}-${Date.now()}`,
      description: 'k6 load test app generation',
      type: 'full-stack',
      workspace_id: __ENV.WORKSPACE_ID || '',
    });

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/api/osa/generate`, payload, { headers });
    const duration = Date.now() - startTime;

    const success = check(response, {
      'generate: status is 200': (r) => r.status === 200,
      'generate: has app_id': (r) => r.json('app_id') !== undefined,
      'generate: response time OK': (r) => r.timings.duration < 5000,
    });

    osaGenerateLatency.add(duration);
    successRate.add(success);

    if (!success) {
      osaGenerateErrors.add(1);
      console.error(`❌ Generate failed: ${response.status} - ${response.body}`);
    }
  });
}

// Test /api/osa/status/:id
function testGetStatus(headers) {
  group('GET /api/osa/status/:id', () => {
    // Use a mock app ID or get from environment
    const appId = __ENV.TEST_APP_ID || '00000000-0000-0000-0000-000000000001';

    const startTime = Date.now();
    const response = http.get(`${BASE_URL}/api/osa/status/${appId}`, { headers });
    const duration = Date.now() - startTime;

    const success = check(response, {
      'status: response is OK': (r) => r.status === 200 || r.status === 404,
      'status: has status field': (r) => r.status === 404 || r.json('status') !== undefined,
      'status: response time OK': (r) => r.timings.duration < 500,
    });

    osaStatusLatency.add(duration);
    successRate.add(success);

    if (!success) {
      osaStatusErrors.add(1);
      console.error(`❌ Status check failed: ${response.status} - ${response.body}`);
    }
  });
}

// Test /api/osa/orchestrate
function testOrchestrate(headers) {
  group('POST /api/osa/orchestrate', () => {
    const payload = JSON.stringify({
      user_id: `user-${__VU}`,
      input: 'Generate a simple task management app',
      context: {
        workspace_id: __ENV.WORKSPACE_ID || '',
      },
    });

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/api/osa/orchestrate`, payload, { headers });
    const duration = Date.now() - startTime;

    const success = check(response, {
      'orchestrate: status is 200': (r) => r.status === 200,
      'orchestrate: has result': (r) => r.body.length > 0,
      'orchestrate: response time OK': (r) => r.timings.duration < 5000,
    });

    osaOrchestrateLatency.add(duration);
    successRate.add(success);

    if (!success) {
      osaOrchestrateErrors.add(1);
      console.error(`❌ Orchestrate failed: ${response.status} - ${response.body}`);
    }
  });
}

// Teardown: Log summary
export function teardown(data) {
  console.log('================================================');
  console.log('  🏁 OSA Load Test Complete');
  console.log('================================================');
  console.log('');
  console.log('📊 Custom Metrics Summary:');
  console.log(`  - Generate Errors: ${osaGenerateErrors.value || 0}`);
  console.log(`  - Status Errors: ${osaStatusErrors.value || 0}`);
  console.log(`  - Orchestrate Errors: ${osaOrchestrateErrors.value || 0}`);
  console.log('');
  console.log('💡 Tip: Check k6 HTML report for detailed results');
  console.log('   Run with: k6 run --out json=results.json load_test_osa.js');
  console.log('');
}

// Default export for single scenario execution
export default function() {
  testOSAEndpoints({ token: AUTH_TOKEN });
}
