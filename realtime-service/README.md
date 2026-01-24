# Realtime Service  
> **Last Updated:** 2026-01-24

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Use Cases](#use-cases)
4. [API Reference](#api-reference)
5. [Message Formats](#message-formats)
6. [Configuration](#configuration)
7. [Quick Start](#quick-start)

---

## Overview

**Realtime Service** là một WebSocket server phục vụ việc truyền tải dữ liệu sức khỏe (health metrics) theo thời gian thực từ hệ thống đến các client đã đăng ký theo dõi.

### Key Features
- 🔐 **JWT Authentication**: Xác thực token JWT cục bộ (không cần gọi auth-service)
- 📡 **WebSocket**: Kết nối real-time hai chiều với client
- 🔔 **Pub/Sub Model**: Client subscribe theo dõi metrics của user cụ thể
- 🛡️ **Permission-based**: Kiểm tra quyền truy cập dựa trên group membership
- ⚡ **Kafka Integration**: Nhận health metrics từ Kafka topic

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

### UC1: Theo dõi nhịp tim của thành viên trong nhóm
**Actor**: Bác sĩ, Người thân  
**Flow**:
1. User đăng nhập và lấy access token từ auth-service
2. Kết nối WebSocket với token: `ws://host:5001?token=<JWT>`
3. Subscribe theo dõi: `{"action": "subscribe", "items": [{"target_user_id": "patient-123", "metric_type": "heart_rate"}]}`
4. Nhận dữ liệu real-time mỗi khi bệnh nhân có nhịp tim mới

### UC2: Dashboard theo dõi nhiều chỉ số của một người
**Actor**: Bệnh nhân tự theo dõi, Bác sĩ  
**Flow**:
1. Kết nối WebSocket
2. Subscribe nhiều metric types cùng lúc:
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
3. Nhận tất cả metrics của user đó theo thời gian thực

### UC3: Hủy theo dõi khi không cần thiết
**Actor**: Bất kỳ client nào  
**Flow**:
1. Gửi message unsubscribe:
```json
{
  "action": "unsubscribe",
  "items": [{"target_user_id": "user-456", "metric_type": "heart_rate"}]
}
```

---

## API Reference

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
- `heart_rate` - Nhịp tim (bpm)
- `steps_count` - Số bước chân
- `calories_burned` - Calories tiêu thụ

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
  "type": "heart_rate",
  "value": 75.5,
  "timestamp": "2026-01-24T14:30:00Z"
}
```

---

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JWT_SECRET` | Secret key để validate JWT token | - | ✅ |
| `HTTP_PORT` | Port cho WebSocket server | `5001` | ✅ |
| `KAFKA_ADDR` | Kafka broker address | - | ✅ |
| `KAFKA_TOPIC` | Topic chứa health metrics | `health_metrics` | ✅ |
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

Để subscribe theo dõi metrics của một user, client phải có quyền truy cập dựa trên:

1. **Cùng nhóm (Group Membership)**: Cả observer và target phải cùng thuộc một group với status `accepted`
2. **Sharing Permission**: Target user phải bật chia sẻ metric type đó trong group

**Database Schema (Simplified):**
```sql
-- group_members: Thành viên của các nhóm
-- sharing_permissions: Các metric types mà user cho phép chia sẻ trong group
```

---

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| 401 Unauthorized | Invalid/expired JWT | Lấy access token mới từ auth-service |
| "No permission for X" | Không có quyền xem | Kiểm tra target user đã share metric trong group chưa |
| Connection closes immediately | Token expired | Refresh token và reconnect |
| No metrics received | Không có data mới từ Kafka | Kiểm tra Kafka producer đang publish data |
