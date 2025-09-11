# Kafka → Consumer → MS SQL → Read API Pipeline

A minimal event pipeline implementation in Go with Kafka, MS SQL Server, Redis DLQ, and Docker Compose.

## Architecture

```
HTTP Producer → Kafka → Consumer → MS SQL Server
                     ↓
                   Redis DLQ
                     ↓
                 Read API ← HTTP Client
```

## Stack

- **Go** - Main application language
- **Kafka** - Message streaming platform
- **MS SQL Server** - Primary data store
- **Redis** - Dead Letter Queue (DLQ)
- **Docker Compose** - Container orchestration
- **Prometheus** - Metrics collection

## Event Types

The pipeline supports 4 event types:

1. **UserCreated** (key: userId)
2. **OrderPlaced** (key: orderId)
3. **PaymentSettled** (key: orderId)
4. **InventoryAdjusted** (key: sku)

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Git

### 1. Clone and Build

```bash
git clone https://github.com/doomslayer1436786/MotilalOswalAssignment.git
cd MotilalOswalAssignment
```

### 2. Start All Services

```bash
docker compose up --build
```

This will start:
- Zookeeper (port 2181)
- Kafka (port 9092)
- MS SQL Server (port 1433)
- Redis (port 6379)
- Producer service (port 8080)
- Consumer service (port 8081)
- Read API service (port 8082)

### 3. Wait for Services to be Ready

Wait for all services to be healthy (check with `docker compose ps`). The services have health checks configured.

### 4. Initialize Database Schema

**For Windows users:**

Run the provided initialization script:
```powershell
# PowerShell
.\init-db.ps1

# Or Command Prompt
init-db.bat
```

**For Linux/Mac users:**

```bash
# Connect to MS SQL container and run schema
docker exec -it mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1" -i /docker-entrypoint-initdb.d/schema.sql
```

**Manual method (all platforms):**

```bash
# Copy schema to container
docker cp sql/schema.sql mssql:/tmp/schema.sql

# Execute schema (try different paths based on MS SQL version)
docker exec -it mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1" -i /tmp/schema.sql

# Alternative path for older versions:
# docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1" -i /tmp/schema.sql
```

**Alternative: Connect directly to MS SQL container:**

```bash
# Connect to the container
docker exec -it mssql bash

# Inside the container, run:
/opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1"

# Then paste the contents of sql/schema.sql
```

### 5. Test the Pipeline

Use the provided sample requests:

```bash
# Using curl (examples from sample_curls.http)
curl -X POST http://localhost:8080/produce \
  -H "Content-Type: application/json" \
  -d '{
    "eventId": "550e8400-e29b-41d4-a716-446655440001",
    "type": "UserCreated",
    "timestamp": "2025-01-11T12:00:00Z",
    "data": {
      "userId": "user-123",
      "name": "Alice Example",
      "email": "alice@example.com",
      "createdAt": "2025-01-11T12:00:00Z"
    }
  }'
```

Or use the provided `sample_curls.http` file with your HTTP client (VS Code REST Client, IntelliJ HTTP Client, etc.).

## Environment Variables

### Producer Service
- `KAFKA_BROKERS` - Comma-separated Kafka broker addresses (default: localhost:9092)
- `KAFKA_TOPIC` - Kafka topic name (default: events)
- `SERVICE_PORT` - HTTP server port (default: 8080)
- `LOG_LEVEL` - Logging level (default: INFO)

### Consumer Service
- `KAFKA_BROKERS` - Comma-separated Kafka broker addresses (default: localhost:9092)
- `KAFKA_TOPIC` - Kafka topic name (default: events)
- `KAFKA_GROUP_ID` - Consumer group ID (default: consumer-group)
- `MSSQL_CONN` - MS SQL connection string
- `REDIS_ADDR` - Redis address (default: localhost:6379)
- `REDIS_PASSWORD` - Redis password (optional)
- `SERVICE_PORT` - Metrics server port (default: 8081)
- `LOG_LEVEL` - Logging level (default: INFO)

### API Service
- `MSSQL_CONN` - MS SQL connection string
- `SERVICE_PORT` - HTTP server port (default: 8082)
- `LOG_LEVEL` - Logging level (default: INFO)

## API Endpoints

### Producer Service (Port 8080)

- `POST /produce` - Publish event to Kafka
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

### Consumer Service (Port 8081)

- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

### Read API Service (Port 8082)

- `GET /users/{id}` - Get user with recent orders
- `GET /orders/{id}` - Get order with payment status
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

## Database Schema

The pipeline uses the following tables:

- `users` - User information
- `orders` - Order details
- `payments` - Payment information
- `inventory` - Inventory tracking

See `sql/schema.sql` for the complete schema.

## Dead Letter Queue (DLQ)

Failed messages are stored in Redis under the key `dlq:events`. To inspect DLQ messages:

```bash
# Connect to Redis container
docker exec -it redis redis-cli

# List DLQ messages (newest first)
LRANGE dlq:events 0 10
```

## Metrics

Prometheus metrics are exposed on `/metrics` endpoint for each service:

- `messages_processed_total{type="<eventType>"}` - Counter of processed messages
- `dlq_count_total` - Counter of messages sent to DLQ
- `db_latency_seconds` - Histogram of database operation latency
- `http_requests_total` - Counter of HTTP requests
- `http_latency_seconds` - Histogram of HTTP request latency

## Logging

Structured logging with correlation IDs:
- All event-related logs include `eventId` field
- Logs are in JSON format for easy parsing
- Log levels: DEBUG, INFO, WARN, ERROR

## Testing DLQ Functionality

The pipeline includes several test cases for DLQ:

1. **Malformed JSON** - Missing required fields
2. **Invalid Event Type** - Unknown event type
3. **Database Constraint Violation** - Foreign key violations

## Acceptance Checklist

### ✅ Happy Path Flow
- [ ] POST /produce for UserCreated event → message appears in Kafka
- [ ] Consumer processes UserCreated → user record created in MS SQL
- [ ] GET /users/{id} returns user data
- [ ] POST /produce for OrderPlaced event → message appears in Kafka
- [ ] Consumer processes OrderPlaced → order record created in MS SQL
- [ ] GET /orders/{id} returns order data
- [ ] POST /produce for PaymentSettled event → payment record created
- [ ] GET /orders/{id} returns order with payment status
- [ ] POST /produce for InventoryAdjusted event → inventory record created

### ✅ Idempotency
- [ ] Send same event twice with same eventId
- [ ] Verify no duplicate records in database
- [ ] Verify record is updated, not duplicated

### ✅ Dead Letter Queue
- [ ] Send malformed event (missing data field)
- [ ] Verify message appears in Redis DLQ (`dlq:events`)
- [ ] Verify DLQ message contains original payload + error details
- [ ] Send event with invalid event type
- [ ] Verify message appears in DLQ
- [ ] Send OrderPlaced with non-existent userId
- [ ] Verify DB constraint violation message appears in DLQ

### ✅ Metrics
- [ ] GET /metrics on producer shows `events_produced_total`
- [ ] GET /metrics on consumer shows `messages_processed_total`
- [ ] GET /metrics on consumer shows `dlq_count_total`
- [ ] GET /metrics on consumer shows `db_latency_seconds` histogram
- [ ] GET /metrics on API shows `http_requests_total`
- [ ] GET /metrics on API shows `http_latency_seconds` histogram

### ✅ Logging
- [ ] Log lines include `eventId` field for correlation
- [ ] Log lines include structured fields (type, key, topic, partition, offset)
- [ ] Error logs include error details
- [ ] Success logs include latency information

### ✅ Retry Behavior
- [ ] Consumer commits offset only after successful processing
- [ ] Consumer commits offset after pushing to DLQ
- [ ] No message loss during failures
- [ ] At-least-once delivery semantics maintained

## Troubleshooting

### Services Not Starting
```bash
# Check service status
docker compose ps

# Check logs
docker compose logs <service-name>

# Restart specific service
docker compose restart <service-name>
```

### Database Connection Issues
```bash
# Test MS SQL connection
docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1" -Q "SELECT 1"
```

### Kafka Issues
```bash
# Check Kafka logs
docker compose logs kafka

# List topics
docker exec -it kafka kafka-topics.sh --bootstrap-server localhost:9092 --list
```

### Redis Issues
```bash
# Test Redis connection
docker exec -it redis redis-cli ping

# Check DLQ
docker exec -it redis redis-cli LRANGE dlq:events 0 10
```

## Development

### Project Structure
```
/
├── cmd/
│   ├── producer/main.go    # HTTP producer service
│   ├── consumer/main.go    # Kafka consumer service
│   └── api/main.go         # Read API service
├── internal/
│   ├── kafka/              # Kafka client code
│   ├── store/              # Database models and operations
│   └── dlq/                # Redis DLQ implementation
├── sql/
│   └── schema.sql          # Database schema
├── docker-compose.yml      # Container orchestration
├── Dockerfile              # Multi-service build
├── sample_curls.http       # Test requests
└── README.md               # This file
```

### Building Locally
```bash
# Build all services
go build -o producer ./cmd/producer
go build -o consumer ./cmd/consumer
go build -o api ./cmd/api
```

### Running Tests
```bash
# Run sample requests
# Use the provided sample_curls.http file
```

## License

This project is provided as-is for demonstration purposes.
