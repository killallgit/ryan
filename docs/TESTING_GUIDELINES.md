# Testing Guidelines

## Test Organization

### File Structure
- Unit tests: `pkg/*/[component]_test.go`
- Integration tests: `integration/[feature]_test.go`
- Mock objects: `pkg/testutil/mocks/[component].go`
- Test fixtures: `pkg/testutil/fixtures/[data].go`

### Naming Conventions
- Test functions: `TestComponentFunction`
- Benchmark functions: `BenchmarkComponentFunction`
- Example functions: `ExampleComponentFunction`

### Test Categories

#### Unit Tests
- Test individual functions and methods
- Use mocks for external dependencies
- Fast execution (< 100ms per test)
- No network or file system access

#### Integration Tests
- Test component interactions
- May use real dependencies (databases, APIs)
- Slower execution acceptable
- Require proper setup and teardown

#### End-to-End Tests
- Test complete workflows
- Use real or containerized dependencies
- Longest execution times
- Most comprehensive coverage

## Coverage Standards

### Target Coverage Levels
- **Critical packages**: 80%+ coverage
- **Core packages**: 60%+ coverage
- **Utility packages**: 40%+ coverage
- **Test utilities**: Coverage not required

### Current Status
See `docs/TEST_COVERAGE_REPORT.md` for detailed package-by-package analysis.

## Testing Best Practices

### Test Structure
```go
func TestComponentFunction(t *testing.T) {
    // Arrange
    setup := createTestSetup()

    // Act
    result, err := componentFunction(setup.input)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Mock Usage
- Use `testify/mock` for complex dependencies
- Create interface-based mocks in `pkg/testutil/mocks`
- Verify mock expectations after test execution

### Error Testing
- Always test error conditions
- Verify error messages and types
- Test error propagation through call stack

### Concurrency Testing
- Use `go test -race` to detect race conditions
- Test with multiple goroutines when applicable
- Use proper synchronization in tests

## Common Patterns

### Table-Driven Tests
```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    bool
        wantErr bool
    }{
        {"valid input", "valid", true, false},
        {"invalid input", "invalid", false, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := validate(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Setup and Teardown
```go
func TestWithSetup(t *testing.T) {
    // Setup
    cleanup := setupTestEnvironment(t)
    defer cleanup()

    // Test logic here
}
```

### Ginkgo/Gomega (BDD Style)
```go
var _ = Describe("Component", func() {
    BeforeEach(func() {
        // Setup before each test
    })

    AfterEach(func() {
        // Cleanup after each test
    })

    It("should handle valid input", func() {
        result := componentFunction("valid")
        Expect(result).To(Equal(expectedValue))
    })
})
```

## Running Tests

### Commands
- `task test` - Full test suite with coverage
- `task test:unit` - Unit tests only
- `task test:integration` - Integration tests only
- `go test -race ./...` - Run with race detection
- `go test -v ./pkg/component` - Verbose output for specific package

### Coverage Analysis
- `go test -coverprofile=coverage.out ./...`
- `go tool cover -html=coverage.out` - HTML coverage report
- `go tool cover -func=coverage.out` - Function-level coverage

## Continuous Integration

### Pre-commit Checks
- All tests must pass
- Coverage must not decrease
- No race conditions detected
- Linting passes without errors

### Quality Gates
- Minimum 60% coverage for new packages
- No failing tests in CI
- Performance regression detection
