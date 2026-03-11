# Realtime Service  
> **Last Updated:** 2026-01-24

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Use Cases](#use-cases)
4. [API Reference (Link Docs)](#api-reference-link-docs)
5. [Message Formats](#message-formats)
6. [Configuration](#configuration)
7. [Quick Start](#quick-start)

---

## Overview

**Realtime Service** is a WebSocket server dedicated to transmitting real-time health data (health metrics) from the system to subscribed clients.

### Key Features
- 🔐 **JWT Authentication**: Local JWT token verification (no need to call auth-service)
- 📡 **WebSocket**: Two-way real-time connection with clients
- 🔔 **Pub/Sub Model**: Clients subscribe to track metrics of a specific user
- 🛡️ **Permission-based**: Access control based on group membership
- ⚡ **Kafka Integration**: Receives health metrics from Kafka topics

---

## Architecture

```mermaid
flowchart LR
    subgraph External
        IoT["IoT Devices/Apps"]
        Client["Web/Mobile Clients"]
    end
    
    subgraph HealthMate System
        Kafka[("Kafka\nhealth_metrics")]
        RS["Realtime Service"]
        DB[("PostgreSQL\nPermissions")]
    end
    
    IoT -->|Publish metrics| Kafka
    Kafka -->|Consume| RS
    Client <-->|WebSocket| RS
    RS -->|Check permissions| DB
```

### Component Diagram

```mermaid
graph TB
    subgraph Realtime Service
        Handler["Handler\n(JWT Validation)"]
        Hub["Hub\n(Connection Manager)"]
        Client["Client\n(WebSocket Session)"]
        Consumer["Kafka Consumer"]
        PermRepo["Permission Repository"]
    end
    
    Handler -->|Create| Client
    Client -->|Register| Hub
    Consumer -->|Broadcast| Hub
    Hub -->|Check| PermRepo
    Hub -->|Send| Client
```

---

## Use Cases

### UC1: Monitor a group member's heart rate
**Actor**: Doctor, Family member  
**Flow**:
1. User logs in and gets an access token from auth-service
2. Connects to WebSocket with token: `ws://host:5001?token=<JWT>`
3. Subscribes to track: `{"action": "subscribe", "items": [{"target_user_id": "patient-123", "metric_type": "heart_rate"}]}`
4. Receives real-time data every time the patient has a new heart rate reading

### UC2: Dashboard monitoring multiple metrics of a person
**Actor**: Patient self-monitoring, Doctor  
**Flow**:
1. Connects to WebSocket
2. Subscribes to multiple metric types at once:
```json
{
  "action": "subscribe",
  "items": [
    {"target_user_id": "user-456", "metric_type": "heart_rate"},
    {"target_user_id": "user-456", "metric_type": "steps_count"},
    {"target_user_id": "user-456", "metric_type": "calories_burned"}
  ]
}
```
3. Receives all metrics of that user in real time

### UC3: Unsubscribe when no longer needed
**Actor**: Any client  
**Flow**:
1. Sends unsubscribe message:
```json
{
  "action": "unsubscribe",
  "items": [{"target_user_id": "user-456", "metric_type": "heart_rate"}]
}
```

---

## API Reference (Link Docs)

*Since the service uses the WebSocket protocol, API documentation is described directly below (no Swagger UI support like HTTP API at the moment).*

### WebSocket Endpoint

| Property | Value |
|----------|-------|
| **URL** | `ws://<host>:5001` |
| **Protocol** | WebSocket |
| **Authentication** | JWT token via query param |

#### Connection

```
ws://localhost:5001?token=<JWT_ACCESS_TOKEN>
```

**Responses:**
| Status | Description |
|--------|-------------|
| 101 Switching Protocols | Connection successful |
| 401 Unauthorized | Missing/invalid token |

---

## Message Formats

### Client → Server

#### Subscribe Request
```json
{
  "action": "subscribe",
  "items": [
    {
      "target_user_id": "uuid-of-target-user",
      "metric_type": "heart_rate"
    }
  ]
}
```

#### Unsubscribe Request
```json
{
  "action": "unsubscribe",
  "items": [
    {
      "target_user_id": "uuid-of-target-user",
      "metric_type": "heart_rate"
    }
  ]
}
```

**Supported `metric_type` values:**
- `heart_rate` - Heart rate (bpm)
- `steps_count` - Steps count
- `calories_burned` - Calories burned

### Server → Client

#### Success Response
```json
{
  "type": "success",
  "payload": "Subscribe success"
}
```

#### Error Response
```json
{
  "type": "error",
  "payload": "No permission for user-123/heart_rate"
}
```

#### Health Metric Data
```json
{
  "user_id": "patient-123",
  "metric_type": "heart_rate",
  "value": 75.5,
  "timestamp": "2026-01-24T14:30:00Z"
}
```

---

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JWT_SECRET` | Secret key to validate JWT token | - | ✅ |
| `HTTP_PORT` | Port for WebSocket server | `5001` | ✅ |
| `KAFKA_ADDR` | Kafka broker address | - | ✅ |
| `KAFKA_TOPIC` | Topic containing health metrics | `health_metrics` | ✅ |
| `KAFKA_GROUP_ID` | Consumer group ID | `realtime-service` | ✅ |
| `POSTGRES_URL` | PostgreSQL connection string | - | ✅ |

### Example `.env`
```env
JWT_SECRET=a2585871c1d7511ab671226640fd9eebdd6c0df8ee981a864b38138846dc0bb1
HTTP_PORT=5001
KAFKA_ADDR=localhost:9092
KAFKA_TOPIC=health_metrics
KAFKA_GROUP_ID=realtime-service
POSTGRES_URL=postgres://postgres:postgres@localhost:5432/healthmate?sslmode=disable
```

---

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL running with schema
- Kafka broker running

### How to Build
Run the following command at the root directory of the project (where `docker-compose.yml` is located):
```bash
docker compose build realtime-service
```
Or build Docker image independently:
```bash
docker build -t brookvita3/realtime-service:local ./realtime-service
```

### Run Locally

```bash
cd realtime-service
go run ./cmd/server/main.go
```

### Test Connection

**Using JavaScript (Browser Console):**
```javascript
// Replace with valid JWT token
const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...";
const ws = new WebSocket(`ws://localhost:5001?token=${token}`);

ws.onopen = () => {
  console.log("Connected!");
  
  // Subscribe to heart rate of a user
  ws.send(JSON.stringify({
    action: "subscribe",
    items: [
      { target_user_id: "target-uuid", metric_type: "heart_rate" }
    ]
  }));
};

ws.onmessage = (event) => {
  console.log("Received:", JSON.parse(event.data));
};

ws.onerror = (e) => console.error("Error:", e);
ws.onclose = (e) => console.log("Closed:", e.code);
```

**Using wscat:**
```bash
# Install wscat
npm install -g wscat

# Connect
wscat -c "ws://localhost:5001?token=<JWT_TOKEN>"

# Then send subscribe message
{"action":"subscribe","items":[{"target_user_id":"uuid","metric_type":"heart_rate"}]}
```

---

## Project Structure

```
realtime-service/
├── app/
│   └── app.go              # Application bootstrap & dependency injection
├── cmd/server/
│   └── main.go             # Entry point
├── config/
│   └── config.go           # Environment configuration
├── internal/
│   ├── auth/
│   │   └── jwt_validator.go    # Local JWT validation
│   ├── kafka/
│   │   └── consumer.go         # Kafka consumer for health metrics
│   ├── metric/
│   │   └── metric.go           # HealthMetric data structure
│   ├── permission/
│   │   ├── permission_repository.go       # Interface
│   │   └── postgres_permission_repository.go  # PostgreSQL implementation
│   └── realtime/
│       ├── handler.go          # HTTP handler (JWT auth + WebSocket upgrade)
│       ├── hub.go              # WebSocket connection hub (pub/sub)
│       └── client.go           # WebSocket client session
└── .env
```

---

## Permission Model

To subscribe and monitor a user's metrics, the client must have access privileges based on:

1. **Group Membership**: Both observer and target must belong to the same group with `accepted` status
2. **Sharing Permission**: Target user must enable sharing for that metric type in the group

**Database Schema (Simplified):**
```sql
-- group_members: Group members
-- sharing_permissions: Metric types that the user allows sharing in a group
```

---

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| 401 Unauthorized | Invalid/expired JWT | Get a new access token from auth-service |
| "No permission for X" | No permission to view | Check if the target user has shared the metric in the group |
| Connection closes immediately | Token expired | Refresh token and reconnect |
| No metrics received | No new data from Kafka | Check if Kafka producer is publishing data |
