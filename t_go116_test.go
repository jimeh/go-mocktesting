//go:build go1.16
// +build go1.16

package mocktesting

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestT_Setenv(t *testing.T) {
	type fields struct {
		env map[string]string
	}
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]string
	}{
		{
			name: "empty key and value",
			args: args{},
			want: map[string]string{},
		},
		{
			name: "empty key",
			args: args{value: "bar"},
			want: map[string]string{},
		},
		{
			name: "empty value",
			args: args{key: "foo"},
			want: map[string]string{"foo": ""},
		},
		{
			name: "key and value",
			args: args{key: "foo", value: "bar"},
			want: map[string]string{"foo": "bar"},
		},
		{
			name:   "add to existing",
			fields: fields{env: map[string]string{"hello": "world"}},
			args:   args{key: "foo", value: "bar"},
			want:   map[string]string{"hello": "world", "foo": "bar"},
		},
		{
			name:   "overwrite existing",
			fields: fields{env: map[string]string{"foo": "world"}},
			args:   args{key: "foo", value: "bar"},
			want:   map[string]string{"foo": "bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{env: tt.fields.env}

			mt.Setenv(tt.args.key, tt.args.value)

			assert.Equal(t, tt.want, mt.env)
		})
	}
}

func TestT_Getenv(t *testing.T) {
	type fields struct {
		env map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{
			name: "nil env",
			want: map[string]string{},
		},
		{
			name:   "empty env",
			fields: fields{env: map[string]string{}},
			want:   map[string]string{},
		},
		{
			name: "env",
			fields: fields{
				env: map[string]string{
					"hello": "world",
					"foo":   "bar",
				},
			},
			want: map[string]string{
				"hello": "world",
				"foo":   "bar",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt := &T{env: tt.fields.env}

			got := mt.Getenv()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestT_Run_Go116(t *testing.T) {
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
			name: "set environment variables",
			fields: fields{
				name:        "set_environment_variables",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
			},
			args: args{
				f: func(t testing.TB) {
					// Type assert to *TB for compatibility with Go 1.16 and
					// earlier.
					mt := t.(*T)
					mt.Setenv("GO_ENV", "test")
					mt.Setenv("FOO", "bar")
				},
			},
			want: &T{
				name:        "set_environment_variables",
				abort:       true,
				baseTempdir: os.TempDir(),
				deadline:    time.Now().Add(10 * time.Minute),
				timeout:     true,
				env: map[string]string{
					"GO_ENV": "test",
					"FOO":    "bar",
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
