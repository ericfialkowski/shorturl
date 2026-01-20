# Running ShortURL

This document describes how to run the ShortURL service with various database configurations.

## Quick Start with Docker Compose

The easiest way to run the service is using Docker Compose with one of the available profiles:

```bash
cd docker

# MongoDB
docker compose --profile mongo up --build

# PostgreSQL
docker compose --profile postgres up --build

# MySQL
docker compose --profile mysql up --build

# In-Memory (no persistence)
docker compose --profile memory up --build
```

The service will be available at `http://localhost:8800`.

### Admin UIs

Each database profile includes an admin UI:

| Profile    | Admin UI                  | Credentials                     |
|------------|---------------------------|---------------------------------|
| mongo      | http://localhost:8081     | dbadmin / p@ssw0rd!             |
| postgres   | http://localhost:8082     | (auto-configured)               |
| mysql      | http://localhost:8083     | (auto-configured)               |

## Running Locally

Build and run the service directly:

```bash
go build -o shorturl .
./shorturl
```

By default, the service runs with an in-memory database on port 8800.

## Database Configurations

Only one database option can be configured at a time. If multiple are set, the service will exit with an error.

### In-Memory (Default)

No configuration required. Data is not persisted across restarts.

```bash
./shorturl
```

### MongoDB

```bash
export mongo_uri="mongodb://user:password@localhost:27017/admin"
./shorturl
```

### PostgreSQL

```bash
export postgres_uri="postgres://user:password@localhost:5432/shorturl"
./shorturl
```

### MySQL

```bash
export mysql_dsn="user:password@tcp(localhost:3306)/shorturl"
./shorturl
```

Note: The MySQL driver automatically adds `parseTime=true` if not present.

### SQLite

```bash
# File-based
export sqlite_path="./shorturl.db"
./shorturl

# In-memory SQLite
export sqlite_path=":memory:"
./shorturl
```

## Environment Variables

### Server Configuration

| Variable                | Default   | Description                              |
|-------------------------|-----------|------------------------------------------|
| `port`                  | 8800      | HTTP server port                         |
| `ip`                    | ""        | IP address to bind to (empty = all)      |
| `logrequests`           | true      | Enable request logging                   |
| `status_interval`       | 30s       | Health check interval                    |
| `http_write_timeout`    | 10s       | HTTP write timeout                       |
| `http_read_timeout`     | 15s       | HTTP read timeout                        |
| `http_idle_timeout`     | 60s       | HTTP idle timeout                        |
| `shutdown_wait_timeout` | 15s       | Graceful shutdown timeout                |

### Database Connection Strings

| Variable       | Description                                          |
|----------------|------------------------------------------------------|
| `mongo_uri`    | MongoDB connection URI                               |
| `postgres_uri` | PostgreSQL connection URI                            |
| `mysql_dsn`    | MySQL DSN (Data Source Name)                         |
| `sqlite_path`  | Path to SQLite database file (or `:memory:`)         |

### Database-Specific Options

#### PostgreSQL

| Variable            | Default | Description                |
|---------------------|---------|----------------------------|
| `postgres_timeout`  | 10s     | Query timeout              |
| `postgres_max_conns`| 10      | Maximum connections        |

#### MySQL

| Variable                        | Default | Description                      |
|---------------------------------|---------|----------------------------------|
| `mysql_timeout`                 | 10s     | Query timeout                    |
| `mysql_max_conns`               | 10      | Maximum open connections         |
| `mysql_max_idle_conns`          | 5       | Maximum idle connections         |
| `mysql_conn_max_lifetime_minutes`| 5      | Connection max lifetime (minutes)|

#### MongoDB

| Variable        | Default | Description    |
|-----------------|---------|----------------|
| `mongo_timeout` | 10s     | Query timeout  |

### URL Abbreviation

| Variable          | Default | Description                                    |
|-------------------|---------|------------------------------------------------|
| `startingkeysize` | 1       | Initial length of generated abbreviations      |
| `keygrowretries`  | 10      | Retries before increasing abbreviation length  |

### OpenTelemetry

| Variable                     | Default                 | Description                    |
|------------------------------|-------------------------|--------------------------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT`| http://localhost:4318   | OTLP exporter endpoint         |
| `OTEL_SERVICE_NAME`          | shorturl                | Service name for telemetry     |
| `OTEL_METRICS_ENABLED`       | true                    | Enable/disable OTEL metrics    |

**Local Debugging:** To run locally without an OTLP collector, disable metrics to avoid connection errors:

```bash
export OTEL_METRICS_ENABLED=false
./shorturl
```

## API Endpoints

| Method | Path           | Description                      |
|--------|----------------|----------------------------------|
| POST   | /              | Create a short URL               |
| GET    | /:abv          | Redirect to original URL         |
| DELETE | /:abv          | Delete a short URL               |
| GET    | /:abv/stats    | Get statistics for a short URL   |
| GET    | /:abv/stats/ui | View statistics in HTML          |
| GET    | /diag/status   | Health check endpoint            |
| GET    | /diag/metrics  | Service metrics                  |

## Examples

### Create a short URL

```bash
curl -X POST http://localhost:8800/ \
  -H "Content-Type: application/json" \
  -d '"https://example.com/very/long/url"'
```

Response:
```json
{
  "abv": "a",
  "urlLink": "/a",
  "statsLink": "/a/stats",
  "statsUiLink": "/a/stats/ui"
}
```

### Access the short URL

```bash
curl -L http://localhost:8800/a
```

### Get statistics

```bash
curl http://localhost:8800/a/stats
```

### Delete a short URL

```bash
curl -X DELETE http://localhost:8800/a
```
