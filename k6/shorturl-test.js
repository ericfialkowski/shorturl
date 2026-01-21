import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const createUrlDuration = new Trend('create_url_duration');
const redirectDuration = new Trend('redirect_duration');
const statsDuration = new Trend('stats_duration');
const urlsCreated = new Counter('urls_created');
const redirectsPerformed = new Counter('redirects_performed');

// Configuration - override with environment variables
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8800';

// Test options - can be overridden via CLI
export const options = {
  scenarios: {
    // Smoke test - quick validation
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '30s',
      startTime: '0s',
      tags: { test_type: 'smoke' },
      exec: 'smokeTest',
    },
    // Load test - normal expected load
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 10 },
        { duration: '1m', target: 10 },
        { duration: '30s', target: 0 },
      ],
      startTime: '35s',
      tags: { test_type: 'load' },
      exec: 'loadTest',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.05'],
    create_url_duration: ['p(95)<300'],
    redirect_duration: ['p(95)<200'],
    stats_duration: ['p(95)<200'],
  },
};

// Test URLs to use
const testUrls = [
  'https://example.com/page1',
  'https://example.com/page2',
  'https://github.com/test/repo',
  'https://google.com/search?q=test',
  'https://stackoverflow.com/questions/12345',
];

// Store created abbreviations for cleanup
const createdAbvs = [];

// Helper function to create a short URL
function createShortUrl(url) {
  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const startTime = Date.now();
  const res = http.post(BASE_URL + '/', JSON.stringify(url), params);
  createUrlDuration.add(Date.now() - startTime);

  const success = check(res, {
    'create: status is 200': (r) => r.status === 200,
    'create: has abbreviation': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.abv !== undefined && body.abv.length > 0;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);

  if (success) {
    urlsCreated.add(1);
    try {
      return JSON.parse(res.body);
    } catch {
      return null;
    }
  }
  return null;
}

// Helper function to perform redirect
function performRedirect(abv) {
  const startTime = Date.now();
  const res = http.get(BASE_URL + '/' + abv, { redirects: 0 });
  redirectDuration.add(Date.now() - startTime);

  const success = check(res, {
    'redirect: status is 302': (r) => r.status === 302,
    'redirect: has location header': (r) => r.headers['Location'] !== undefined,
  });

  errorRate.add(!success);

  if (success) {
    redirectsPerformed.add(1);
  }

  return success;
}

// Helper function to get stats
function getStats(abv) {
  const startTime = Date.now();
  const res = http.get(BASE_URL + '/' + abv + '/stats');
  statsDuration.add(Date.now() - startTime);

  const success = check(res, {
    'stats: status is 200': (r) => r.status === 200,
    'stats: has abbreviation': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.abbreviation === abv;
      } catch {
        return false;
      }
    },
    'stats: has hits count': (r) => {
      try {
        const body = JSON.parse(r.body);
        return typeof body.hits === 'number';
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
  return success;
}

// Helper function to delete a short URL
function deleteShortUrl(abv) {
  const res = http.del(BASE_URL + '/' + abv);

  const success = check(res, {
    'delete: status is 200': (r) => r.status === 200,
  });

  errorRate.add(!success);
  return success;
}

// Health check
function checkHealth() {
  const res = http.get(BASE_URL + '/diag/status');

  return check(res, {
    'health: status is 200': (r) => r.status === 200,
    'health: status_code is 0': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status_code === 0;
      } catch {
        return false;
      }
    },
  });
}

// Metrics check
function checkMetrics() {
  const res = http.get(BASE_URL + '/diag/metrics');

  return check(res, {
    'metrics: status is 200': (r) => r.status === 200,
    'metrics: has uptime': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.uptime !== undefined;
      } catch {
        return false;
      }
    },
  });
}

// Smoke test scenario - basic functionality validation
export function smokeTest() {
  group('Health Checks', () => {
    checkHealth();
    checkMetrics();
  });

  group('Create and Use Short URL', () => {
    const testUrl = testUrls[Math.floor(Math.random() * testUrls.length)] + '/' + Date.now();
    const result = createShortUrl(testUrl);

    if (result && result.abv) {
      sleep(0.5);

      // Test redirect
      performRedirect(result.abv);
      sleep(0.5);

      // Test stats
      getStats(result.abv);
      sleep(0.5);

      // Test stats UI (just check it returns HTML)
      const statsUiRes = http.get(BASE_URL + result.stats_ui_link);
      check(statsUiRes, {
        'stats UI: status is 200': (r) => r.status === 200,
        'stats UI: returns HTML': (r) => r.headers['Content-Type']?.includes('text/html'),
      });

      // Cleanup
      deleteShortUrl(result.abv);
    }
  });

  sleep(1);
}

// Load test scenario - sustained traffic
export function loadTest() {
  const iteration = __ITER;
  const vu = __VU;

  // Create a unique URL for this VU/iteration
  const testUrl = `https://example.com/load-test/vu${vu}/iter${iteration}/${Date.now()}`;

  group('Create Short URL', () => {
    const result = createShortUrl(testUrl);

    if (result && result.abv) {
      group('Use Short URL', () => {
        // Perform multiple redirects
        for (let i = 0; i < 3; i++) {
          performRedirect(result.abv);
          sleep(0.1);
        }

        // Check stats
        getStats(result.abv);
      });

      // Cleanup created URL
      deleteShortUrl(result.abv);
    }
  });

  sleep(0.5);
}

// Default function (runs if no scenario specified)
export default function () {
  smokeTest();
}

// Setup function - runs once at the start
export function setup() {
  console.log(`Testing against: ${BASE_URL}`);

  // Verify service is available
  const healthRes = http.get(BASE_URL + '/diag/status');
  if (healthRes.status !== 200) {
    throw new Error(`Service not available at ${BASE_URL}. Status: ${healthRes.status}`);
  }

  console.log('Service is healthy, starting tests...');

  return { baseUrl: BASE_URL };
}

// Teardown function - runs once at the end
export function teardown(data) {
  console.log('Tests completed.');
}