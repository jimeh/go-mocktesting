// Package mocktesting provides a mock of *testing.T for the purpose of testing
// test helpers.
package mocktesting

import (
	"sync"
)

// Go runs the provided function in a new goroutine, and blocks until the
// goroutine has exited.
//
// This is essentially a helper function to avoid aborting the current goroutine
// when a *T instance aborts the goroutine that any of FailNow(), Fatal(),
// Fatalf(), SkipNow(), Skip(), or Skipf() are called from.
func Go(f func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
	wg.Wait()
}
