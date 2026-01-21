import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const createErrors = new Counter('create_errors');
const redirectErrors = new Counter('redirect_errors');
const responseTime = new Trend('response_time');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8800';

// Stress test options - ramp up to high load
export const options = {
  stages: [
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '2m', target: 50 },   // Stay at 50 users
    { duration: '1m', target: 100 },  // Ramp up to 100 users
    { duration: '2m', target: 100 },  // Stay at 100 users
    { duration: '1m', target: 150 },  // Ramp up to 150 users
    { duration: '2m', target: 150 },  // Stay at 150 users
    { duration: '2m', target: 0 },    // Ramp down to 0
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000', 'p(99)<2000'],
    http_req_failed: ['rate<0.05'],
    errors: ['rate<0.1'],
  },
};

// Create short URL helper
function createShortUrl() {
  const testUrl = `https://example.com/stress/${__VU}/${__ITER}/${Date.now()}`;

  const res = http.post(
    BASE_URL + '/',
    JSON.stringify(testUrl),
    { headers: { 'Content-Type': 'application/json' } }
  );

  responseTime.add(res.timings.duration);

  const success = check(res, {
    'create: status 200': (r) => r.status === 200,
  });

  if (!success) {
    createErrors.add(1);
    errorRate.add(1);
    return null;
  }

  errorRate.add(0);

  try {
    return JSON.parse(res.body).abv;
  } catch {
    return null;
  }
}

// Redirect helper
function redirect(abv) {
  const res = http.get(BASE_URL + '/' + abv, { redirects: 0 });

  responseTime.add(res.timings.duration);

  const success = check(res, {
    'redirect: status 302': (r) => r.status === 302,
  });

  if (!success) {
    redirectErrors.add(1);
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }

  return success;
}

// Delete helper
function deleteUrl(abv) {
  const res = http.del(BASE_URL + '/' + abv);
  responseTime.add(res.timings.duration);
  return res.status === 200;
}

export default function () {
  // Create a short URL
  const abv = createShortUrl();

  if (abv) {
    sleep(0.1);

    // Perform redirects (simulates real usage)
    for (let i = 0; i < 5; i++) {
      redirect(abv);
      sleep(0.05);
    }

    // Cleanup
    deleteUrl(abv);
  }

  sleep(0.2);
}

export function setup() {
  console.log(`Starting stress test against: ${BASE_URL}`);

  const healthRes = http.get(BASE_URL + '/diag/status');
  if (healthRes.status !== 200) {
    throw new Error(`Service not available at ${BASE_URL}`);
  }

  return {};
}

export function teardown(data) {
  console.log('Stress test completed.');
}
