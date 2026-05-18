---
skill: add_domain_error
description: >
  Add a new typed domain error for a package, following the project's
  BusinessError pattern so it is automatically rendered by HandleError.
when_to_use:
  - A new business rule violation needs its own HTTP status and message
  - An existing error message is too generic and needs to be split
  - A new package is being created and needs its own error set
---

## Instructions

### 1. OPEN THE ERRORS FILE
Open (or create) `internal/<package>/errors.go`.

### 2. IMPORT common
```go
import "auth-service/internal/common"
```

### 3. ADD THE SENTINEL
Inside the existing `var (...)` block:
```go
var (
    ErrXxxNotFound    = &common.BusinessError{Code: 404, Message: "xxx not found"}
    ErrXxxForbidden   = &common.BusinessError{Code: 403, Message: "access denied"}
    // ...
)
```

#### HTTP Status Code Guide
| Code | Semantic                        | Use when…                               |
|------|---------------------------------|-----------------------------------------|
| 400  | Bad Request                     | Validation failure, malformed input     |
| 401  | Unauthorized                    | Missing or invalid auth token           |
| 403  | Forbidden                       | Authenticated but not authorized        |
| 404  | Not Found                       | Resource does not exist                 |
| 409  | Conflict                        | Uniqueness violation, duplicate         |

### 4. USE IN SERVICE, NOT IN HANDLER
Return the sentinel from the service layer.
`webHelpers.HandleError` in the handler will convert it to the correct HTTP response automatically.

**Never** call `c.JSON` with a raw error directly in a handler.
