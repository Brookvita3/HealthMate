---
skill: write_service_unit_tests
description: >
  Write unit tests for a service layer using testify/mock mocks, following
  the table-driven, env-struct pattern used throughout the project.
when_to_use:
  - Writing tests for a new or existing service method
  - Adding coverage for a previously untested business rule or error branch
  - Verifying authorization guard logic (owner checks, membership checks)
---

## Instructions

### 1. CREATE THE TEST FILE
- Path: `internal/<package>/<entity>_service_test.go`
- Use `package <package>_test` (black-box test package — external to the package under test).

### 2. DEFINE testEnv
Hold mock repositories and the live service under test in a single struct:
```go
type testEnv struct {
    XxxRepository *mocks.XxxRepository
    Service       xxx.Service
}
```

### 3. WRITE setupTest()
Wire fresh mocks into `NewService` every time — never reuse mocks across test cases:
```go
func setupTest() *testEnv {
    mockRepo := new(mocks.XxxRepository)
    return &testEnv{
        XxxRepository: mockRepo,
        Service:       xxx.NewService(mockRepo),
    }
}
```

### 4. STRUCTURE EACH TEST FUNCTION
One `TestXxx` per exported service method; use `t.Run` sub-tests for each branch:
```go
func TestCreateXxx(t *testing.T) {
    t.Run("success", func(t *testing.T) { ... })
    t.Run("fail: invalid name", func(t *testing.T) { ... })
    t.Run("fail: not owner", func(t *testing.T) { ... })
    t.Run("fail: repository error", func(t *testing.T) { ... })
}
```

### 5. SUB-TEST PATTERN
Follow this order inside every sub-test:
a. `env := setupTest()`
b. Set mock expectations: `.On(...).Return(...).Once()`
c. Call the service method with `context.Background()`
d. Assert results:
   - `assert.NoError(t, err)` / `assert.Error(t, err)`
   - `assert.Equal(t, expected, actual)`
   - `assert.Nil(t, result)` / `assert.NotNil(t, result)`
e. Always end with `env.XxxRepository.AssertExpectations(t)`

### 6. REQUIRED COVERAGE BRANCHES
Cover at minimum:
- ✅ Happy path — returns correct data, no error
- ✅ Input validation failure — no mock calls expected
- ✅ Authorization failure — requester is not owner / not member
- ✅ Repository returns not-found sentinel error
- ✅ Repository returns generic DB error

### 7. RUN TESTS
```
cd <service-dir>
go test ./internal/<package>/... -v -run TestXxx
```
