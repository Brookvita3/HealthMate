---
skill: debug_request_flow
description: >
  Trace a failing HTTP request end-to-end through the HealthMate service
  layers to identify where it breaks.
when_to_use:
  - A handler returns an unexpected status code or error message
  - A request succeeds at one layer but fails silently at another
  - Debugging a 500 that should have been a 403 or 404
  - Tracking down a missing response payload
---

## Instructions

### 1. IDENTIFY THE ENTRY POINT
- Find the route in `app/app.go` (`NewHTTPServer`).
- Note which middleware runs before the handler:
  - Auth middleware (`authHandler.AuthMiddleware()`)
  - Group validation middleware (`groupMiddleware.ValidateGroupExists()`)
  - Member validation middleware (`groupMiddleware.ValidateMemberExists()`)

### 2. HANDLER LAYER
- Confirm `webHelpers.GetAuthUserID(c)` succeeds and returns the expected user ID.
- Confirm path param helpers return `ok = true`:
  - `GetValidatedGroupID(c)`
  - `GetValidatedMemberID(c)`
  - `GetGroupID(c)` (fallback without middleware)
- Confirm `c.ShouldBindJSON(&req)` parses the body without error.

### 3. SERVICE LAYER
- Check each guard in order: input validation → ownership check → existence check → business rule.
- Look for early-return errors that might swallow the real failure cause.
- Confirm the correct method is being called with the right arguments.

### 4. REPOSITORY LAYER
- Confirm SQL query parameters match expected values. Add temporary logging if needed:
  ```go
  fmt.Printf("query params: %v\n", params)
  ```
- Verify `pgx.ErrNoRows` is being mapped to the package-local not-found sentinel,
  not propagated as a raw error.

### 5. ERROR RENDERING
- Confirm the error bubbling up to the handler is a `*common.BusinessError`.
- If it is not, `webHelpers.HandleError` returns a generic 500. Wrap it:
  ```go
  return &common.BusinessError{Code: 422, Message: "meaningful message"}
  ```

### 6. REPRODUCE WITH CURL
Craft a minimal request that isolates the failing scenario. Example:
```bash
# Obtain a token first
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/app \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}' | jq -r '.token')

# Then call the failing endpoint
curl -s -X GET http://localhost:8080/api/groups/<id> \
  -H "Authorization: Bearer $TOKEN" | jq
```
Replace the method, path, and body to match the failing request. Use `-v` for full request/response headers.
