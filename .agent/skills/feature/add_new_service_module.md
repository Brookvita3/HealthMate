---
skill: add_new_service_module
description: >
  Bootstrap a complete new internal service module (e.g. `notification`,
  `audit`) following the established package layout.
when_to_use:
  - Creating a brand-new domain concern that doesn't fit into any existing package
  - Scaffolding a new module from scratch with its full layer stack
  - Adding a new cross-cutting capability (e.g. notifications, audit logging)
---

## Instructions

### 1. CREATE THE DIRECTORY
- Create `internal/<module>/`.

### 2. CREATE FILES IN ORDER
a. `errors.go` — package-local `BusinessError` sentinels.
b. `<entity>.go` — repository interface + Params structs.
c. `postgres_<entity>_repository.go` — pgx implementation.
d. `<entity>_service.go` — `Service` interface + `serviceImpl`.
e. `handler.go` — Gin `Handler` struct + methods.

Follow the conventions in `implement_new_feature.md` for each layer.

### 3. WIRE INTO app/app.go
a. Add repo and service fields to the `Dependencies` struct.
b. Instantiate them in `NewDependencies`.
c. Register HTTP routes in `NewHTTPServer`.

### 4. GENERATE MOCKS
```
mockery --name=<Entity>Repository --dir=internal/<module> --output=mocks --outpkg=mocks
mockery --name=Service --dir=internal/<module> --output=mocks --outpkg=mocks --filename=<Module>Service.go
```

### 5. WRITE UNIT TESTS
Follow the `write_service_unit_tests` skill.

### 6. ADD MIGRATIONS (if needed)
Follow the `add_database_migration` skill if the module requires new tables.
