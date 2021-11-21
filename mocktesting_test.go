package mocktesting

import (
	"sync"
)

func runInGoroutine(f func()) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		f()
	}()
	wg.Wait()
}

func stringsUniq(strs []string) []string {
	m := map[string]bool{}

	for _, s := range strs {
		m[s] = true
	}

	r := make([]string, 0, len(m))
	for s := range m {
		r = append(r, s)
	}

	return r
}
