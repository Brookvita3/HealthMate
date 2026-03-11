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
