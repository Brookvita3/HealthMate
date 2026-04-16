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
Run the following command at the root directory of the project (where `docker-compose.yml` is located):
```bash
docker compose build auth-service
```
Or build independently at the project root:
```bash
docker build -t brookvita3/auth-service:local ./auth-service
```

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
> **Permission Hierarchy**: A user can only share a metric with a **specific member** if that metric is already shared with the **entire group** globally. If you try to share a metric individually that is not enabled at the group level, the API will return a `403 Forbidden` error.

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

### 2. Share with a SPECIFIC USER in Group
To share a metric with only one specific member:
- **Endpoint**: `POST /groups/:id/permissions`
- **Body**: Provide the UUID of the target member in `target_user_id`.

```json
{
  "metric_type": "spo2",
  "enabled": true,
  "target_user_id": "987e6543-e21b-12d3-a456-426614174000"
}
```

### 3. Bulk Update Permissions
To set all shared metrics for a target at once:
- **Endpoint**: `PUT /groups/:id/permissions`
- **Body**: `metric_types` should be an array of all metrics allowed for that target.

```json
{
  "metric_types": ["heart_rate", "spo2", "blood_pressure"],
  "target_user_id": "" 
}
```

### 4. Fetching Current Permissions
- **Endpoint**: `GET /groups/:id/permissions`
- **Query Param**: `target_user_id` (optional). If provided, it filters for permissions granted to that specific user (including global group shares).
