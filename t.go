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

// T is a fake/mock implementation of *testing.T. All methods available on
// *testing.T are available on *T with the exception of Run(), which has a
// slightly different func type.
//
// All method calls against *T are recorded, so they can be inspected and
// asserted later. To be able to pass in *testing.T or *mocktesting.T, functions
// will need to use an interface instead of *testing.T explicitly.
//
// For basic use cases, the testing.TB interface should suffice. For more
// advanced use cases, create a custom interface that exactly specifies the
// methods of *testing.T which are needed, and then freely pass *testing.T or
// *mocktesting.T.
type T struct {
	// Settings - These fields control the behavior of T.
	name        string
	abort       bool
	baseTempdir string
	testingT    TestingT
	deadline    time.Time
	timeout     bool

	// State - Fields which record how T has been modified via method calls.
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

// Name returns the name given to the *T instance.
func (t *T) Name() string {
	return t.name
}

// Name returns the time at which the *T instance is set to timeout. If no
// timeout is set, the bool return value is false, otherwise it is true.
func (t *T) Deadline() (time.Time, bool) {
	return t.deadline, t.timeout
}

// Error logs the given args with Log(), and then calls Fail() to mark the *T
// instance as failed.
func (t *T) Error(args ...interface{}) {
	t.Log(args...)
	t.Fail()
}

// Errorf logs the given format and args with Logf(), and then calls Fail() to
// mark the *T instance as failed.
func (t *T) Errorf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.Fail()
}

// Fail marks the *T instance as having failed. You can check if the *T instance
// has been failed with Failed(), or how many times it has been failed with
// FailedCount().
func (t *T) Fail() {
	t.failed++
}

// FailNow marks the *T instance as having failed, and also aborts the current
// goroutine with runtime.Goexit(). If the WithNoAbort() option was used when
// initializing the *T instance, runtime.Goexit() will not be called.
func (t *T) FailNow() {
	t.Fail()
	t.goexit()
}

// Failed returns true if the *T instance has been marked as failed.
func (t *T) Failed() bool {
	return t.failed > 0
}

// Fatal logs the given args with Log(), and then calls FailNow() to fail the *T
// instance and abort the current goroutine.
//
// See FailNow() and WithNoAbort() for details about how abort works.
func (t *T) Fatal(args ...interface{}) {
	t.Log(args...)
	t.FailNow()
}

// Fatalf logs the given format and args with Logf(), and then calls FailNow()
// to fail the *T instance and abort the current goroutine.
//
// See FailNow() and WithNoAbort() for details about how abort works.
func (t *T) Fatalf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.FailNow()
}

// Log renders given args to a string with fmt.Sprintln() and stores the result
// in a string slice which can be accessed with Output().
func (t *T) Log(args ...interface{}) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.output = append(t.output, fmt.Sprintln(args...))
}

// Logf renders given format and args to a string with fmt.Sprintf() and stores
// the result in a string slice which can be accessed with Output().
func (t *T) Logf(format string, args ...interface{}) {
	t.mux.Lock()
	defer t.mux.Unlock()

	if len(format) == 0 || format[len(format)-1] != '\n' {
		format += "\n"
	}
	t.output = append(t.output, fmt.Sprintf(format, args...))
}

// Parallel marks the *T instance to indicate Parallel() has been called.
// Use Paralleled() to check if Parallel() has been called.
func (t *T) Parallel() {
	t.parallel = true
}

// Skip logs the given args with Log(), and then uses SkipNow() to mark the *T
// instance as skipped and aborts the current goroutine.
//
// See SkipNow() for more details about aborting the current goroutine.
func (t *T) Skip(args ...interface{}) {
	t.Log(args...)
	t.SkipNow()
}

// Skipf logs the given format and args with Logf(), and then uses SkipNow() to
// mark the *T instance as skipped and aborts the current goroutine.
//
// See SkipNow() for more details about aborting the current goroutine.
func (t *T) Skipf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.SkipNow()
}

// SkipNow marks the *T instance as skipped, and then aborts the current
// goroutine with runtime.Goexit(). If the WithNoAbort() option was used when
// initializing the *T instance, runtime.Goexit() will not be called.
func (t *T) SkipNow() {
	t.skipped = true
	t.goexit()
}

// Skipped returns true if the *T instance has been marked as skipped, otherwise
// it returns false.
func (t *T) Skipped() bool {
	return t.skipped
}

// Helper marks the function that is calling Helper() as a helper function.
// Within *T it simply stores a reference to the function.
//
// The list of functions which have called Helper() can be inspected with
// HelperNames(). The names are resolved using runtime.FuncForPC(), meaning they
// include the absolute Go package path to the function, along with the function
// name itself.
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

// Cleanup registers a cleanup function. *T does not run cleanup functions, it
// simply records them for the purpose of later inspection via CleanupFuncs() or
// CleanupNames().
func (t *T) Cleanup(f func()) {
	t.mux.Lock()
	defer t.mux.Unlock()

	t.cleanups = append(t.cleanups, f)
}

// TempDir creates an actual temporary directory on the system using
// ioutil.TempDir(). This actually does perform a action, rather than just
// recording the fact the method was called list most other *T methods.
//
// This is because returning a string that is not the path to a real directory,
// would most likely be useless. Hence it does create a real temporary
// directory.
//
// It is important to note that the temporary directory is not cleaned up by
// mocktesting. But it is created via ioutil.TempDir(), so the operating system
// should eventually clean it up.
//
// A string slice of temporary directory paths created by calls to TempDir() can
// be accessed with TempDirs().
func (t *T) TempDir() string {
	// Allow setting MkdirTemp function for the purpose of testing mocktesting
	// itself..
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

// Run allows running sub-tests just very much like *testing.T. The one
// difference is that the function argument accepts a testing.TB instead of
// *testing.T type. This is to allow passing a *mocktesting.T to the sub-test
// function instead of a *testing.T.
//
// Sub-test functions are executed in a separate blocking goroutine, so calls to
// SkipNow() and FailNow() abort the new goroutine that the sub-test is running
// in, rather than the gorouting which is executing Run().
//
// The sub-test function will receive a new instance of *T which is a sub-test,
// which name and other attributes set accordingly.
//
// If any sub-test *T is marked as failed, the parent *T instance will also
// be marked as failed.
//
// The list of sub-test *T instances can be accessed with Subtests().
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

// Output returns a string slice of all output produced by calls to Log() and
// Logf().
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

// CleanupNames returns a string slice of function names given to Cleanup(). The
// names are resolved using runtime.FuncForPC(), meaning they include the
// absolute Go package path to the function, along with the function name
// itself.
func (t *T) CleanupNames() []string {
	r := make([]string, 0, len(t.cleanups))
	for _, f := range t.cleanups {
		p := reflect.ValueOf(f).Pointer()
		r = append(r, runtime.FuncForPC(p).Name())
	}

	return r
}

// FailedCount returns the number of times the *T instance has been marked as
// failed.
func (t *T) FailedCount() int {
	return t.failed
}

// Aborted returns true if the *T instance aborted the current goroutine via
// runtime.Goexit(), which is called by FailNow() and SkipNow().
//
// This returns true even if *T was initialized using the WithNoAbort() option.
// Because the test was still instructed to abort, which is a separate matter
// than that *T was specifically set to not abort the current goroutine.
func (t *T) Aborted() bool {
	return t.aborted
}

// HelperNames returns a list of function names which called Helper(). The names
// are resolved using runtime.FuncForPC(), meaning they include the absolute Go
// package path to the function, along with the function name itself.
func (t *T) HelperNames() []string {
	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.helpers
}

// Paralleled returns true if Parallel() has been called.
func (t *T) Paralleled() bool {
	return t.parallel
}

// Subtests returns a slice of *T instances created for any subtests executed
// via Run().
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

// TempDirs returns a string slice of temporary directories created by
// TempDir().
func (t *T) TempDirs() []string {
	if t.tempdirs == nil {
		t.mux.Lock()
		t.tempdirs = []string{}
		t.mux.Unlock()
	}

	t.mux.RLock()
	defer t.mux.RUnlock()

	return t.tempdirs
}
