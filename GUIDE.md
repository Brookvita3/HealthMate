# HealthMate Project - Developer Guide

HealthMate is a real-time health monitoring system based on a microservices architecture. This guide provides instructions for environment setup, service configuration, and API documentation.

## 1. Prerequisites
- **Docker & Docker Compose** (Highly recommended for local development)
- **Go 1.22+** (For local service execution or debugging)
- **Postman/Insomnia** (For API testing)

## 2. Environment Configuration (.env files)

Each service requires specific environment variables. Below are the formats for each service's `.env` file.

### 2.1. API Gateway (`api-gateway/.env`)
```env
JWT_SECRET="your_jwt_secret_here"
AUTH_HTTP_URL="http://DACN-auth-service:5000"
REALTIME_HTTP_URL="http://DACN-realtime-service:5001"
PORT=5002
KAFKA_BROKER_URL="DACN-kafka:9094"
KAFKA_TOPIC="health_metrics"
API_PREFIX="/api/v1"
```

### 2.2. Auth Service (`auth-service/.env`)
```env
HTTP_PORT=5000
JWT_SECRET="your_jwt_secret_here"
REDIS_URL="redis://DACN1-redis:6379"
POSTGRES_URL="postgres://postgres:postgres@DACN1-timescaledb:5432/healthmate?sslmode=disable"
GOOGLE_CLIENT_ID="your_google_client_id"
SMTP_HOST="smtp.gmail.com"
SMTP_PORT=587
SMTP_USERNAME="your_email@gmail.com"
SMTP_APP_PASSWORD="your_app_password"
SMTP_SENDER_NAME="HealthMate Support"
API_PREFIX="api/v1"
```

### 2.3. Realtime Service (`realtime-service/.env`)
```env
JWT_SECRET="your_jwt_secret_here"
KAFKA_ADDR="DACN-kafka:9094"
HTTP_PORT=5001
POSTGRES_URL="postgres://postgres:postgres@DACN1-timescaledb:5432/healthmate?sslmode=disable"
KAFKA_TOPIC="health_metrics"
KAFKA_GROUP_ID="realtime-service"
```

### 2.4. Storage Service (`storage-service/.env`)
```env
KAFKA_ADDR="DACN-kafka:9094"
PORT=5003
TIMESCALEDB_URL="postgres://postgres:postgres@DACN1-timescaledb:5432/healthmate?sslmode=disable"
KAFKA_TOPIC="health_metrics"
KAFKA_GROUP_ID="storage-service"
```

## 3. Running with Docker

To start the entire infrastructure (Databases, Kafka, Redis) and all microservices:

```bash
# Start all services
docker-compose up -d

# Rebuild and restart a specific service after code changes
docker-compose up -d --build <service-name>
```

### Important Service Ports:
- **Nginx (Main Entry):** `http://localhost:8080`
- **Kafka UI:** `http://localhost:8081` (Monitor Kafka topics and messages)
- **Auth Swagger:** `http://localhost:8080/auth/swagger/index.html`
- **PostgreSQL/TimescaleDB:** `localhost:5432`
- **Redis:** `localhost:6379`

## 4. API Documentation

Requests should be sent through the **Nginx Gateway** (Port 8080).

### 4.1. API Documentation (Swagger)
Each service provides its own Swagger documentation. When running through the Gateway, you can access them as follows:

- **Auth & Group Service:**
  - Local: `http://localhost:5000/swagger/index.html`
- **Realtime Service:**
  - Local: `http://localhost:5001/swagger/index.html`
  - Real-time WebSocket: `ws://localhost:5001/ws?token=<JWT>`

The Swagger documentation replaces the manual API tables and remains always up-to-date with the code.

### 4.3. Real-time Data Flow
The system uses a WebSocket-based ingestion and broadcast mechanism:

1. **Connection**: Connect to the WebSocket endpoint:
   - `ws://localhost:8080/ws?token=<JWT_TOKEN>`

2. **Metric Subscription (Follower Role)**: To receive data from another user, send a subscription message:
   ```json
   {
     "action": "subscribe",
     "items": [
       {
         "target_user_id": "USER_ID",
         "metric_type": "heart_rate"
       }
     ]
   }
   ```
   *Note: Permissions will be verified against the `sharing_permissions` table.*

3. **Data Ingestion (Source Role)**: Send health metrics as JSON directly through the same connection:
   - Payload: `{"user_id": "YOUR_ID", "metric_type": "heart_rate", "value": 72}`
   - *Note: `user_id` must match your JWT identity.*

4. **Internal Routing & Broadcast**:
   - `realtime-service` publishes received metrics to **Kafka** (topic: `health_metrics`) and broadcasts the data to all authorized followers who have performed the **Subscription** step above.
   - `storage-service` consumes from Kafka to persist data.

## 5. Directory Structure
- `/api-gateway`: Request routing and Kafka production.
- `/auth-service`: Core business logic (Auth, Groups, Mail, User Management).
- `/realtime-service`: WebSocket management and real-time broadcasting.
- `/storage-service`: Worker service for persisting Kafka data to TimescaleDB.
- `/nginx`: Routing and load balancing configuration.

## 6. Maintenance & Cleanup Commands

```bash
# Stop all services (keeps volumes/data intact)
docker-compose down

# Stop all services and remove orphan containers
docker-compose down --remove-orphans

# Cleanup unused Docker images to save disk space
docker image prune -f

# View real-time logs for a specific service
docker logs -f <container-name>
```
