---
skill: register_new_route
description: >
  Add a new HTTP endpoint to an existing handler, including middleware
  wiring, Swagger docs, and route registration.
when_to_use:
  - Adding an endpoint to an already-existing handler/package
  - Exposing a new service method via HTTP without creating a whole new module
  - Wiring a route that needs validation middleware (group ID, member ID, etc.)
---

## Instructions

### 1. ADD THE HANDLER METHOD
- Add the handler method to `internal/<package>/handler.go`.
- Follow the handler shape: extract auth ID → parse params → bind body → call service → respond.
- Add full Swagger annotations:
  ```go
  // @Summary <short summary>
  // @Description <longer description>
  // @Tags <tag>
  // @Accept json
  // @Produce json
  // @Param id path string true "Entity ID"
  // @Success 200 {object} webHelpers.OKResponse
  // @Failure 400,403,404 {object} webHelpers.ErrorResponse
  // @Security BearerAuth
  // @Router /path/{id} [method]
  ```

### 2. DEFINE DTOs NEXT TO THE HANDLER
- If a new request/response struct is needed, define it in `handler.go` next to the method.
- Keep DTOs co-located with their handler — do not place them in `domain/`.

### 3. REGISTER THE ROUTE IN app.go
- In `NewHTTPServer`, add the route to the appropriate group.
- Routes under `/:id` must be inside the subgroup that applies the group validation middleware.
- Routes with `:member_id` must also chain `groupMiddleware.ValidateMemberExists()`:
  ```go
  groupWithID.DELETE("/members/:member_id",
      groupMiddleware.ValidateMemberExists(),
      groupHandler.RemoveMember,
  )
  ```

### 4. ADD OR EXTEND THE SERVICE METHOD
- If the service method doesn't exist yet, add it to the `Service` interface and implement it in `serviceImpl`.
- Follow the authorization guard pattern from `implement_new_feature.md`.

### 5. REGENERATE SWAGGER DOCS
```
swag init
```
Run from the service root after adding new annotations.
