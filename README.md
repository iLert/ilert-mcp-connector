# ilert MCP Connector

A HTTP server that exposes information from internal tools (Kafka, MySQL) via REST endpoints. Designed to be used with ilert and MCP servers.

## Features

- **Health and Readiness Endpoints**: Kubernetes-ready with `/health` and `/ready` endpoints
- **Kafka Integration**: List topics, describe topics, manage consumer groups, and monitor consumer lag
- **MySQL Integration**: Query databases, tables, schemas, and metrics
- **Configurable**: Enable/disable tools individually via environment variables
- **Zero Logging**: Minimal logging for production use
- **Docker Support**: Containerized and ready for Kubernetes deployment

## Endpoints

### Health & Readiness

- `GET /health` - Health check endpoint (Public - No Auth Required)
- `GET /ready` - Readiness check endpoint (verifies connections to enabled tools) - **Requires Authentication**
- `GET /version` - Version information endpoint (returns version and commit) (Public - No Auth Required)

### Authentication

The following endpoints require authentication via the `Authorization` header:
- `/ready` - Readiness check endpoint (contains connection status and potential error messages)
- All `/kafka/*` endpoints
- All `/mysql/*` endpoints
- All `/clickhouse/*` endpoints

The token can be provided in two ways:

1. **Environment Variable**: Set `AUTH_TOKEN` environment variable with your desired token
2. **Auto-generated**: If `AUTH_TOKEN` is not set, a random 64-character hex token will be generated and logged on startup

**Usage:**
```bash
# Using Bearer token format
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8383/kafka/topics

# Or without Bearer prefix
curl -H "Authorization: YOUR_TOKEN" http://localhost:8383/kafka/topics
```

### Kafka Endpoints (when `KAFKA_ENABLED=true`) - **Requires Authentication**

- `GET /kafka/topics` - List all Kafka topics
- `GET /kafka/topics/:topic` - Describe a specific topic
- `GET /kafka/consumers` - List all consumer groups
- `GET /kafka/consumers/:group` - Describe a specific consumer group
- `GET /kafka/consumers/:group/lag` - Get consumer lag for a consumer group

### MySQL Endpoints (when `MYSQL_ENABLED=true`) - **Requires Authentication**

- `GET /mysql/databases` - List all databases
- `GET /mysql/databases/:database/tables` - List tables in a database
- `GET /mysql/databases/:database/tables/:table` - Describe table schema
- `GET /mysql/metrics` - Get MySQL metrics (InnoDB stats, global status, variables)

### ClickHouse Endpoints (when `CLICKHOUSE_ENABLED=true`) - **Requires Authentication**

- `GET /clickhouse/databases` - List all databases
- `GET /clickhouse/databases/:database/tables` - List tables in a database
- `GET /clickhouse/databases/:database/tables/:table` - Describe table schema with columns and table info
- `GET /clickhouse/metrics` - Get ClickHouse state and critical metrics (system.metrics, system.events, system.asynchronous_metrics, replicas, processes, merges, mutations)

## Configuration

Configuration is done via environment variables:

### Server Configuration

- `PORT` - Server port (default: `8383`)
- `LOG_LEVEL` - Log level: `debug`, `info`, `warn`, `error`, `fatal`, `panic` (default: `info`)
- `LOG_FORMAT` - Log format: `json` for JSON output, or empty/any other value for pretty console output (default: pretty console)
- `AUTH_TOKEN` - Authentication token for protecting endpoints. If not set, a random token will be generated and logged on startup (default: auto-generated)

### Kafka Configuration

- `KAFKA_ENABLED` - Enable Kafka endpoints (default: `false`)
- `KAFKA_BROKERS` - Comma-separated list of Kafka brokers (default: `localhost:9092`)
- `KAFKA_CLIENT_ID` - Kafka client ID (default: `ilert-mcp-connector`)

### MySQL Configuration

- `MYSQL_ENABLED` - Enable MySQL endpoints (default: `false`)
- `MYSQL_HOST` - MySQL host (default: `localhost`)
- `MYSQL_PORT` - MySQL port (default: `3306`)
- `MYSQL_USER` - MySQL user (default: `root`)
- `MYSQL_PASSWORD` - MySQL password (default: empty)
- `MYSQL_DATABASE` - MySQL database name (default: empty)

### ClickHouse Configuration

- `CLICKHOUSE_ENABLED` - Enable ClickHouse endpoints (default: `false`)
- `CLICKHOUSE_HOST` - ClickHouse host (default: `localhost`)
- `CLICKHOUSE_PORT` - ClickHouse port (default: `9000`)
- `CLICKHOUSE_USER` - ClickHouse user (default: `default`)
- `CLICKHOUSE_PASSWORD` - ClickHouse password (default: empty)
- `CLICKHOUSE_DATABASE` - ClickHouse database name (default: `default`)

## Building

### Using Make (Recommended)

```bash
# Build the application
make build

# Build and start the server
make start

# Run the application (without building)
make run

# Show all available commands
make help
```

### Manual Build

```bash
go build -o ilert-mcp-connector .
```

## Running Locally

```bash
# Set environment variables
export KAFKA_ENABLED=true
export KAFKA_BROKERS=localhost:9092
export MYSQL_ENABLED=true
export MYSQL_HOST=localhost
export MYSQL_USER=root
export MYSQL_PASSWORD=password

# Run the server
./ilert-mcp-connector
```

## Docker

### Build Docker Image

**Using Make:**
```bash
make docker-build
# Or with custom version
make docker-build VERSION=1.0.0 DOCKER_TAG=1.0.0
```

**Manual build:**
```bash
docker build -t ilert-mcp-connector .
```

### Run Docker Container

**Using Make:**
```bash
make docker-run
# Or with custom port
make docker-run PORT=8383
```

**Manual run:**
```bash
docker run -p 8383:8383 \
  -e KAFKA_ENABLED=true \
  -e KAFKA_BROKERS=kafka1:9092,kafka2:9092 \
  -e MYSQL_ENABLED=true \
  -e MYSQL_HOST=mysql \
  -e MYSQL_USER=root \
  -e MYSQL_PASSWORD=password \
  ilert-mcp-connector
```

## Kubernetes Deployment

Example Kubernetes deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ilert-mcp-connector
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ilert-mcp-connector
  template:
    metadata:
      labels:
        app: ilert-mcp-connector
    spec:
      containers:
      - name: ilert-mcp-connector
        image: ghcr.io/ilert/ilert-mcp-connector:latest
        ports:
        - containerPort: 8383
        env:
        - name: PORT
          value: "8383"
        - name: KAFKA_ENABLED
          value: "true"
        - name: KAFKA_BROKERS
          value: "kafka:9092"
        - name: MYSQL_ENABLED
          value: "true"
        - name: MYSQL_HOST
          value: "mysql"
        - name: MYSQL_USER
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: username
        - name: MYSQL_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mysql-secret
              key: password
        livenessProbe:
          httpGet:
            path: /health
            port: 8383
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8383
          initialDelaySeconds: 5
          periodSeconds: 5
          # Note: /ready endpoint requires authentication and contains sensitive connection info
          # Use /health for Kubernetes probes, or configure httpHeaders with Authorization token
---
apiVersion: v1
kind: Service
metadata:
  name: ilert-mcp-connector
spec:
  selector:
    app: ilert-mcp-connector
  ports:
  - port: 8383
    targetPort: 8383
```

## Versioning

The server version is set at build time using Go's `-ldflags` and is logged on startup. The version is synchronized with GitHub tags and Docker image tags.

### Building with Version

When building locally, you can set the version:

**Using Make:**
```bash
make build VERSION=1.0.0
```

**Manual build:**
```bash
go build -ldflags "-X main.Version=1.0.0 -X main.Commit=$(git rev-parse --short HEAD)" -o ilert-mcp-connector .
```

### Docker Build

The Dockerfile accepts build arguments for version information:
- `VERSION` - Version string (default: `dev`)
- `COMMIT` - Git commit hash (default: `unknown`)

## GitHub Actions

The repository includes a GitHub Actions workflow that automatically builds and pushes Docker images to GitHub Container Registry when:

- A tag starting with `v` is pushed (e.g., `v1.0.0`)
- The workflow is manually triggered with a custom tag

The workflow automatically:
- Extracts the version from the Git tag (removes `v` prefix if present)
- Sets the version and commit SHA as build arguments
- Tags Docker images with semantic version tags

Images are tagged with:
- Semantic version tags (e.g., `v1.0.0`, `v1.0`, `v1`)
- Branch names for branch pushes
- `latest` for the default branch

The version is logged on server startup and available via the `/version` endpoint.

## Development

### Testing

```bash
# Run all tests
make test

# Run tests without race detector (faster)
make test-short

# Run tests with coverage report
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run go vet
make vet

# Run all checks (fmt, vet, test)
make check

# Run full CI pipeline locally
make ci
```

### Development Workflow

```bash
# Download dependencies
make deps

# Format and check code
make fmt vet

# Run tests
make test

# Build and run locally
make start

# Or run without building first
make run
```
