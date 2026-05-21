# Auth Service

Authentication and User Management service for HealthMate.

## Features
- User Authentication (Login, Register).
- Authentication via email and OTP.
- Authentication via Google OAuth.
- User and Group Management (Profile, Group, Role).
- Role-based Access Control (RBAC).

## Environment Variables
Required environment variables (can be set in a `.env` file):
| Variable | Description |
|---|---|
| `HTTP_PORT` | HTTP server port (e.g., `5000`) |
| `JWT_SECRET` | Secret key for JWT Token generation and validation |
| `REDIS_URL` | Redis connection string (used for OTP/caching) |
| `POSTGRES_URL` | PostgreSQL Database connection string |
| `GOOGLE_CLIENT_ID` | Client ID for Google OAuth |
| `SMTP_HOST`, `SMTP_PORT` | SMTP configuration for sending emails |
| `SMTP_USERNAME`, `SMTP_APP_PASSWORD` | Gmail account and App Password |
| `API_PREFIX` | API prefix (e.g., `api/v1`) |

## How to Build
From repo root (`deploy/docker-compose.prod.yml` or your environment):

```bash
docker compose -f deploy/docker-compose.prod.yml build auth-service
```

Or build the image only:
```bash
docker build -t brookvita3/auth-service:local ./auth-service
```
After deploying permission/sharing changes, rebuild and redeploy **auth-service**, **storage-service**, and **realtime-service** together.

## API Documentation (Link Docs)
Swagger UI (available after running the service):
- [http://localhost:5000/swagger/index.html](http://localhost:5000/swagger/index.html)

---

### Project Structure
```
auth-service
├─ app/           # App bootstrap
├─ cmd/server/    # Entry point
├─ config/        # Configuration
├─ docs/          # Swagger docs
├─ internal/      # Core logic (auth, cache, domain, mail, platform, user, web)
...
```

---

## Group Metric Sharing Permissions

The system supports sharing health metrics either with an entire group or with specific users within that group.

> [!IMPORTANT]
> **Permission Hierarchy (Strict Validation)**: A user can only share a metric with a **specific member** if that metric is already shared with the **entire group** globally.
> - If you try to share a metric individually that is not enabled at the group level, the API will return a `400 Bad Request` with a specific error message.
> - If the Group Admin (owner) revokes a metric from the Group Base, all specific member overrides for that metric are automatically revoked as well.

### 1. Share with EVERYONE in Group
To share a metric with all current and future members of a group:
- **Endpoint**: `POST /groups/:id/permissions`
- **Body**: Leave `target_user_id` as `null` or `""`.

```json
{
  "metric_type": "heart_rate",
  "enabled": true,
  "target_user_id": null
}
```

### 2. Share with a SPECIFIC USER in Group (bulk only)
Per-viewer rules **must** use `PUT`, not `POST`. The service writes an `access_control` marker plus metric rows so storage/realtime `CheckAccess` stays aligned with `GET .../members/visible-metrics`.

- **Endpoint**: `PUT /groups/:id/permissions`
- **Body**: `target_user_id` = viewer UUID; `metric_types` = subset of your global metrics.

```json
{
  "metric_types": ["heart_rate", "spo2"],
  "target_user_id": "987e6543-e21b-12d3-a456-426614174000"
}
```

`POST /groups/:id/permissions` with a non-null `target_user_id` returns **400** (`use PUT ... for per-member sharing`).

### 3. Bulk Update Global Permissions
- **Endpoint**: `PUT /groups/:id/permissions`
- **Body**: Omit `target_user_id` (or empty string).

```json
{
  "metric_types": ["heart_rate", "spo2", "blood_pressure"]
}
```

### 4. Reset Viewer to Global Defaults
Remove per-viewer filter so the viewer sees your full global share again:

```json
{
  "metric_types": [],
  "target_user_id": "987e6543-e21b-12d3-a456-426614174000"
}
```

### 5. Run unit tests (permission sharing)

From repo root (no local Go required):

```bash
docker run --rm -v "$(pwd)/auth-service:/app" -w /app golang:1.24-alpine \
  go test ./internal/permission/... -count=1 -v
```

Or with Go installed:

```bash
cd auth-service && go test ./internal/permission/... -count=1 -v
```

### 6. Fetching Current Permissions
- **Endpoint**: `GET /groups/:id/permissions`
- **Query Param**: `target_user_id` (optional). If provided, it filters for permissions granted to that specific user (including global group shares).
