```
auth-service
├─ app
│  ├─ app.go
│  └─ router.go
├─ cmd
│  └─ server
│     └─ main.go
├─ config
│  └─ config.go
├─ docs
│  ├─ docs.go
│  ├─ swagger.json
│  └─ swagger.yaml
├─ go.mod
├─ go.sum
├─ internal
│  ├─ auth
│  │  ├─ auth_service.go
│  │  ├─ auth_service_test.go
│  │  ├─ error.go
│  │  ├─ google_verifier.go
│  │  ├─ handler.go
│  │  ├─ jwt_token_service.go
│  │  ├─ otp_service.go
│  │  ├─ redis_otp_service.go
│  │  └─ token_service.go
│  ├─ cache
│  │  └─ cache.go
│  ├─ common
│  │  ├─ context_keys.go
│  │  └─ errors.go
│  ├─ domain
│  │  ├─ guser.go
│  │  ├─ otp.go
│  │  └─ user.go
│  ├─ mail
│  │  ├─ email_service.go
│  │  └─ gmail_service.go
│  ├─ platform
│  │  ├─ postgres
│  │  │  └─ client.go
│  │  └─ redis
│  │     ├─ client.go
│  │     └─ redis_cache.go
│  ├─ user
│  │  ├─ errors.go
│  │  ├─ handler.go
│  │  ├─ postgres_user_repository.go
│  │  ├─ user_repository.go
│  │  └─ user_service.go
│  └─ web
│     └─ middleware
│        ├─ error.go
│        ├─ prometheus.go
│        └─ rbac.go
├─ mocks
│  ├─ Cache.go
│  ├─ EmailService.go
│  ├─ GoogleTokenVerifier.go
│  ├─ OTPService.go
│  ├─ scannable.go
│  ├─ Service.go
│  ├─ TokenService.go
│  └─ UserRepository.go
└─ proto
   └─ auth.proto

```
