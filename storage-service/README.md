# Storage Service

Service for storing and retrieving Health Metrics data for the HealthMate system.

## Features
- Store health metrics into TimescaleDB, specialized for time-series data.
- Kafka Consumer: automatically pull data from the `health_metrics` topic on Kafka.
- Provide APIs to retrieve raw metrics or aggregated data over time (time buckets).
- Manage Metric Types (store metadata, cache with Redis to improve performance).

## Environment Variables
Required environment variables (can be set in a `.env` file):
| Variable | Description |
|---|---|
| `PORT` | HTTP API port (e.g., `5003`) |
| `TIMESCALEDB_URL` | TimescaleDB (PostgreSQL) connection string |
| `REDIS_URL` | Redis connection string (for metadata caching) |
| `KAFKA_ADDR` | Kafka broker address (e.g., `localhost:9092`) |
| `KAFKA_TOPIC` | Topic for receiving health metrics (e.g., `health_metrics`) |
| `KAFKA_GROUP_ID` | Consumer group ID |
| `JWT_SECRET` | Secret key used to verify Authorization token |
| `API_PREFIX` | API prefix (e.g., `api/v1`) |

## How to Build
Run the following command at the root directory of the project (where `docker-compose.yml` is located):
```bash
docker compose build storage-service
```
Or build independently at the project root:
```bash
docker build -t brookvita3/storage-service:local ./storage-service
```

## API Documentation (Link Docs)
Access the Swagger Docs for storage-service at the following URL after running the service:
- [http://localhost:5003/swagger/index.html](http://localhost:5003/swagger/index.html)

## Permission & Privacy Model

The system uses a **"Group Rule as Base"** model to ensure granular yet safe data sharing.

### 1. The Logic
- **Rule 1 (Base - Global Rule)**: A metric type (e.g., `heart_rate`) must be explicitly enabled for the Group (shared with `null`) for any member to have a chance to see it.
- **Rule 2 (Filter - Specific Rule)**: If a specific rule exists for a member, they only see the intersection of the Base and their Filter. If no specific rules exist, they see all Base metrics.

### 2. Implementation in Storage Service
- **Notifications**: When a health metric exceeds a configured threshold, the system calculates the authorized "watchers" using a strict SQL join across `group_members` and `sharing_permissions`. Only members with valid "Base + Filter" permissions receive the notification.
- **Charts/History**: All data retrieval APIs verify current sharing permissions in real-time against the database. Unauthorized requests return `403 Forbidden`.
- **Consistency**: The same permission logic is shared across `realtime-service` and `storage-service` to ensure privacy is maintained regardless of the data delivery channel.
