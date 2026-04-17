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
- 🔔 **Pub/Sub Model**: Clients subscribe to track metrics of a specific user within a specific group
- 🛡️ **Group Isolation**: Permissions are re-evaluated on **every broadcast** against the database. Metrics are only sent if authorized by the specific group used during subscription.
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
3. Subscribes to track (specifying the Group ID): `{"action": "subscribe", "items": [{"target_user_id": "patient-123", "metric_type": "heart_rate", "group_id": "group-abc"}]}`
4. Receives real-time data every time the patient has a new heart rate reading, provided that Group ABC still authorizes the access.

### UC2: Dashboard monitoring multiple metrics of a person

**Actor**: Patient self-monitoring, Doctor  
**Flow**:

1. Connects to WebSocket
2. Subscribes to multiple metric types at once:

```json
{
  "action": "subscribe",
  "items": [
    {
      "target_user_id": "user-456",
      "metric_type": "heart_rate",
      "group_id": "uuid-of-group"
    },
    {
      "target_user_id": "user-456",
      "metric_type": "steps_count",
      "group_id": "uuid-of-group"
    },
    {
      "target_user_id": "user-456",
      "metric_type": "calories_burned",
      "group_id": "uuid-of-group"
    }
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
  "items": [{ "target_user_id": "user-456", "metric_type": "heart_rate" }]
}
```

---

## API Reference (Link Docs)

_Since the service uses the WebSocket protocol, API documentation is described directly below (no Swagger UI support like HTTP API at the moment)._

### WebSocket Endpoint

| Property           | Value                     |
| ------------------ | ------------------------- |
| **URL**            | `ws://<host>:5001`        |
| **Protocol**       | WebSocket                 |
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
      "metric_type": "heart_rate",
      "group_id": "uuid-of-group"
    }
  ]
}
```

> [!TIP]
> **Group Isolation**: If you omit `group_id`, the system will send the metric if you have permission in ANY group. If you provide `group_id`, it will ONLY send if the permission is granted within that specific group.

#### Unsubscribe Request

```json
{
  "action": "unsubscribe",
  "items": [
    {
      "target_user_id": "uuid-of-target-user",
      "metric_type": "heart_rate",
      "group_id": "uuid-of-group"
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

| Variable         | Description                      | Default            | Required |
| ---------------- | -------------------------------- | ------------------ | -------- |
| `JWT_SECRET`     | Secret key to validate JWT token | -                  | ✅       |
| `HTTP_PORT`      | Port for WebSocket server        | `5001`             | ✅       |
| `KAFKA_ADDR`     | Kafka broker address             | -                  | ✅       |
| `KAFKA_TOPIC`    | Topic containing health metrics  | `health_metrics`   | ✅       |
| `KAFKA_GROUP_ID` | Consumer group ID                | `realtime-service` | ✅       |
| `POSTGRES_URL`   | PostgreSQL connection string     | -                  | ✅       |

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
const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...';
const ws = new WebSocket(`ws://localhost:5001?token=${token}`);

ws.onopen = () => {
  console.log('Connected!');

  // Subscribe to heart rate of a user within a specific group
  ws.send(
    JSON.stringify({
      action: 'subscribe',
      items: [
        {
          target_user_id: 'target-uuid',
          metric_type: 'heart_rate',
          group_id: 'group-uuid',
        },
      ],
    }),
  );
};

ws.onmessage = (event) => {
  console.log('Received:', JSON.parse(event.data));
};

ws.onerror = (e) => console.error('Error:', e);
ws.onclose = (e) => console.log('Closed:', e.code);
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

## Permission & Privacy Model

The system uses a **"Group Rule as Base"** model to ensure granular yet safe data sharing.

### 1. The Logic

- **Rule 1 (Base - Global Rule)**: A metric type (e.g., `heart_rate`) must be explicitly enabled for the Group (shared with `null`) for any member to have a chance to see it.
- **Rule 2 (Filter - Specific Rule)**: If a specific rule exists for a member, they only see the intersection of the Base and their Filter. If no specific rules exist, they see all Base metrics.

### 2. Live Enforcement (Zero-Cache)

Unlike typical systems that check permissions once at connection time, **Realtime Service re-evaluates permissions on every single broadcast**.

1. Hub receives a metric from Kafka.
2. Hub queries the database for all authorized observers and their associated Group IDs.
3. Hub filters the currently connected subscribers:
   - If the subscriber provided a `group_id`, they only receive the data if that specific group authorizes them.
   - If a permission is revoked in the database, the stream stops for that user **instantly** on the next packet.
   - If a permission is granted, a user already "listening" will automatically start receiving data without needing to resubscribe.

### 3. Access Matrix (Example)

| Scenario    | Global Rule (Base) | Specific Rule for User A | Result for User A            | Result for others |
| ----------- | ------------------ | ------------------------ | ---------------------------- | ----------------- |
| Standard    | HR, SpO2           | (None)                   | Sees HR, SpO2                | Sees HR, SpO2     |
| Restricted  | HR, SpO2           | SpO2                     | **Sees ONLY SpO2**           | Sees HR, SpO2     |
| Blocked     | HR, SpO2           | (Empty)                  | **Blocked**                  | Sees HR, SpO2     |
| System Deny | SpO2               | HR                       | **Blocked** (HR not in Base) | Sees SpO2         |

### 4. Database Enforcement

All services (`realtime-service` and `storage-service`) use synchronized SQL logic to verify these permissions before streaming or returning data.

---

## Troubleshooting

| Issue                         | Cause                  | Solution                                                    |
| ----------------------------- | ---------------------- | ----------------------------------------------------------- |
| 401 Unauthorized              | Invalid/expired JWT    | Get a new access token from auth-service                    |
| "No permission for X"         | No permission to view  | Check if the target user has shared the metric in the group |
| Connection closes immediately | Token expired          | Refresh token and reconnect                                 |
| No metrics received           | No new data from Kafka | Check if Kafka producer is publishing data                  |
