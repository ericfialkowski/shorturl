# k6 Test Suite for ShortURL Service

Load and functional testing suite using [k6](https://k6.io/).

## Prerequisites

Install k6: https://k6.io/docs/get-started/installation/

```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

## Test Files

| File | Purpose |
|------|---------|
| `shorturl-test.js` | Main test suite with smoke and load test scenarios |
| `functional-test.js` | Single-iteration functional tests for all endpoints |
| `stress-test.js` | High-load stress testing |

## Running Tests

### Functional Tests (Recommended First)

Run single-iteration tests to verify all endpoints work correctly:

```bash
k6 run k6/functional-test.js
```

### Smoke Test

Quick validation with minimal load:

```bash
k6 run --env BASE_URL=http://localhost:8800 k6/shorturl-test.js
```

### Load Test

Normal load simulation:

```bash
k6 run k6/shorturl-test.js
```

### Stress Test

High concurrency test:

```bash
k6 run k6/stress-test.js
```

### Custom Configuration

Override the base URL:

```bash
k6 run --env BASE_URL=http://your-server:8800 k6/shorturl-test.js
```

Run specific scenario only:

```bash
k6 run --env BASE_URL=http://localhost:8800 -e K6_SCENARIO=smoke k6/shorturl-test.js
```

Adjust VUs and duration:

```bash
k6 run --vus 20 --duration 2m k6/shorturl-test.js
```

### Using Docker

```bash
docker run --rm -i --network host grafana/k6 run - <k6/functional-test.js
```

## Test Coverage

The test suite covers:

- **Health Endpoints**
  - `GET /diag/status` - Service health check
  - `GET /diag/metrics` - Service metrics

- **URL Creation**
  - `POST /` - Create short URL (valid URLs)
  - `POST /` - Duplicate URL handling
  - `POST /` - Invalid/empty URL error handling

- **Redirect**
  - `GET /:abv` - Valid abbreviation redirect
  - `GET /:abv` - Non-existent abbreviation (404)

- **Statistics**
  - `GET /:abv/stats` - JSON statistics
  - `GET /:abv/stats/ui` - HTML statistics page

- **Deletion**
  - `DELETE /:abv` - Delete short URL
  - Verification that deleted URLs return 404

## Thresholds

Default performance thresholds:

| Metric | Threshold |
|--------|-----------|
| HTTP request duration (p95) | < 500ms |
| HTTP request duration (p99) | < 1000ms |
| HTTP request failure rate | < 1% |
| Error rate | < 5% |

## Output & Reporting

Generate HTML report:

```bash
k6 run --out json=results.json k6/shorturl-test.js
# Then use k6-reporter or similar tool
```

Output to InfluxDB for Grafana dashboards:

```bash
k6 run --out influxdb=http://localhost:8086/k6 k6/shorturl-test.js
```
