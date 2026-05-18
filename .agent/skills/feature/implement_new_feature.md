---
skill: implement_new_feature
description: >
  Add a new business feature end-to-end following the project's clean
  architecture: domain model → repository interface + params structs →
  postgres implementation → service interface + serviceImpl → Gin handler
  → route registration in app.go.
when_to_use:
  - Adding a brand-new entity or resource to an existing service
  - Implementing a new business capability that requires persistence
  - Extending the API surface with new CRUD or workflow operations
---

## Instructions

### 1. DOMAIN MODEL
- Add or extend a struct in `internal/domain/<entity>.go`.
- Keep domain structs pure: no db tags, no JSON tags (those belong to the handler or params).

### 2. ERRORS
- Create or update `internal/<package>/errors.go`.
- Use `&common.BusinessError{Code: <HTTP status>, Message: "<message>"}` for every domain-level error.
- Export each error as a named var: `var ErrXxx = &common.BusinessError{...}`.

### 3. REPOSITORY INTERFACE + PARAMS
- Define the interface in `internal/<package>/<entity>_repository.go`.
- Put Params structs (e.g. `CreateXxxParams`, `UpdateXxxParams`) in the same file.
- Use pointer fields (`*string`, `*uuid.UUID`) for optional/partial-update fields.
- Always include `ctx context.Context` as the first parameter.

### 4. POSTGRES IMPLEMENTATION
- Create `internal/<package>/postgres_<entity>_repository.go`.
- Use `pgxpool.Pool` via the existing `postgrePlatform.NewPostgreSQLConnFromURL`.
- Map `pgx.ErrNoRows` → the package-local `ErrXxxNotFound` sentinel.
- Do not leak raw DB errors; wrap only when you need to add context.

### 5. SERVICE INTERFACE + IMPL
- Define `Service` interface and `serviceImpl` struct in `internal/<package>/<entity>_service.go`.
- Constructor: `func NewService(repo XxxRepository, ...) Service`.
- Business rules (auth checks, validation, cross-repo consistency) live here, never in the handler.
- Authorization guard pattern:
  ```go
  isOwner, err := s.IsXxxOwner(ctx, id, requesterID)
  if err != nil { return err }
  if !isOwner { return ErrNotXxxOwner }
  ```

### 6. GIN HANDLER
- Create `internal/<package>/handler.go` with a `Handler` struct holding service dependencies.
- Constructor: `func NewHandler(svc Service, ...) *Handler`.
- Every handler method follows this exact shape:
  a. Extract auth user ID: `id, ok := webHelpers.GetAuthUserID(c); if !ok { return }`
  b. Parse path params: `entityID, ok := webHelpers.GetValidatedXxxID(c); if !ok { return }`
  c. Bind JSON body: `if err := c.ShouldBindJSON(&req); err != nil { handler.handleError(c, common.ErrInvalidRequest); return }`
  d. Call service.
  e. On error: `handler.handleError(c, err); return`
  f. On success: `c.JSON(http.StatusOK, webHelpers.OKResponse{Message: "..."})` or `c.JSON(http.StatusCreated, entity)` for creation.
- Add a private `handleError` delegation:
  ```go
  func (h *Handler) handleError(c *gin.Context, err error) {
      webHelpers.HandleError(c, err)
  }
  ```
- Add Swagger annotations above every exported method (`@Summary`, `@Tags`, `@Success`, `@Failure`, `@Security BearerAuth`, `@Router`).

### 7. ROUTE REGISTRATION (app.go)
- In `NewHTTPServer`, instantiate the handler and wire its routes.
- Routes that use `:id` path params must be grouped under a subgroup that applies the validation middleware (e.g. `groupMiddleware.ValidateXxxExists()`).
- Inject the handler's dependencies via the `Dependencies` struct:
  ```go
  deps.XxxRepo = xxx.NewPostgresRepository(db)
  deps.XxxService = xxx.NewService(deps.XxxRepo, ...)
  ```

### 8. MOCK GENERATION
- After defining new interfaces run:
  ```
  mockery --name=XxxRepository --dir=internal/<package> --output=mocks --outpkg=mocks
  ```
- Confirm the generated mock appears in `mocks/` before writing tests.
