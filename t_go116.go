//go:build go1.16
// +build go1.16

package mocktesting

func (t *T) Setenv(key string, value string) {
	t.mux.Lock()
	defer t.mux.Unlock()

	if t.env == nil {
		t.env = map[string]string{}
	}

	if key != "" {
		t.env[key] = value
	}
}

// Getenv returns a map[string]string of keys/values given to Setenv().
func (t *T) Getenv() map[string]string {
	if t.env == nil {
		t.mux.Lock()
		t.env = map[string]string{}
		t.mux.Unlock()
	}

	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.env
}
