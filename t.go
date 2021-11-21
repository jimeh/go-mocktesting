package mocktesting

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestingT is an interface covering *mocktesting.T's internal use of
// *testing.T. See WithTestingT() for more details.
type TestingT interface {
	Fatal(args ...interface{})
}

// T is a fake/mock implementation of testing.T. It records all actions
// performed via all methods on the testing.T interface, so they can be
// inspected and asserted.
//
// It is specifically intended for testing test helpers which accept a
// *testing.T or *testing.B, so you can verify that the helpers call Fatal(),
// Error(), etc, as they need.
type T struct {
	name        string
	abort       bool
	baseTempdir string
	testingT    TestingT
	deadline    time.Time
	timeout     bool

	mux      sync.RWMutex
	skipped  bool
	failed   int
	parallel bool
	output   []string
	helpers  []string
	aborted  bool
	cleanups []func()
	env      map[string]string
	subtests []*T
	tempdirs []string

	// subtestNames is used to ensure subtests do not have conflicting names.
	subtestNames map[string]bool

	// mkdirTempFunc is used by the TempDir function instead of ioutil.TempDir()
	// if it is not nil. This is only used by tests for TempDir itself to ensure
	// it behaves correctly if temp directory creation fails.
	mkdirTempFunc func(string, string) (string, error)

	// Embed *testing.T to implement the testing.TB interface, which has a
	// private method to prevent it from being implemented. However that means
	// it's very difficult to test testing helpers.
	*testing.T
}

// Ensure T struct implements testing.TB interface.
var _ testing.TB = (*T)(nil)

func NewT(name string, options ...Option) *T {
	t := &T{
		name:        strings.ReplaceAll(name, " ", "_"),
		abort:       true,
		baseTempdir: os.TempDir(),
		deadline:    time.Now().Add(10 * time.Minute),
		timeout:     true,
	}

	for _, opt := range options {
		opt.apply(t)
	}

	return t
}

type Option interface {
	apply(*T)
}

type optionFunc func(*T)

func (fn optionFunc) apply(g *T) {
	fn(g)
}

// WithTimeout specifies a custom timeout for the mock test. It effectively
// determines the return values of Deadline().
//
// When given a zero-value time.Duration, Deadline() will act as if no timeout
// has been set.
//
// If this option is not used, the default timeout value is set to 10 minutes.
func WithTimeout(d time.Duration) Option {
	return optionFunc(func(t *T) {
		if d > 0 {
			t.timeout = true
			t.deadline = time.Now().Add(d)
		} else {
			t.timeout = false
			t.deadline = time.Time{}
		}
	})
}

// WithDeadline specifies a custom timeout for the mock test, but setting the
// deadline to an exact value, rather than setting it based on the offset from
// now of a time.Duration. It effectively determines the return values of
// Deadline().
//
// When given a empty time.Time{}, Deadline() will act as if no timeout has been
// set.
//
// If this option is not used, the default timeout value is set to 10 minutes.
func WithDeadline(d time.Time) Option {
	return optionFunc(func(t *T) {
		if d != (time.Time{}) {
			t.timeout = true
			t.deadline = d
		} else {
			t.timeout = false
			t.deadline = time.Time{}
		}
	})
}

// WithNoAbort disables aborting the current goroutine with runtime.Goexit()
// when SkipNow or FailNow is called. This should be used with care, as it
// causes behavior to diverge from normal *tesing.T, as code after calling
// t.Fatal() will be executed.
func WithNoAbort() Option {
	return optionFunc(func(t *T) {
		t.abort = false
	})
}

// WithBaseTempdir sets the base directory that TempDir() creates temporary
// directories within.
//
// If this option is not used, the default base directory used is os.TempDir().
func WithBaseTempdir(dir string) Option {
	return optionFunc(func(t *T) {
		if dir != "" {
			t.baseTempdir = dir
		}
	})
}

// WithTestingT accepts a *testing.T instance which is used to report internal
// errors within *mocktesting.T itself. For example if the TempDir() function
// fails to create a temporary directory on disk, it will call Fatal() on the
// *testing.T instance provided here.
//
// If this option is not used, internal errors will instead cause a panic.
func WithTestingT(testingT TestingT) Option {
	return optionFunc(func(t *T) {
		t.testingT = testingT
	})
}

func (t *T) goexit() {
	t.aborted = true
	if t.abort {
		runtime.Goexit()
	}
}

func (t *T) internalError(err error) {
	err = fmt.Errorf("mocktesting: %w", err)

	if t.testingT != nil {
		t.testingT.Fatal(err)
	} else {
		panic(err)
	}
}

func (t *T) Name() string {
	return t.name
}

func (t *T) Deadline() (time.Time, bool) {
	return t.deadline, t.timeout
}

func (t *T) Error(args ...interface{}) {
	t.Log(args...)
	t.Fail()
}

func (t *T) Errorf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.Fail()
}

func (t *T) Fail() {
	t.failed++
}

func (t *T) FailNow() {
	t.Fail()
	t.goexit()
}

func (t *T) Failed() bool {
	return t.failed > 0
}

func (t *T) Fatal(args ...interface{}) {
	t.Log(args...)
	t.FailNow()
}

func (t *T) Fatalf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.FailNow()
}

func (t *T) Log(args ...interface{}) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.output = append(t.output, fmt.Sprintln(args...))
}

func (t *T) Logf(format string, args ...interface{}) {
	t.mux.Lock()
	defer t.mux.Unlock()

	if len(format) == 0 || format[len(format)-1] != '\n' {
		format += "\n"
	}
	t.output = append(t.output, fmt.Sprintf(format, args...))
}

func (t *T) Parallel() {
	t.parallel = true
}

func (t *T) Skip(args ...interface{}) {
	t.Log(args...)
	t.SkipNow()
}

func (t *T) Skipf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.SkipNow()
}

func (t *T) SkipNow() {
	t.skipped = true
	t.goexit()
}

func (t *T) Skipped() bool {
	return t.skipped
}

func (t *T) Helper() {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}

	fnName := runtime.FuncForPC(pc).Name()

	t.mux.Lock()
	defer t.mux.Unlock()

	t.helpers = append(t.helpers, fnName)
}

func (t *T) Cleanup(f func()) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.cleanups = append(t.cleanups, f)
}

func (t *T) TempDir() string {
	// Allow setting MkdirTemp function for testing purposes.
	f := t.mkdirTempFunc
	if f == nil {
		f = ioutil.TempDir
	}

	dir, err := f(t.baseTempdir, "go-mocktesting*")
	if err != nil {
		err = fmt.Errorf("TempDir() failed to create directory: %w", err)
		t.internalError(err)
	}

	t.mux.Lock()
	defer t.mux.Unlock()
	t.tempdirs = append(t.tempdirs, dir)

	return dir
}

func (t *T) Run(name string, f func(testing.TB)) bool {
	name = t.newSubTestName(name)
	fullname := name
	if t.name != "" {
		fullname = t.name + "/" + name
	}

	subtest := NewT(fullname)
	subtest.abort = t.abort
	subtest.baseTempdir = t.baseTempdir
	subtest.testingT = t.testingT
	subtest.deadline = t.deadline
	subtest.timeout = t.timeout

	if t.subtestNames == nil {
		t.subtestNames = map[string]bool{}
	}

	t.mux.Lock()
	t.subtests = append(t.subtests, subtest)
	t.subtestNames[name] = true
	t.mux.Unlock()

	Go(func() {
		f(subtest)
	})

	if subtest.Failed() {
		t.Fail()
	}

	return !subtest.Failed()
}

func (t *T) newSubTestName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")

	if !t.subtestNames[name] {
		return name
	}

	i := 1
	for {
		n := name + "#" + fmt.Sprintf("%02d", i)
		if !t.subtestNames[n] {
			return n
		}

		i++
	}
}

//
// Inspection Methods which are not part of the testing.TB interface.
//

// Output returns a string slice of all output produced by calls to Log(),
// Logf(), Error(), Errorf(), Fatal(), Fatalf(), Skip(), and Skipf().
func (t *T) Output() []string {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.output
}

// CleanupFuncs returns a slice of functions given to Cleanup().
func (t *T) CleanupFuncs() []func() {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.cleanups
}

// CleanupNames returns a string slice of function names given to Cleanup().
func (t *T) CleanupNames() []string {
	r := make([]string, 0, len(t.cleanups))
	for _, f := range t.cleanups {
		p := reflect.ValueOf(f).Pointer()
		r = append(r, runtime.FuncForPC(p).Name())
	}

	return r
}

// FailedCount returns the number of times Error(), Errorf(), Fail(), Failf(),
// FailNow(), Fatal(), and Fatalf() were called.
func (t *T) FailedCount() int {
	return t.failed
}

// Aborted returns true if the TB instance aborted the current goroutine via
// runtime.Goexit(), which is called by FailNow() and SkipNow().
func (t *T) Aborted() bool {
	return t.aborted
}

// HelperNames returns a list of function names which called Helper().
func (t *T) HelperNames() []string {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.helpers
}

// Paralleled returns true if Parallel() has been called.
func (t *T) Paralleled() bool {
	return t.parallel
}

// Subtests returns a list map of *TB instances for any subtests executed via
// Run().
func (t *T) Subtests() []*T {
	if t.subtests == nil {
		t.mux.Lock()
		t.subtests = []*T{}
		t.mux.Unlock()
	}

	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.subtests
}
