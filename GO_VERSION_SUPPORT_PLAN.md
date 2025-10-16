# Go Version Support Plan for go-mocktesting

## Research Summary

### Current State
- **Base Go Version**: 1.15 (from go.mod)
- **Currently Supported**: Go 1.15, 1.16 (with Setenv method)
- **Latest Go Version**: 1.24
- **Missing Methods**: Context(), Chdir()

### Complete testing.T Method Inventory

All public methods on *testing.T as of Go 1.24:
- Chdir ❌ (Go 1.24) - **MISSING**
- Cleanup ✅
- Context ❌ (Go 1.24) - **MISSING**
- Deadline ✅
- Error ✅
- Errorf ✅
- Fail ✅
- FailNow ✅
- Failed ✅
- Fatal ✅
- Fatalf ✅
- Helper ✅
- Log ✅
- Logf ✅
- Name ✅
- Parallel ✅
- Run ✅
- Setenv ✅ (Go 1.16+)
- Skip ✅
- SkipNow ✅
- Skipf ✅
- Skipped ✅
- TempDir ✅

## Version-by-Version Analysis

### Go 1.15 (Base)
**File**: `t.go`
**Status**: ✅ Fully implemented

Methods implemented:
- Core test methods (Fail, FailNow, Failed, Error, Errorf, Fatal, Fatalf)
- Logging (Log, Logf)
- Skipping (Skip, Skipf, SkipNow, Skipped)
- Helpers (Helper)
- Test organization (Name, Parallel, Run, Deadline)
- Cleanup (Cleanup)
- Temporary directories (TempDir)

### Go 1.16
**File**: `t_go116.go` (with build tag `//go:build go1.16`)
**Status**: ✅ Fully implemented

New methods:
- `Setenv(key, value string)` - Set environment variable for test

### Go 1.17
**Status**: ✅ No new testing.T methods

### Go 1.18
**Status**: ✅ No new testing.T methods

### Go 1.19
**Status**: ✅ No new testing.T methods

### Go 1.20
**Status**: ✅ No new testing.T methods

### Go 1.21
**Status**: ✅ No new testing.T methods

### Go 1.22
**Status**: ✅ No new testing.T methods

### Go 1.23
**Status**: ✅ No new testing.T methods

### Go 1.24
**File**: `t_go124.go` (needs to be created)
**Status**: ❌ Not implemented

New methods:
1. **Context() context.Context**
   - Returns a context.Context that is canceled just before
     Cleanup-registered functions are called
   - Allows cleanup functions to wait for resources that shut down
     on Context.Done before the test completes
   
2. **Chdir(dir string)**
   - Calls os.Chdir(dir) and uses Cleanup to restore the current
     working directory after the test
   - On Unix, also sets PWD environment variable for the duration
   - Cannot be used in parallel tests (affects whole process)

## Implementation Plan

### Phase 1: Add Go 1.24 Support ⏳

#### Step 1.1: Create `t_go124.go`
- Add build constraint: `//go:build go1.24`
- Implement `Context() context.Context` method
  - Store context in T struct (add ctx field)
  - Create context on T initialization
  - Cancel context before cleanup functions run
  - Return the context
- Implement `Chdir(dir string)` method
  - Call os.Chdir(dir)
  - Register cleanup to restore original directory
  - Store original PWD and restore it
  - Track current directory changes

#### Step 1.2: Update T struct in `t.go`
- Add `ctx context.Context` field
- Add `ctxCancel context.CancelFunc` field  
- Add `origDir string` field for Chdir tracking
- Initialize context in NewT()

#### Step 1.3: Create `t_go124_test.go`
- Add build constraint: `//go:build go1.16`
- Test Context() method:
  - Verify context is not nil
  - Verify context is canceled during cleanup
  - Test that cleanup can use Context.Done
- Test Chdir() method:
  - Verify directory changes work
  - Verify directory is restored after test
  - Verify PWD environment variable handling
  - Verify panic/error on parallel tests

#### Step 1.4: Update TestT_methods test
- Ensure it runs on all Go versions
- Verify no failures for Go 1.24+

### Phase 2: Update CI Configuration ⏳

#### Step 2.1: Update `.github/workflows/ci.yml`
Current test matrix:
```yaml
go_version:
  - "1.15"
  - "1.16"
  - "1.17"
```

Add newer versions:
```yaml
go_version:
  - "1.15"
  - "1.16"
  - "1.17"
  - "1.18"
  - "1.19"
  - "1.20"
  - "1.21"
  - "1.22"
  - "1.23"
  - "1.24"
```

#### Step 2.2: Update other CI jobs
- Update `lint` job to use Go 1.24
- Update `tidy` job to use Go 1.24
- Update `cov` job to use Go 1.24

### Phase 3: Documentation Updates ⏳

#### Step 3.1: Update README.md
- Note Go 1.24 support
- Document new Context() and Chdir() methods
- Add examples if appropriate

#### Step 3.2: Update CHANGELOG.md
Add new version entry:
```markdown
## [Next Release]

### Features

* **testing:** Add support for Go 1.24 Context() and Chdir() methods
* **ci:** Test against Go 1.18-1.24
```

#### Step 3.3: Update go.mod
Consider whether to bump minimum version or keep at 1.15

### Phase 4: Testing & Validation ⏳

#### Step 4.1: Run TestT_methods
```bash
go test -v -run TestT_methods
```
Should pass with no failures.

#### Step 4.2: Run full test suite on all versions
```bash
for v in 1.15 1.16 1.17 1.18 1.19 1.20 1.21 1.22 1.23 1.24; do
  go$v test -v -race ./...
done
```

#### Step 4.3: Verify coverage
```bash
make cov
```
Ensure new methods are covered.

## Implementation Details for Go 1.24 Methods

### Context() Implementation

```go
//go:build go1.24
// +build go1.24

package mocktesting

import "context"

// Context returns a context that is canceled just before Cleanup-registered
// functions are called.
func (t *T) Context() context.Context {
	t.mux.RLock()
	defer t.mux.RUnlock()
	return t.ctx
}
```

Required struct changes:
```go
type T struct {
	// ... existing fields ...
	ctx       context.Context
	ctxCancel context.CancelFunc
}
```

Initialize in NewT():
```go
func NewT(name string, options ...Option) *T {
	ctx, cancel := context.WithCancel(context.Background())
	t := &T{
		// ... existing fields ...
		ctx:       ctx,
		ctxCancel: cancel,
	}
	// ... rest of init ...
}
```

### Chdir() Implementation

```go
//go:build go1.24
// +build go1.24

package mocktesting

import (
	"os"
)

// Chdir calls os.Chdir(dir) and uses Cleanup to restore the current
// working directory to its original value after the test.
func (t *T) Chdir(dir string) {
	t.mux.Lock()
	defer t.mux.Unlock()
	
	// Store original directory on first call
	if t.origDir == "" {
		origDir, err := os.Getwd()
		if err != nil {
			t.internalError(err)
			return
		}
		t.origDir = origDir
		
		// Register cleanup to restore directory
		t.Cleanup(func() {
			os.Chdir(t.origDir)
		})
	}
	
	// Change directory
	if err := os.Chdir(dir); err != nil {
		t.internalError(err)
	}
}
```

Required struct changes:
```go
type T struct {
	// ... existing fields ...
	origDir string
}
```

## Key Design Decisions

### Build Tags
Use `//go:build go1.24` syntax (not older `// +build` only) for
consistency with modern Go.

### Context Lifecycle
- Context is created at T initialization
- Context is NOT canceled during test execution
- Context IS canceled before Cleanup functions run
- This allows cleanup functions to wait on Context.Done

### Chdir Behavior
- First call to Chdir stores original directory
- Cleanup is registered once to restore directory
- Subsequent calls change directory but don't add more cleanups
- Errors in Chdir call internalError (consistent with TempDir)

### Backward Compatibility
- Keep base version at Go 1.15 in go.mod
- Use build tags to conditionally compile version-specific methods
- All existing code continues to work on Go 1.15+
- New methods only available when compiled with Go 1.24+

## Testing Strategy

### Unit Tests
Each new method needs comprehensive unit tests:
- Context():
  - Returns non-nil context
  - Context is canceled before cleanup
  - Can be used in cleanup functions
  - Works with subtests
- Chdir():
  - Directory changes work
  - Directory is restored
  - Multiple calls work correctly
  - Error handling

### Integration Tests
- Verify TestT_methods passes (the reflective test)
- Test interaction with existing methods
- Test with subtests (Run())
- Test error conditions

### CI Testing
- Test on all Go versions 1.15-1.24
- Test on multiple OS (Linux, macOS, Windows)
- Run with race detector
- Verify coverage

## Timeline Estimate

- Phase 1 (Implementation): 2-3 hours
- Phase 2 (CI Updates): 30 minutes
- Phase 3 (Documentation): 30 minutes
- Phase 4 (Testing): 1 hour
- **Total**: ~4-5 hours

## Risk Assessment

### Low Risk
- Go 1.17-1.23 have no new methods (already compatible)
- Implementation pattern established with Go 1.16 (Setenv)
- Good test coverage with TestT_methods

### Medium Risk
- Context lifecycle management needs careful implementation
- Chdir affects process state (need proper cleanup)
- CI configuration changes need validation

### Mitigation
- Follow established patterns from Setenv implementation
- Write comprehensive tests before implementation
- Test on multiple Go versions locally before CI
- Use build tags to isolate version-specific code

## Success Criteria

1. ✅ TestT_methods passes on Go 1.24
2. ✅ All existing tests pass on Go 1.15-1.24
3. ✅ New methods have >90% test coverage
4. ✅ CI tests pass on all versions
5. ✅ Documentation is updated
6. ✅ No breaking changes to existing API

## Notes

- The test failure currently shows: "Chdir" and "Context" are not
  implemented
- These are the ONLY two methods missing
- Go 1.17-1.23 are already fully supported (no new methods)
- Implementation should be straightforward following existing patterns
