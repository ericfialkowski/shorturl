import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8800';

// Single iteration functional test
export const options = {
  vus: 1,
  iterations: 1,
  thresholds: {
    errors: ['rate==0'],
    // Allow ~20% HTTP "failures" since we test 400/404 error responses intentionally
    http_req_failed: ['rate<0.25'],
  },
};

// Helper to make JSON POST request
function postJson(path, body) {
  return http.post(BASE_URL + path, JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json' },
  });
}

export default function () {
  let createdAbv = null;

  group('1. Health Endpoints', () => {
    group('GET /diag/status - Health Check', () => {
      const res = http.get(BASE_URL + '/diag/status');
      const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'response has status_code': (r) => JSON.parse(r.body).status_code !== undefined,
        'response has status_msg': (r) => JSON.parse(r.body).status_msg !== undefined,
        'response has timestamp': (r) => JSON.parse(r.body).timestamp !== undefined,
        'status_code is 0 (OK)': (r) => JSON.parse(r.body).status_code === 0,
      });
      errorRate.add(!success);
    });

    group('GET /diag/metrics - Metrics', () => {
      const res = http.get(BASE_URL + '/diag/metrics');
      const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'has redirect_counts': (r) => JSON.parse(r.body).redirect_counts !== undefined,
        'has new_url_counts': (r) => JSON.parse(r.body).new_url_counts !== undefined,
        'has uptime': (r) => JSON.parse(r.body).uptime !== undefined,
      });
      errorRate.add(!success);
    });
  });

  group('2. Create Short URL - POST /', () => {
    group('Valid URL creation', () => {
      const testUrl = 'https://example.com/test/' + Date.now();
      const res = postJson('/', testUrl);
      const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'has abv field': (r) => JSON.parse(r.body).abv !== undefined,
        'has url_link field': (r) => JSON.parse(r.body).url_link !== undefined,
        'has stats_link field': (r) => JSON.parse(r.body).stats_link !== undefined,
        'has stats_ui_link field': (r) => JSON.parse(r.body).stats_ui_link !== undefined,
        'abv is not empty': (r) => JSON.parse(r.body).abv.length > 0,
      });
      errorRate.add(!success);

      if (success) {
        createdAbv = JSON.parse(res.body).abv;
      }
    });

    group('Duplicate URL returns same abbreviation', () => {
      const testUrl = 'https://example.com/duplicate-test/' + Date.now();

      const res1 = postJson('/', testUrl);
      const res2 = postJson('/', testUrl);

      const success = check(
        { res1, res2 },
        {
          'first request succeeds': () => res1.status === 200,
          'second request succeeds': () => res2.status === 200,
          'same abbreviation returned': () => {
            const abv1 = JSON.parse(res1.body).abv;
            const abv2 = JSON.parse(res2.body).abv;
            return abv1 === abv2;
          },
        }
      );
      errorRate.add(!success);

      // Cleanup
      if (res1.status === 200) {
        http.del(BASE_URL + '/' + JSON.parse(res1.body).abv);
      }
    });

    group('Empty URL returns 400', () => {
      const res = postJson('/', '');
      const success = check(res, {
        'status is 400': (r) => r.status === 400,
      });
      errorRate.add(!success);
    });

    group('Invalid JSON returns error', () => {
      const res = http.post(BASE_URL + '/', 'not valid json', {
        headers: { 'Content-Type': 'application/json' },
      });
      const success = check(res, {
        'status is 400 or 500': (r) => r.status === 400 || r.status === 500,
      });
      errorRate.add(!success);
    });
  });

  group('3. Redirect - GET /:abv', () => {
    // Create a URL first
    const testUrl = 'https://httpbin.org/get';
    const createRes = postJson('/', testUrl);
    const abv = JSON.parse(createRes.body).abv;

    group('Valid abbreviation redirects', () => {
      const res = http.get(BASE_URL + '/' + abv, { redirects: 0 });
      const success = check(res, {
        'status is 302': (r) => r.status === 302,
        'Location header present': (r) => r.headers['Location'] !== undefined,
        'Location matches original URL': (r) => r.headers['Location'] === testUrl,
        'has x-instance-uuid header': (r) => r.headers['X-Instance-Uuid'] !== undefined,
      });
      errorRate.add(!success);
    });

    group('Non-existent abbreviation returns 404', () => {
      const res = http.get(BASE_URL + '/nonexistent999xyz', { redirects: 0 });
      const success = check(res, {
        'status is 404': (r) => r.status === 404,
      });
      errorRate.add(!success);
    });

    // Cleanup
    http.del(BASE_URL + '/' + abv);
  });

  group('4. Statistics - GET /:abv/stats', () => {
    // Create a URL and access it a few times
    const testUrl = 'https://example.com/stats-test/' + Date.now();
    const createRes = postJson('/', testUrl);
    const abv = JSON.parse(createRes.body).abv;

    // Access the URL twice
    http.get(BASE_URL + '/' + abv, { redirects: 0 });
    http.get(BASE_URL + '/' + abv, { redirects: 0 });

    // Small delay to ensure hits are persisted
    sleep(0.1);

    group('Stats JSON returns correct data', () => {
      const res = http.get(BASE_URL + '/' + abv + '/stats');
      const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'has abbreviation': (r) => JSON.parse(r.body).abbreviation === abv,
        'has url': (r) => JSON.parse(r.body).url === testUrl,
        'has hits field': (r) => JSON.parse(r.body).hits !== undefined,
        'has last_access': (r) => JSON.parse(r.body).last_access !== undefined,
        'daily_hits has entries': (r) => {
          const body = JSON.parse(r.body);
          const totalDailyHits = Object.values(body.daily_hits || {}).reduce((a, b) => a + b, 0);
          return totalDailyHits >= 2;
        },
      });
      errorRate.add(!success);
    });

    group('Stats for non-existent returns 404', () => {
      const res = http.get(BASE_URL + '/nonexistent999xyz/stats');
      const success = check(res, {
        'status is 404': (r) => r.status === 404,
      });
      errorRate.add(!success);
    });

    // Cleanup
    http.del(BASE_URL + '/' + abv);
  });

  group('5. Statistics UI - GET /:abv/stats/ui', () => {
    // Create a URL
    const testUrl = 'https://example.com/stats-ui-test/' + Date.now();
    const createRes = postJson('/', testUrl);
    const abv = JSON.parse(createRes.body).abv;

    group('Stats UI returns HTML', () => {
      const res = http.get(BASE_URL + '/' + abv + '/stats/ui');
      const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'content-type is HTML': (r) => r.headers['Content-Type']?.includes('text/html'),
        'contains abbreviation': (r) => r.body.includes(abv),
      });
      errorRate.add(!success);
    });

    // Cleanup
    http.del(BASE_URL + '/' + abv);
  });

  group('6. Delete - DELETE /:abv', () => {
    // Create a URL to delete
    const testUrl = 'https://example.com/delete-test/' + Date.now();
    const createRes = postJson('/', testUrl);
    const abv = JSON.parse(createRes.body).abv;

    group('Delete existing URL succeeds', () => {
      const res = http.del(BASE_URL + '/' + abv);
      const success = check(res, {
        'status is 200': (r) => r.status === 200,
      });
      errorRate.add(!success);
    });

    group('Deleted URL no longer accessible', () => {
      const res = http.get(BASE_URL + '/' + abv, { redirects: 0 });
      const success = check(res, {
        'status is 404': (r) => r.status === 404,
      });
      errorRate.add(!success);
    });
  });

  group('7. Home Page - GET /', () => {
    const res = http.get(BASE_URL + '/');
    const success = check(res, {
      'status is 200': (r) => r.status === 200,
      'content-type is HTML': (r) => r.headers['Content-Type']?.includes('text/html'),
    });
    errorRate.add(!success);
  });

  // Cleanup the first created URL if it still exists
  if (createdAbv) {
    http.del(BASE_URL + '/' + createdAbv);
  }
}

export function setup() {
  console.log(`Running functional tests against: ${BASE_URL}`);

  const healthRes = http.get(BASE_URL + '/diag/status');
  if (healthRes.status !== 200) {
    throw new Error(`Service not available at ${BASE_URL}`);
  }

  return {};
}

export function teardown(data) {
  console.log('Functional tests completed.');
}
