//go:build go1.24
// +build go1.24

package mocktesting

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestT_Context(t *testing.T) {
	t.Run("returns non-nil context", func(t *testing.T) {
		mt := NewT("TestContext")

		ctx := mt.Context()

		assert.NotNil(t, ctx)
	})

	t.Run("context is not canceled initially", func(t *testing.T) {
		mt := NewT("TestContext")

		ctx := mt.Context()

		select {
		case <-ctx.Done():
			t.Fatal("context should not be canceled initially")
		default:
			// Expected
		}
	})

	t.Run("context is canceled after test completes", func(t *testing.T) {
		mt := NewT("TestContext")
		var ctx context.Context

		mt.Run("subtest", func(t testing.TB) {
			subMt := t.(*T)
			ctx = subMt.Context()
		})

		// After subtest completes, its context should be canceled
		select {
		case <-ctx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("context should be canceled after test completes")
		}
	})

	t.Run("cleanup can wait on context", func(t *testing.T) {
		mt := NewT("TestContext")
		var subCtx context.Context

		mt.Run("subtest", func(t testing.TB) {
			subMt := t.(*T)
			subCtx = subMt.Context()

			// Add a cleanup function
			subMt.Cleanup(func() {
				// This will be recorded but not run automatically
			})
		})

		// After subtest completes, context should be canceled
		select {
		case <-subCtx.Done():
			// Expected - context is canceled
		case <-time.After(100 * time.Millisecond):
			t.Fatal("context should be canceled after subtest completes")
		}

		// Verify cleanup was registered
		subtests := mt.Subtests()
		assert.Len(t, subtests, 1)
		assert.Len(t, subtests[0].CleanupFuncs(), 1,
			"cleanup should be registered")
	})

	t.Run("same context returned on multiple calls", func(t *testing.T) {
		mt := NewT("TestContext")

		ctx1 := mt.Context()
		ctx2 := mt.Context()

		assert.Same(t, ctx1, ctx2)
	})

	t.Run("subtests have their own contexts", func(t *testing.T) {
		mt := NewT("TestContext")
		var ctx1, ctx2 context.Context

		mt.Run("subtest1", func(t testing.TB) {
			ctx1 = t.(*T).Context()
		})

		mt.Run("subtest2", func(t testing.TB) {
			ctx2 = t.(*T).Context()
		})

		assert.NotSame(t, ctx1, ctx2,
			"each subtest should have its own context")
	})
}

func TestT_Chdir(t *testing.T) {
	t.Run("changes directory", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(origDir)

		tempDir := t.TempDir()
		mt := NewT("TestChdir")

		runInGoroutine(func() {
			mt.Chdir(tempDir)
		})

		newDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, tempDir, newDir)

		// Cleanup
		os.Chdir(origDir)
	})

	t.Run("stores original directory", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(origDir)

		tempDir := t.TempDir()
		mt := NewT("TestChdir")

		runInGoroutine(func() {
			mt.Chdir(tempDir)
		})

		assert.Equal(t, origDir, mt.OrigDir())

		// Cleanup
		os.Chdir(origDir)
	})

	t.Run("registers cleanup", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(origDir)

		tempDir := t.TempDir()
		mt := NewT("TestChdir")

		runInGoroutine(func() {
			mt.Chdir(tempDir)
		})

		cleanups := mt.CleanupFuncs()
		assert.Len(t, cleanups, 1,
			"Chdir should register one cleanup function")

		// Cleanup
		os.Chdir(origDir)
	})

	t.Run("multiple calls change directory", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(origDir)

		tempDir1 := t.TempDir()
		tempDir2 := filepath.Join(tempDir1, "subdir")
		require.NoError(t, os.Mkdir(tempDir2, 0755))

		mt := NewT("TestChdir")

		runInGoroutine(func() {
			mt.Chdir(tempDir1)
			dir1, _ := os.Getwd()
			assert.Equal(t, tempDir1, dir1)

			mt.Chdir(tempDir2)
			dir2, _ := os.Getwd()
			assert.Equal(t, tempDir2, dir2)
		})

		// Original dir should still be stored
		assert.Equal(t, origDir, mt.OrigDir())

		// Should only have one cleanup (from first call)
		cleanups := mt.CleanupFuncs()
		assert.Len(t, cleanups, 1,
			"should only register cleanup once")

		// Cleanup
		os.Chdir(origDir)
	})

	t.Run("cleanup restores original directory", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)

		tempDir := t.TempDir()
		mt := NewT("TestChdir")

		runInGoroutine(func() {
			mt.Chdir(tempDir)

			// Verify we're in temp dir
			currentDir, _ := os.Getwd()
			assert.Equal(t, tempDir, currentDir)

			// Run cleanup
			cleanups := mt.CleanupFuncs()
			for _, cleanup := range cleanups {
				cleanup()
			}
		})

		// Verify we're back in original dir
		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, origDir, currentDir)
	})

	t.Run("error on invalid directory", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(origDir)

		mt := NewT("TestChdir", WithNoAbort())
		panicValue := ""

		runInGoroutine(func() {
			defer func() {
				if r := recover(); r != nil {
					panicValue = r.(error).Error()
				}
			}()
			mt.Chdir("/nonexistent/directory/that/does/not/exist")
		})

		assert.Contains(t, panicValue, "mocktesting:",
			"should panic with mocktesting error")
	})

	t.Run("works with subtests", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(origDir)

		tempDir := t.TempDir()
		mt := NewT("TestChdir")

		mt.Run("subtest", func(t testing.TB) {
			subMt := t.(*T)
			subMt.Chdir(tempDir)

			currentDir, _ := os.Getwd()
			assert.Equal(t, tempDir, currentDir)
		})

		// After subtest, should be back to original (if cleanup ran)
		// But since we're in the same process, we need to manually
		// restore for this test
		os.Chdir(origDir)
	})
}

func TestT_Run_Go124(t *testing.T) {
	t.Run("subtest context is canceled after completion", func(t *testing.T) {
		mt := NewT("TestRun")
		var subCtx context.Context

		mt.Run("subtest", func(t testing.TB) {
			subMt := t.(*T)
			subCtx = subMt.Context()
		})

		// After subtest completes, context should be canceled
		select {
		case <-subCtx.Done():
			// Expected - context is canceled
		case <-time.After(100 * time.Millisecond):
			t.Fatal("subtest context should be canceled after completion")
		}
	})

	t.Run("parent context not affected by subtest", func(t *testing.T) {
		mt := NewT("TestRun")
		parentCtx := mt.Context()

		mt.Run("subtest", func(t testing.TB) {
			// Subtest runs and completes
		})

		// Parent context should still be active
		select {
		case <-parentCtx.Done():
			t.Fatal("parent context should not be canceled")
		default:
			// Expected
		}
	})
}

func TestT_OrigDir(t *testing.T) {
	t.Run("empty before Chdir called", func(t *testing.T) {
		mt := NewT("TestOrigDir")

		origDir := mt.OrigDir()

		assert.Empty(t, origDir)
	})

	t.Run("set after Chdir called", func(t *testing.T) {
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(origDir)

		tempDir := t.TempDir()
		mt := NewT("TestOrigDir")

		runInGoroutine(func() {
			mt.Chdir(tempDir)
		})

		storedOrigDir := mt.OrigDir()
		assert.Equal(t, origDir, storedOrigDir)

		// Cleanup
		os.Chdir(origDir)
	})
}
