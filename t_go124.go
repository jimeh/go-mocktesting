//go:build go1.24
// +build go1.24

package mocktesting

import (
	"context"
	"os"
)

// Context returns a context that is canceled just before Cleanup-registered
// functions are called.
//
// Cleanup functions can wait for any resources that shut down on
// Context.Done before the test or benchmark completes.
func (t *T) Context() context.Context {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.ctx
}

// Chdir calls os.Chdir(dir) and uses Cleanup to restore the current
// working directory to its original value after the test. On Unix, it
// also sets PWD environment variable for the duration of the test.
//
// Because Chdir affects the whole process, it cannot be used in parallel
// tests or tests with parallel ancestors.
func (t *T) Chdir(dir string) {
	// Store original directory on first call
	t.mux.Lock()
	needsCleanup := t.origDir == ""
	if needsCleanup {
		origDir, err := os.Getwd()
		if err != nil {
			t.mux.Unlock()
			err = os.ErrInvalid
			t.internalError(err)
			return
		}
		t.origDir = origDir
	}
	t.mux.Unlock()

	// Register cleanup outside of lock (Cleanup acquires lock)
	if needsCleanup {
		origDir := t.origDir
		t.Cleanup(func() {
			_ = os.Chdir(origDir)
		})
	}

	// Change directory
	if err := os.Chdir(dir); err != nil {
		t.internalError(err)
	}
}

// OrigDir returns the original working directory when the test was
// created, or the empty string if Chdir has not been called yet.
//
// This is a helper method specific to *mocktesting.T to allow inspection
// of the original directory in tests.
func (t *T) OrigDir() string {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.origDir
}
