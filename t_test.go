package mocktesting

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertEqualMocktestingT(t *testing.T, want *T, got *T) {
	assert.Equalf(t, want.name, got.name, "name field in %s", got.name)
	assert.Equalf(t, want.abort, got.abort, "abort field in %s", got.name)
	assert.Equalf(t,
		want.baseTempdir, got.baseTempdir,
		"baseTempdir field in %s", got.name,
	)
	assert.Equalf(t,
		want.testingT, got.testingT,
		"testingT field in %s", got.name,
	)
	assert.WithinDurationf(t,
		want.deadline, got.deadline, 1*time.Second,
		"deadline field in %s", got.name,
	)
	assert.Equalf(t, want.timeout, got.timeout, "timeout field in %s", got.name)
	assert.Equalf(t, want.skipped, got.skipped, "skipped field in %s", got.name)
	assert.Equalf(t, want.failed, got.failed, "failed field in %s", got.name)
	assert.Equalf(t,
		want.parallel, got.parallel,
		"parallel field in %s", got.name,
	)
	assert.Equalf(t, want.output, got.output, "output field in %s", got.name)
	assert.Equalf(t, want.helpers, got.helpers, "helpers field in %s", got.name)
	assert.Equalf(t, want.aborted, got.aborted, "aborted field in %s", got.name)

	wantFuncs := make([]string, 0, len(want.cleanups))
	for _, f := range want.cleanups {
		p := reflect.ValueOf(f).Pointer()
		wantFuncs = append(wantFuncs, runtime.FuncForPC(p).Name())
	}
	gotFuncs := make([]string, 0, len(got.cleanups))
	for _, f := range got.cleanups {
		p := reflect.ValueOf(f).Pointer()
		gotFuncs = append(gotFuncs, runtime.FuncForPC(p).Name())
	}
	assert.Equalf(t, wantFuncs, gotFuncs, "cleanups field in %s", got.name)

	assert.Equalf(t, want.env, got.env, "env field in %s", got.name)

	for i, wantSubTest := range want.subtests {
		gotSubTest := got.subtests[i]
		assertEqualMocktestingT(t, wantSubTest, gotSubTest)
	}

	assert.Equalf(t,
		want.tempdirs, got.tempdirs,
		"tempdirs field in %s", got.name,
	)
	assert.Equalf(t,
		want.subtestNames, got.subtestNames,
		"subtestNames field in %s", got.name,
	)
}

// TestT_methods is a horrible hack of a test to verify that *T directly
// implements/overloads all exported methods of *testing.T. The goal is for this
// test to fail on future versions of Go which add new methods to *testing.T.
func TestT_methods(t *testing.T) {
	// Methods should be defined on a file within the same directory as this
	// test file.
	pc, _, _, _ := runtime.Caller(0)
	testFile, _ := runtime.FuncForPC(pc).FileLine(pc)
	wantDir := filepath.Dir(testFile)

	tType := reflect.TypeOf(&T{})
	testingType := reflect.TypeOf(t)

	for i := 0; i < testingType.NumMethod(); i++ {
		method := testingType.Method(i)
		t.Run(method.Name, func(t *testing.T) {
			mth, ok := tType.MethodByName(method.Name)
			if !ok {
				require.FailNowf(t, "method not implemented",
					"*mocktesting.T does not implement method %s from "+
						"*testing.T",
					method.Name,
				)
			}

			fp := mth.Func.Pointer()
			file, line := runtime.FuncForPC(fp).FileLine(fp)
			dir := filepath.Dir(file)

			if dir != wantDir || line <= 1 {
				require.FailNowf(t, "method not implemented",
					"*mocktesting.T does not implement method %s from "+
						"*testing.T",
					method.Name,
				)
			}
		})
	}
}

func TestNewT(t *testing.T) {
	testingT := &T{name: "real"}

	type args struct {
		name    string
		options []Option
	}
	tests := []struct {
		name string
		args args
		want *T
	}{
		{
			name: "empty name",
			args: args{name: ""},
			want: &T{
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
		},
		{
			name: "with a name",
			args: args{name: "TestFooBar_Nope"},
			want: &T{
				name:        "TestFooBar_Nope",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
		},
		{
			name: "name with spaces",
			args: args{name: "foo bar yep"},
			want: &T{
				name:        "foo_bar_yep",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
		},
		{
			name: "with options",
			args: args{
				name: "with options",
				options: []Option{
					WithTimeout(6 * time.Minute),
					WithNoAbort(),
					WithBaseTempdir("/tmp/go-mocktesting"),
					WithTestingT(testingT),
				},
			},
			want: &T{
				name:        "with_options",
				abort:       false,
				baseTempdir: "/tmp/go-mocktesting",
				testingT:    testingT,
				deadline:    time.Now().Add(6 * time.Minute),
				timeout:     true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewT(tt.args.name, tt.args.options...)

			// assert.Equal(t, tt.want, got)
			assertEqualMocktestingT(t, tt.want, got)
		})
	}
}

func TestWithTimeout(t *testing.T) {
	type fields struct {
		deadline time.Time
		timeout  bool
	}
	type args struct {
		d time.Duration
	}
	tests := []struct {
		name string
		args args
		want fields
	}{
		{
			name: "zero",
			args: args{d: time.Duration(0)},
			want: fields{
				timeout:  false,
				deadline: time.Time{},
			},
		},
		{
			name: "1 minutes",
			args: args{d: 1 * time.Minute},
			want: fields{
				timeout:  true,
				deadline: time.Now().Add(1 * time.Minute),
			},
		},
		{
			name: "10 minutes",
			args: args{d: 10 * time.Minute},
			want: fields{
				timeout:  true,
				deadline: time.Now().Add(10 * time.Minute),
			},
		},
		{
			name: "3 hours",
			args: args{d: 3 * time.Hour},
			want: fields{
				timeout:  true,
				deadline: time.Now().Add(3 * time.Hour),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{}

			WithTimeout(tt.args.d).apply(mt)

			assert.Equal(t, tt.want.timeout, mt.timeout)
			assert.WithinDuration(t,
				tt.want.deadline, mt.deadline, 1*time.Second,
			)
		})
	}
}

func TestWithDeadline(t *testing.T) {
	in1Minute := time.Now().Add(1 * time.Minute)
	in10Minutes := time.Now().Add(10 * time.Minute)
	in3Hours := time.Now().Add(3 * time.Hour)

	type fields struct {
		deadline time.Time
		timeout  bool
	}
	type args struct {
		d time.Time
	}
	tests := []struct {
		name string
		args args
		want fields
	}{
		{
			name: "zero",
			args: args{d: time.Time{}},
			want: fields{
				timeout:  false,
				deadline: time.Time{},
			},
		},
		{
			name: "1 minutes",
			args: args{d: in1Minute},
			want: fields{
				timeout:  true,
				deadline: in1Minute,
			},
		},
		{
			name: "10 minutes",
			args: args{d: in10Minutes},
			want: fields{
				timeout:  true,
				deadline: in10Minutes,
			},
		},
		{
			name: "3 hours",
			args: args{d: in3Hours},
			want: fields{
				timeout:  true,
				deadline: in3Hours,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{}

			WithDeadline(tt.args.d).apply(mt)

			assert.Equal(t, tt.want.timeout, mt.timeout)
			assert.Equal(t, tt.want.deadline, mt.deadline)
		})
	}
}

func TestWithNoAbort(t *testing.T) {
	mt := &T{abort: true}

	WithNoAbort().apply(mt)

	assert.Equal(t, false, mt.abort)
}

func TestWithBaseTempdir(t *testing.T) {
	type args struct {
		dir string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty string",
			args: args{dir: ""},
			want: os.TempDir(),
		},
		{
			name: "non-empty string",
			args: args{dir: "/tmp/foo-bar-nope"},
			want: "/tmp/foo-bar-nope",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{baseTempdir: os.TempDir()}

			WithBaseTempdir(tt.args.dir).apply(mt)

			assert.Equal(t, tt.want, mt.baseTempdir)
		})
	}
}

func TestWithTestingT(t *testing.T) {
	fakeTestingT := &T{name: "parent"}

	type args struct {
		t testing.TB
	}
	tests := []struct {
		name string
		args args
		want TestingT
	}{
		{
			name: "nil",
			args: args{t: nil},
			want: nil,
		},
		{
			name: "with *TB instance",
			args: args{t: fakeTestingT},
			want: fakeTestingT,
		},
		{
			name: "with *testing.T instance",
			args: args{t: t},
			want: t,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{baseTempdir: os.TempDir()}

			WithTestingT(tt.args.t).apply(mt)

			assert.Equal(t, tt.want, mt.testingT)
		})
	}
}

func TestT_Name(t *testing.T) {
	type fields struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "empty",
			fields: fields{name: ""},
			want:   "",
		},
		{
			name:   "foo",
			fields: fields{name: "foo"},
			want:   "foo",
		},
		{
			name:   "foo/bar",
			fields: fields{name: "foo/bar"},
			want:   "foo/bar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{name: tt.fields.name}

			got := mt.Name()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_Deadline(t *testing.T) {
	in1Minute := time.Now().Add(1 * time.Minute)
	in10Minutes := time.Now().Add(10 * time.Minute)
	in3Hours := time.Now().Add(3 * time.Hour)

	type fields struct {
		deadline time.Time
		timeout  bool
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Time
		wantOK bool
	}{
		{
			name:   "empty",
			fields: fields{},
			want:   time.Time{},
			wantOK: false,
		},
		{
			name: "in 1 minutes",
			fields: fields{
				deadline: in1Minute,
				timeout:  true,
			},
			want:   in1Minute,
			wantOK: true,
		},
		{
			name: "in 10 minutes",
			fields: fields{
				deadline: in10Minutes,
				timeout:  true,
			},
			want:   in10Minutes,
			wantOK: true,
		},
		{
			name: "in 3 hours",
			fields: fields{
				deadline: in3Hours,
				timeout:  true,
			},
			want:   in3Hours,
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{
				deadline: tt.fields.deadline,
				timeout:  tt.fields.timeout,
			}

			got, gotOK := mt.Deadline()

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOK, gotOK)
		})
	}
}

func TestT_Error(t *testing.T) {
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []string
	}{
		{
			name:     "no args",
			args:     args{},
			wantLogs: []string{"\n"},
		},
		{
			name:     "one arg",
			args:     args{args: []interface{}{"hello world"}},
			wantLogs: []string{"hello world\n"},
		},
		{
			name: "many args",
			args: args{
				args: []interface{}{
					"hello world",
					"where's my car?",
					1024,
				},
			},
			wantLogs: []string{"hello world where's my car? 1024\n"},
		},
	}
	variants := map[int]string{
		0: "not failed",
		1: "failed once",
		2: "failed twice",
	}
	for _, tt := range tests {
		for failedCount, nameSuffix := range variants {
			t.Run(tt.name+", "+nameSuffix, func(t *testing.T) {
				mt := &T{failed: failedCount}

				mt.Error(tt.args.args...)

				assert.Equal(t, failedCount+1, mt.failed)
				assert.Equal(t, tt.wantLogs, mt.output)
			})
		}
	}
}

func TestT_Errorf(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []string
	}{
		{
			name:     "no format or args",
			args:     args{},
			wantLogs: []string{"\n"},
		},
		{
			name:     "format with no args",
			args:     args{format: "something went wrong"},
			wantLogs: []string{"something went wrong\n"},
		},
		{
			name: "format with args",
			args: args{
				format: "something went wrong: %s (%d)",
				args: []interface{}{
					"not found",
					404,
				},
			},
			wantLogs: []string{"something went wrong: not found (404)\n"},
		},
	}
	variants := map[int]string{
		0: "not failed",
		1: "failed once",
		2: "failed twice",
	}
	for _, tt := range tests {
		for failedCount, nameSuffix := range variants {
			t.Run(tt.name+", "+nameSuffix, func(t *testing.T) {
				mt := &T{failed: failedCount}

				mt.Errorf(tt.args.format, tt.args.args...)

				assert.Equal(t, failedCount+1, mt.failed)
				assert.Equal(t, tt.wantLogs, mt.output)
			})
		}
	}
}

func TestT_Fail(t *testing.T) {
	type fields struct {
		failed int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "not failed",
			fields: fields{failed: 0},
			want:   1,
		},
		{
			name:   "failed once",
			fields: fields{failed: 1},
			want:   2,
		},
		{
			name:   "failed twice",
			fields: fields{failed: 2},
			want:   3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{failed: tt.fields.failed}

			mt.Fail()

			assert.Equal(t, tt.want, mt.failed)
		})
	}
}

func TestT_FailNow(t *testing.T) {
	type fields struct {
		abort  bool
		failed int
	}
	tests := []struct {
		name            string
		fields          fields
		wantFailedCount int
	}{
		{
			name:            "not failed",
			fields:          fields{abort: true, failed: 0},
			wantFailedCount: 1,
		},
		{
			name:            "not failed, without abort",
			fields:          fields{abort: false, failed: 0},
			wantFailedCount: 1,
		},
		{
			name:            "failed once",
			fields:          fields{abort: true, failed: 1},
			wantFailedCount: 2,
		},
		{
			name:            "failed once, without abort",
			fields:          fields{abort: false, failed: 1},
			wantFailedCount: 2,
		},
		{
			name:            "failed twice",
			fields:          fields{abort: true, failed: 2},
			wantFailedCount: 3,
		},
		{
			name:            "failed twice, without abort",
			fields:          fields{abort: false, failed: 2},
			wantFailedCount: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{
				abort:  tt.fields.abort,
				failed: tt.fields.failed,
			}

			halted := true
			runInGoroutine(func() {
				mt.FailNow()
				halted = false
			})

			assert.Equal(t, tt.wantFailedCount, mt.failed)
			assert.Equal(t, true, mt.aborted)
			assert.Equal(t, tt.fields.abort, halted)
			assert.Empty(t, mt.output)
		})
	}
}

func TestT_Failed(t *testing.T) {
	type fields struct {
		failed int
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "not failed",
			fields: fields{failed: 0},
			want:   false,
		},
		{
			name:   "failed once",
			fields: fields{failed: 1},
			want:   true,
		},
		{
			name:   "failed twice",
			fields: fields{failed: 2},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{failed: tt.fields.failed}

			got := mt.Failed()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_FailedCount(t *testing.T) {
	type fields struct {
		failed int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "not failed",
			fields: fields{failed: 0},
			want:   0,
		},
		{
			name:   "failed once",
			fields: fields{failed: 1},
			want:   1,
		},
		{
			name:   "failed twice",
			fields: fields{failed: 2},
			want:   2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{failed: tt.fields.failed}

			got := mt.FailedCount()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_Fatal(t *testing.T) {
	type fields struct {
		abort  bool
		failed int
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []string
	}{
		{
			name:     "no args",
			args:     args{},
			wantLogs: []string{"\n"},
		},
		{
			name:     "one arg",
			args:     args{args: []interface{}{"hello world"}},
			wantLogs: []string{"hello world\n"},
		},
		{
			name: "many args",
			args: args{
				args: []interface{}{
					"hello world",
					"where's my car?",
					1024,
				},
			},
			wantLogs: []string{"hello world where's my car? 1024\n"},
		},
	}
	variants := map[string]fields{
		", with abort, not failed":      {abort: true, failed: 0},
		", with abort, failed once":     {abort: true, failed: 1},
		", with abort, failed twice":    {abort: true, failed: 2},
		", without abort, not failed":   {abort: false, failed: 0},
		", without abort, failed once":  {abort: false, failed: 1},
		", without abort, failed twice": {abort: false, failed: 2},
	}
	for _, tt := range tests {
		for nameSuffix, flds := range variants {
			t.Run(tt.name+nameSuffix, func(t *testing.T) {
				mt := &T{
					abort:  flds.abort,
					failed: flds.failed,
				}
				halted := true

				runInGoroutine(func() {
					mt.Fatal(tt.args.args...)
					halted = false
				})

				assert.Equal(t, flds.failed+1, mt.failed)
				assert.Equal(t, true, mt.aborted)
				assert.Equal(t, flds.abort, halted)

				assert.Equal(t, tt.wantLogs, mt.output)
			})
		}
	}
}

func TestT_Fatalf(t *testing.T) {
	type fields struct {
		abort  bool
		failed int
	}
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []string
	}{
		{
			name:     "no format or args",
			args:     args{},
			wantLogs: []string{"\n"},
		},
		{
			name:     "format with no args",
			args:     args{format: "something went wrong"},
			wantLogs: []string{"something went wrong\n"},
		},
		{
			name: "format with args",
			args: args{
				format: "something went wrong: %s (%d)",
				args: []interface{}{
					"not found",
					404,
				},
			},
			wantLogs: []string{"something went wrong: not found (404)\n"},
		},
	}
	variants := map[string]fields{
		", with abort, not failed":      {abort: true, failed: 0},
		", with abort, failed once":     {abort: true, failed: 1},
		", with abort, failed twice":    {abort: true, failed: 2},
		", without abort, not failed":   {abort: false, failed: 0},
		", without abort, failed once":  {abort: false, failed: 1},
		", without abort, failed twice": {abort: false, failed: 2},
	}
	for _, tt := range tests {
		for nameSuffix, flds := range variants {
			t.Run(tt.name+nameSuffix, func(t *testing.T) {
				mt := &T{
					abort:  flds.abort,
					failed: flds.failed,
				}
				halted := true

				runInGoroutine(func() {
					mt.Fatalf(tt.args.format, tt.args.args...)
					halted = false
				})

				assert.Equal(t, flds.failed+1, mt.failed)
				assert.Equal(t, true, mt.aborted)
				assert.Equal(t, flds.abort, halted)
				assert.Equal(t, tt.wantLogs, mt.output)
			})
		}
	}
}

func TestT_Log(t *testing.T) {
	type fields struct {
		failed int
	}
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []string
	}{
		{
			name:     "no args",
			args:     args{},
			wantLogs: []string{"\n"},
		},
		{
			name:     "one arg",
			args:     args{args: []interface{}{"hello world"}},
			wantLogs: []string{"hello world\n"},
		},
		{
			name: "many args",
			args: args{
				args: []interface{}{
					"hello world",
					"where's my car?",
					1024,
				},
			},
			wantLogs: []string{"hello world where's my car? 1024\n"},
		},
	}
	variants := map[string]fields{
		", not failed":   {failed: 0},
		", failed once":  {failed: 1},
		", failed twice": {failed: 2},
	}
	for _, tt := range tests {
		for nameSuffix, flds := range variants {
			t.Run(tt.name+nameSuffix, func(t *testing.T) {
				mt := &T{failed: flds.failed}

				mt.Log(tt.args.args...)

				assert.Equal(t, flds.failed, mt.failed)
				assert.Equal(t, tt.wantLogs, mt.output)
			})
		}
	}
}

func TestT_Logf(t *testing.T) {
	type fields struct {
		failed int
	}
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []string
	}{
		{
			name:     "no format or args",
			args:     args{},
			wantLogs: []string{"\n"},
		},
		{
			name:     "format with no args",
			args:     args{format: "something went wrong"},
			wantLogs: []string{"something went wrong\n"},
		},
		{
			name: "format with args",
			args: args{
				format: "something went wrong: %s (%d)",
				args: []interface{}{
					"not found",
					404,
				},
			},
			wantLogs: []string{"something went wrong: not found (404)\n"},
		},
	}
	variants := map[string]fields{
		", not failed":   {failed: 0},
		", failed once":  {failed: 1},
		", failed twice": {failed: 2},
	}
	for _, tt := range tests {
		for nameSuffix, flds := range variants {
			t.Run(tt.name+nameSuffix, func(t *testing.T) {
				mt := &T{failed: flds.failed}

				mt.Logf(tt.args.format, tt.args.args...)

				assert.Equal(t, flds.failed, mt.failed)
				assert.Equal(t, tt.wantLogs, mt.output)
			})
		}
	}
}

func TestT_Parallel(t *testing.T) {
	type fields struct {
		parallel bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "not parallel",
			fields: fields{parallel: false},
			want:   true,
		},
		{
			name:   "already parallel",
			fields: fields{parallel: true},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{parallel: tt.fields.parallel}

			mt.Parallel()

			assert.Equal(t, tt.want, mt.parallel)
		})
	}
}

func TestT_Paralleled(t *testing.T) {
	type fields struct {
		parallel bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "not paralleled",
			fields: fields{parallel: false},
			want:   false,
		},
		{
			name:   "paralleled",
			fields: fields{parallel: true},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{parallel: tt.fields.parallel}

			got := mt.Paralleled()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_Skip(t *testing.T) {
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []string
	}{
		{
			name:     "no args",
			args:     args{},
			wantLogs: []string{"\n"},
		},
		{
			name:     "one arg",
			args:     args{args: []interface{}{"hello world"}},
			wantLogs: []string{"hello world\n"},
		},
		{
			name: "many args",
			args: args{
				args: []interface{}{
					"hello world",
					"where's my car?",
					1024,
				},
			},
			wantLogs: []string{"hello world where's my car? 1024\n"},
		},
	}
	variants := map[bool]string{
		true:  ", with abort",
		false: ", without abort",
	}
	for _, tt := range tests {
		for abort, nameSuffix := range variants {
			t.Run(tt.name+nameSuffix, func(t *testing.T) {
				mt := &T{abort: abort}
				halted := true

				runInGoroutine(func() {
					mt.Skip(tt.args.args...)
					halted = false
				})

				assert.True(t, mt.skipped)
				assert.True(t, mt.aborted)
				assert.Equal(t, abort, halted)
				assert.Equal(t, tt.wantLogs, mt.output)
			})
		}
	}
}

func TestT_Skipf(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantLogs []string
	}{
		{
			name:     "no format or args",
			args:     args{},
			wantLogs: []string{"\n"},
		},
		{
			name:     "format with no args",
			args:     args{format: "something went wrong"},
			wantLogs: []string{"something went wrong\n"},
		},
		{
			name: "format with args",
			args: args{
				format: "something went wrong: %s (%d)",
				args: []interface{}{
					"not found",
					404,
				},
			},
			wantLogs: []string{"something went wrong: not found (404)\n"},
		},
	}
	variants := map[bool]string{
		true:  ", with abort",
		false: ", without abort",
	}
	for _, tt := range tests {
		for abort, nameSuffix := range variants {
			t.Run(tt.name+nameSuffix, func(t *testing.T) {
				mt := &T{abort: abort}
				halted := true

				runInGoroutine(func() {
					mt.Skipf(tt.args.format, tt.args.args...)
					halted = false
				})

				assert.True(t, mt.skipped)
				assert.True(t, mt.aborted)
				assert.Equal(t, abort, halted)
				assert.Equal(t, tt.wantLogs, mt.output)
			})
		}
	}
}

func TestT_SkipNow(t *testing.T) {
	type fields struct {
		abort bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "with abort",
			fields: fields{abort: true},
		},
		{
			name:   "without abort",
			fields: fields{abort: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{abort: tt.fields.abort}
			halted := true

			runInGoroutine(func() {
				mt.SkipNow()
				halted = false
			})

			assert.True(t, mt.skipped)
			assert.True(t, mt.aborted)
			assert.Equal(t, tt.fields.abort, halted)
			assert.Empty(t, mt.output)
		})
	}
}

func TestT_Skipped(t *testing.T) {
	type fields struct {
		skipped bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "not skipped",
			fields: fields{skipped: false},
			want:   false,
		},
		{
			name:   "skipped",
			fields: fields{skipped: true},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{skipped: tt.fields.skipped}

			got := mt.Skipped()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_Helper(t *testing.T) {
	helper1 := func(t testing.TB) {
		t.Helper()
	}
	helper2 := func(t testing.TB) {
		t.Helper()
	}
	helper3 := func(t testing.TB) {
		t.Helper()
		helper1(t)
	}

	mt1 := &T{}
	helper3(mt1)

	mt2 := &T{}
	helper2(mt2)

	assert.Equal(t,
		[]string{
			"github.com/jimeh/go-mocktesting.TestT_Helper.func3",
			"github.com/jimeh/go-mocktesting.TestT_Helper.func1",
		},
		mt1.helpers,
	)
	assert.Equal(t,
		[]string{
			"github.com/jimeh/go-mocktesting.TestT_Helper.func2",
		},
		mt2.helpers,
	)
}

func TestT_Cleanup(t *testing.T) {
	cleanup1 := func() {}
	cleanup2 := func() {}
	cleanup3 := func() {}

	mt1 := &T{}
	mt1.Cleanup(cleanup3)
	mt1.Cleanup(cleanup1)

	mt1CleanupNames := make([]string, 0, len(mt1.cleanups))
	for _, f := range mt1.cleanups {
		p := reflect.ValueOf(f).Pointer()
		mt1CleanupNames = append(mt1CleanupNames, runtime.FuncForPC(p).Name())
	}

	mt2 := &T{}
	mt2.Cleanup(cleanup2)

	mt2CleanupNames := make([]string, 0, len(mt2.cleanups))
	for _, f := range mt2.cleanups {
		p := reflect.ValueOf(f).Pointer()
		mt2CleanupNames = append(mt2CleanupNames, runtime.FuncForPC(p).Name())
	}

	assert.Equal(t,
		[]string{
			"github.com/jimeh/go-mocktesting.TestT_Cleanup.func3",
			"github.com/jimeh/go-mocktesting.TestT_Cleanup.func1",
		},
		mt1CleanupNames,
	)
	assert.Equal(t,
		[]string{
			"github.com/jimeh/go-mocktesting.TestT_Cleanup.func2",
		},
		mt2CleanupNames,
	)
}

func TestT_TempDir(t *testing.T) {
	customTempDir := t.TempDir()
	assert.DirExists(t, customTempDir)

	type fields struct {
		baseTempdir   string
		testingT      testing.TB
		mkdirTempFunc func(string, string) (string, error)
	}
	tests := []struct {
		name         string
		calls        int
		fields       fields
		wantPrefix   string
		wantExists   bool
		wantPanic    interface{}
		wantTestingT *T
	}{
		{
			name:       "not called",
			calls:      0,
			wantExists: false,
		},
		{
			name:       "called once",
			calls:      1,
			wantPrefix: filepath.Join(os.TempDir(), "go-mocktesting"),
			wantExists: true,
		},
		{
			name:       "called twice",
			calls:      2,
			wantPrefix: filepath.Join(os.TempDir(), "go-mocktesting"),
			wantExists: true,
		},
		{
			name:       "called three times",
			calls:      3,
			wantPrefix: filepath.Join(os.TempDir(), "go-mocktesting"),
			wantExists: true,
		},
		{
			name:       "custom base tempdir",
			calls:      1,
			fields:     fields{baseTempdir: customTempDir},
			wantPrefix: filepath.Join(customTempDir, "go-mocktesting"),
			wantExists: true,
		},
		{
			name:  "directory creation fails",
			calls: 1,
			fields: fields{
				mkdirTempFunc: func(_ string, _ string) (string, error) {
					return "", errors.New("can't create dir")
				},
			},
			wantPanic: fmt.Errorf(
				"mocktesting: %w",
				fmt.Errorf(
					"TempDir() failed to create directory: %w",
					errors.New("can't create dir"),
				),
			),
		},
		{
			name:  "directory creation fails with testingT assigned",
			calls: 1,
			fields: fields{
				testingT: &T{name: "real", abort: true},
				mkdirTempFunc: func(_ string, _ string) (string, error) {
					return "", errors.New("can't create dir")
				},
			},
			wantTestingT: &T{
				name:    "real",
				abort:   true,
				failed:  1,
				aborted: true,
				output: []string{
					"mocktesting: TempDir() failed to create directory: " +
						"can't create dir\n",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{
				baseTempdir:   tt.fields.baseTempdir,
				testingT:      tt.fields.testingT,
				mkdirTempFunc: tt.fields.mkdirTempFunc,
			}

			var dirs []string
			for i := 0; i < tt.calls; i++ {
				f := func() (dir string, p interface{}) {
					defer func() { p = recover() }()
					dir = mt.TempDir()

					return
				}

				var dir string
				var p interface{}
				runInGoroutine(func() {
					dir, p = f()
				})

				if dir != "" {
					t.Cleanup(func() { os.Remove(dir) })
					dirs = append(dirs, dir)
				}

				assert.Equal(t, tt.wantPanic, p)
			}

			assert.Equal(t, dirs, mt.tempdirs)
			if tt.calls > 1 {
				assert.Len(t, stringsUniq(dirs), tt.calls,
					"returned temporary directories are not unique",
				)
			}
			for _, dir := range dirs {
				assert.Truef(t, strings.HasPrefix(dir, tt.wantPrefix),
					"temporary directory %s does not start with %s",
					dir, tt.wantPrefix,
				)
				if tt.wantExists {
					assert.DirExists(t, dir)
				}
			}

			if tt.wantTestingT != nil {
				assert.Equal(t, tt.wantTestingT, mt.testingT)
			}
		})
	}
}

func TestT_Run(t *testing.T) {
	cleanup1 := func() {}
	cleanup2 := func() {}
	cleanup3 := func() {}

	helper1 := func(t testing.TB) { t.Helper() }
	helper2 := func(t testing.TB) { t.Helper() }
	helper3 := func(t testing.TB) { t.Helper(); helper1(t) }

	customTempDir, err := ioutil.TempDir(os.TempDir(), t.Name()+"*")
	require.NoError(t, err)

	type fields struct {
		name        string
		abort       bool
		baseTempdir string
		testingT    testing.TB
		deadline    time.Time
		timeout     bool
	}
	type args struct {
		f func(testing.TB)
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *T
	}{
		{
			name: "does nothing",
			fields: fields{
				name:        "nothing",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					_ = fmt.Sprintf("nothing %s", t.Name())
				},
			},
			want: &T{
				name:        "nothing",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
		},
		{
			name: "fails",
			fields: fields{
				name:        "fails",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					t.Log("before Fail")
					t.Fail()
					t.Log("after Fail")
				},
			},
			want: &T{
				name:        "fails",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				failed:      1,
				output:      []string{"before Fail\n", "after Fail\n"},
			},
		},
		{
			name: "fails and halts",
			fields: fields{
				name:        "fails_and_halts",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					t.Log("before Fail")
					t.Fail()
					t.Log("after Fail")
					t.FailNow()
					t.Log("after FailNow")
				},
			},
			want: &T{
				name:        "fails_and_halts",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				aborted:     true,
				failed:      2,
				output:      []string{"before Fail\n", "after Fail\n"},
			},
		},
		{
			name: "skips",
			fields: fields{
				name:        "skips",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					t.Log("before Skip")
					t.Skip("skipping because reasons")
					t.Log("after Skip")
				},
			},
			want: &T{
				name:     "skips",
				abort:    true,
				deadline: time.Now().Add(10 * time.Minute),
				timeout:  true,
				skipped:  true,
				aborted:  true,
				output: []string{
					"before Skip\n",
					"skipping because reasons\n",
				},
				baseTempdir: os.TempDir(),
			},
		},
		{
			name: "fails and skips",
			fields: fields{
				name:        "fails_and_skips",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					t.Log("before Fail")
					t.Error("oops")
					t.Log("before Skip")
					t.Skip("skipping because reasons")
					t.Log("after Skip")
				},
			},
			want: &T{
				name:     "fails_and_skips",
				abort:    true,
				deadline: time.Now().Add(10 * time.Minute),
				timeout:  true,
				skipped:  true,
				failed:   1,
				aborted:  true,
				output: []string{
					"before Fail\n",
					"oops\n",
					"before Skip\n",
					"skipping because reasons\n",
				},
				baseTempdir: os.TempDir(),
			},
		},
		{
			name: "parallel",
			fields: fields{
				name:        "parallel",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					mt, _ := t.(*T)
					mt.Parallel()
				},
			},
			want: &T{
				name:        "parallel",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				parallel:    true,
			},
		},
		{
			name: "helpers",
			fields: fields{
				name:        "helpers",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					helper2(t)
					helper3(t)
				},
			},
			want: &T{
				name:        "helpers",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				helpers: []string{
					"github.com/jimeh/go-mocktesting.TestT_Run.func5",
					"github.com/jimeh/go-mocktesting.TestT_Run.func6",
					"github.com/jimeh/go-mocktesting.TestT_Run.func4",
				},
			},
		},
		{
			name: "cleanups",
			fields: fields{
				name:        "cleanups",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					t.Cleanup(cleanup3)
					t.Cleanup(cleanup1)
					t.Cleanup(cleanup2)
				},
			},
			want: &T{
				name:        "cleanups",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				cleanups:    []func(){cleanup3, cleanup1, cleanup2},
			},
		},
		{
			name: "subtests with no failures",
			fields: fields{
				name:        "subtests_no_failures",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					mt := t.(*T)

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from first sub-test")
					})

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from second sub-test")
					})

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from third sub-test")
					})

					mt.Run("hello, world", func(t testing.TB) {
						t.Log("from fourth sub-test")
					})
				},
			},
			want: &T{
				name:        "subtests_no_failures",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				failed:      0,
				subtests: []*T{
					{
						name:        "subtests_no_failures/foo_bar",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from first sub-test\n"},
					},
					{
						name:        "subtests_no_failures/foo_bar#01",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from second sub-test\n"},
					},
					{
						name:        "subtests_no_failures/foo_bar#02",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from third sub-test\n"},
					},
					{
						name:        "subtests_no_failures/hello,_world",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from fourth sub-test\n"},
					},
				},
				subtestNames: map[string]bool{
					"foo_bar":      true,
					"foo_bar#01":   true,
					"foo_bar#02":   true,
					"hello,_world": true,
				},
			},
		},
		{
			name: "subtests with failures",
			fields: fields{
				name:        "subtests_fail",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					mt := t.(*T)

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from first sub-test")
						t.Fail()
						t.Log("after failure")
					})

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from second sub-test")
					})

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from third sub-test")
						t.FailNow()
						t.Log("after failure")
					})

					mt.Run("hello, world", func(t testing.TB) {
						t.Log("from fourth sub-test")
					})
				},
			},
			want: &T{
				name:        "subtests_fail",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				failed:      2,
				subtests: []*T{
					{
						name:        "subtests_fail/foo_bar",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						failed:      1,
						output: []string{
							"from first sub-test\n",
							"after failure\n",
						},
					},
					{
						name:        "subtests_fail/foo_bar#01",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from second sub-test\n"},
					},
					{
						name:        "subtests_fail/foo_bar#02",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						failed:      1,
						aborted:     true,
						output:      []string{"from third sub-test\n"},
					},
					{
						name:        "subtests_fail/hello,_world",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from fourth sub-test\n"},
					},
				},
				subtestNames: map[string]bool{
					"foo_bar":      true,
					"foo_bar#01":   true,
					"foo_bar#02":   true,
					"hello,_world": true,
				},
			},
		},
		{
			name: "subtests inherit abort value",
			fields: fields{
				name:        "subtests_inherit",
				abort:       false,
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				baseTempdir: os.TempDir(),
			},
			args: args{
				f: func(t testing.TB) {
					mt := t.(*T)

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from first sub-test")
					})

					mt.Run("hello world", func(t testing.TB) {
						t.Log("from second sub-test")
					})
				},
			},
			want: &T{
				name:        "subtests_inherit",
				abort:       false,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				failed:      0,
				subtests: []*T{
					{
						name:        "subtests_inherit/foo_bar",
						abort:       false,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						failed:      0,
						output:      []string{"from first sub-test\n"},
					},
					{
						name:        "subtests_inherit/hello_world",
						abort:       false,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from second sub-test\n"},
					},
				},
				subtestNames: map[string]bool{
					"foo_bar":     true,
					"hello_world": true,
				},
			},
		},
		{
			name: "subtests inherit baseTempdir value",
			fields: fields{
				name:        "subtests_inherit",
				abort:       true,
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				baseTempdir: customTempDir,
			},
			args: args{
				f: func(t testing.TB) {
					mt := t.(*T)

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from first sub-test")
					})

					mt.Run("hello world", func(t testing.TB) {
						t.Log("from second sub-test")
					})
				},
			},
			want: &T{
				name:        "subtests_inherit",
				abort:       true,
				baseTempdir: customTempDir,
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				failed:      0,
				subtests: []*T{
					{
						name:        "subtests_inherit/foo_bar",
						abort:       true,
						baseTempdir: customTempDir,
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						failed:      0,
						output:      []string{"from first sub-test\n"},
					},
					{
						name:        "subtests_inherit/hello_world",
						abort:       true,
						baseTempdir: customTempDir,
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from second sub-test\n"},
					},
				},
				subtestNames: map[string]bool{
					"foo_bar":     true,
					"hello_world": true,
				},
			},
		},
		{
			name: "subtests inherit testingT value",
			fields: fields{
				name:        "subtests_inherit",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				testingT:    &T{name: "my custom testingT"},
			},
			args: args{
				f: func(t testing.TB) {
					mt := t.(*T)

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from first sub-test")
					})

					mt.Run("hello world", func(t testing.TB) {
						t.Log("from second sub-test")
					})
				},
			},
			want: &T{
				name:        "subtests_inherit",
				abort:       true,
				baseTempdir: os.TempDir(),
				testingT:    &T{name: "my custom testingT"},
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				failed:      0,
				subtests: []*T{
					{
						name:        "subtests_inherit/foo_bar",
						abort:       true,
						baseTempdir: os.TempDir(),
						testingT:    &T{name: "my custom testingT"},
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						failed:      0,
						output:      []string{"from first sub-test\n"},
					},
					{
						name:        "subtests_inherit/hello_world",
						abort:       true,
						baseTempdir: os.TempDir(),
						testingT:    &T{name: "my custom testingT"},
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     true,
						output:      []string{"from second sub-test\n"},
					},
				},
				subtestNames: map[string]bool{
					"foo_bar":     true,
					"hello_world": true,
				},
			},
		},
		{
			name: "subtests inherit deadline value",
			fields: fields{
				name:        "subtests_inherit",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(4 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					mt := t.(*T)

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from first sub-test")
					})

					mt.Run("hello world", func(t testing.TB) {
						t.Log("from second sub-test")
					})
				},
			},
			want: &T{
				name:        "subtests_inherit",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(4 * time.Minute),
				timeout:     true,
				failed:      0,
				subtests: []*T{
					{
						name:        "subtests_inherit/foo_bar",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(4 * time.Minute),
						timeout:     true,
						failed:      0,
						output:      []string{"from first sub-test\n"},
					},
					{
						name:        "subtests_inherit/hello_world",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(4 * time.Minute),
						timeout:     true,
						output:      []string{"from second sub-test\n"},
					},
				},
				subtestNames: map[string]bool{
					"foo_bar":     true,
					"hello_world": true,
				},
			},
		},
		{
			name: "subtests inherit timeout value",
			fields: fields{
				name:        "subtests_inherit",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     false,
			},
			args: args{
				f: func(t testing.TB) {
					mt := t.(*T)

					mt.Run("foo bar", func(t testing.TB) {
						t.Log("from first sub-test")
					})

					mt.Run("hello world", func(t testing.TB) {
						t.Log("from second sub-test")
					})
				},
			},
			want: &T{
				name:        "subtests_inherit",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     false,
				failed:      0,
				subtests: []*T{
					{
						name:        "subtests_inherit/foo_bar",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     false,
						failed:      0,
						output:      []string{"from first sub-test\n"},
					},
					{
						name:        "subtests_inherit/hello_world",
						abort:       true,
						baseTempdir: os.TempDir(),
						deadline:    time.Now().Add(10 * time.Minute),
						timeout:     false,
						output:      []string{"from second sub-test\n"},
					},
				},
				subtestNames: map[string]bool{
					"foo_bar":     true,
					"hello_world": true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{
				name:        tt.fields.name,
				abort:       tt.fields.abort,
				baseTempdir: tt.fields.baseTempdir,
				testingT:    tt.fields.testingT,
				deadline:    tt.fields.deadline,
				timeout:     tt.fields.timeout,
			}

			runInGoroutine(func() {
				tt.args.f(mt)
			})

			assertEqualMocktestingT(t, tt.want, mt)
		})
	}
}

func TestT_Output(t *testing.T) {
	type fields struct {
		output []string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "nil",
			fields: fields{},
			want:   nil,
		},
		{
			name:   "empty",
			fields: fields{output: []string{}},
			want:   []string{},
		},
		{
			name:   "one item",
			fields: fields{output: []string{"oops: not found\n"}},
			want:   []string{"oops: not found\n"},
		},
		{
			name:   "multiple items",
			fields: fields{output: []string{"oops: not found\n", "bye\n"}},
			want:   []string{"oops: not found\n", "bye\n"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{output: tt.fields.output}

			got := mt.Output()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_CleanupFuncs(t *testing.T) {
	cleanup1 := func() {}
	cleanup2 := func() {}
	cleanup3 := func() {}

	type fields struct {
		cleanups []func()
	}

	tests := []struct {
		name   string
		fields fields
		want   []func()
	}{
		{
			name: "nil",
			want: nil,
		},
		{
			name:   "empty",
			fields: fields{cleanups: []func(){}},
			want:   []func(){},
		},
		{
			name:   "one func",
			fields: fields{cleanups: []func(){cleanup1}},
			want:   []func(){cleanup1},
		},
		{
			name:   "many funcs",
			fields: fields{cleanups: []func(){cleanup3, cleanup1, cleanup2}},
			want:   []func(){cleanup3, cleanup1, cleanup2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{cleanups: tt.fields.cleanups}

			got := mt.CleanupFuncs()

			switch {
			case tt.want == nil:
				assert.Nil(t, got)
			case len(tt.want) == 0:
				assert.NotNil(t, got)
				assert.Len(t, got, 0)
			default:
				var wantFuncs []string
				for _, f := range tt.want {
					p := reflect.ValueOf(f).Pointer()
					wantFuncs = append(wantFuncs, runtime.FuncForPC(p).Name())
				}
				var gotFuncs []string
				for _, f := range got {
					p := reflect.ValueOf(f).Pointer()
					gotFuncs = append(gotFuncs, runtime.FuncForPC(p).Name())
				}

				assert.Equal(t, wantFuncs, gotFuncs)
			}
		})
	}
}

func TestT_CleanupNames(t *testing.T) {
	cleanup1 := func() {}
	cleanup2 := func() {}
	cleanup3 := func() {}

	type fields struct {
		cleanups []func()
	}

	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "nil",
			want: []string{},
		},
		{
			name:   "empty",
			fields: fields{cleanups: []func(){}},
			want:   []string{},
		},
		{
			name:   "one func",
			fields: fields{cleanups: []func(){cleanup1}},
			want: []string{
				"github.com/jimeh/go-mocktesting.TestT_CleanupNames.func1",
			},
		},
		{
			name:   "many funcs",
			fields: fields{cleanups: []func(){cleanup3, cleanup1, cleanup2}},
			want: []string{
				"github.com/jimeh/go-mocktesting.TestT_CleanupNames.func3",
				"github.com/jimeh/go-mocktesting.TestT_CleanupNames.func1",
				"github.com/jimeh/go-mocktesting.TestT_CleanupNames.func2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{cleanups: tt.fields.cleanups}

			got := mt.CleanupNames()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_HelperNames(t *testing.T) {
	type fields struct {
		helpers []string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "nil",
			fields: fields{},
			want:   nil,
		},
		{
			name:   "empty",
			fields: fields{helpers: []string{}},
			want:   []string{},
		},
		{
			name: "one helper",
			fields: fields{
				helpers: []string{
					"github.com/jimeh/go-mocktesting.TestT_HelperNames.func1",
				},
			},
			want: []string{
				"github.com/jimeh/go-mocktesting.TestT_HelperNames.func1",
			},
		},
		{
			name: "multiple helpers",
			fields: fields{
				helpers: []string{
					"github.com/jimeh/go-mocktesting.TestT_HelperNames.func2",
					"github.com/jimeh/go-mocktesting.TestT_HelperNames.func1",
					"github.com/jimeh/go-mocktesting.TestT_HelperNames.func2",
				},
			},
			want: []string{
				"github.com/jimeh/go-mocktesting.TestT_HelperNames.func2",
				"github.com/jimeh/go-mocktesting.TestT_HelperNames.func1",
				"github.com/jimeh/go-mocktesting.TestT_HelperNames.func2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{helpers: tt.fields.helpers}

			got := mt.HelperNames()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_Aborted(t *testing.T) {
	type fields struct {
		aborted bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "not aborted",
			fields: fields{aborted: false},
			want:   false,
		},
		{
			name:   "aborted",
			fields: fields{aborted: true},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{aborted: tt.fields.aborted}

			got := mt.Aborted()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_Subtests(t *testing.T) {
	type fields struct {
		subtests []*T
	}
	tests := []struct {
		name   string
		fields fields
		want   []*T
	}{
		{
			name:   "nil",
			fields: fields{},
			want:   []*T{},
		},
		{
			name:   "empty",
			fields: fields{subtests: []*T{}},
			want:   []*T{},
		},
		{
			name: "one subtest",
			fields: fields{
				subtests: []*T{
					{name: "foo_bar"},
				},
			},
			want: []*T{
				{name: "foo_bar"},
			},
		},
		{
			name: "multiple subtests",
			fields: fields{
				subtests: []*T{
					{name: "foo_bar"},
					{name: "hello"},
					{name: "world"},
				},
			},
			want: []*T{
				{name: "foo_bar"},
				{name: "hello"},
				{name: "world"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{subtests: tt.fields.subtests}

			got := mt.Subtests()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_TempDirs(t *testing.T) {
	type fields struct {
		tempdirs []string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name:   "nil",
			fields: fields{tempdirs: nil},
			want:   []string{},
		},
		{
			name:   "empty",
			fields: fields{tempdirs: []string{}},
			want:   []string{},
		},
		{
			name:   "one dir",
			fields: fields{tempdirs: []string{"/tmp/foo"}},
			want:   []string{"/tmp/foo"},
		},
		{
			name: "many dirs",
			fields: fields{
				tempdirs: []string{"/tmp/foo", "/tmp/foo", "/tmp/nope"},
			},
			want: []string{"/tmp/foo", "/tmp/foo", "/tmp/nope"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{tempdirs: tt.fields.tempdirs}

			got := mt.TempDirs()

			assert.Equal(t, tt.want, got)
		})
	}
}
