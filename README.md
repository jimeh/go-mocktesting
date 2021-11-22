<h1 align="center">
  go-mocktesting
</h1>

<p align="center">
  <strong>
    Mock *testing.T for the purpose of testing test helpers.
  </strong>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/jimeh/go-mocktesting"><img src="https://img.shields.io/badge/%E2%80%8B-reference-387b97.svg?logo=go&logoColor=white" alt="Go Reference"></a>
  <a href="https://github.com/jimeh/go-mocktesting/releases"><img src="https://img.shields.io/github/v/tag/jimeh/go-mocktesting?label=release" alt="GitHub tag (latest SemVer)"></a>
  <a href="https://github.com/jimeh/go-mocktesting/actions"><img src="https://img.shields.io/github/workflow/status/jimeh/go-mocktesting/CI.svg?logo=github" alt="Actions Status"></a>
  <a href="https://codeclimate.com/github/jimeh/go-mocktesting"><img src="https://img.shields.io/codeclimate/coverage/jimeh/go-mocktesting.svg?logo=code%20climate" alt="Coverage"></a>
  <a href="https://github.com/jimeh/go-mocktesting/issues"><img src="https://img.shields.io/github/issues-raw/jimeh/go-mocktesting.svg?style=flat&logo=github&logoColor=white" alt="GitHub issues"></a>
  <a href="https://github.com/jimeh/go-mocktesting/pulls"><img src="https://img.shields.io/github/issues-pr-raw/jimeh/go-mocktesting.svg?style=flat&logo=github&logoColor=white" alt="GitHub pull requests"></a>
  <a href="https://github.com/jimeh/go-mocktesting/blob/master/LICENSE"><img src="https://img.shields.io/github/license/jimeh/go-mocktesting.svg?style=flat" alt="License Status"></a>
</p>

## Import

```go
import "github.com/jimeh/go-mocktesting"
```

## Usage

```go
func ExampleT_Error() {
	assertTrue := func(t testing.TB, v bool) {
		if v != true {
			t.Error("expected false to be true")
		}
	}

	mt := mocktesting.NewT("TestMyBoolean1")
	assertTrue(mt, true)
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Aborted: %+v\n", mt.Aborted())

	mt = mocktesting.NewT("TestMyBoolean2")
	assertTrue(mt, false)
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Aborted: %+v\n", mt.Aborted())
	fmt.Printf("Output: %s\n", strings.Join(mt.Output(), ""))

	// Output:
	// Name: TestMyBoolean1
	// Failed: false
	// Aborted: false
	// Name: TestMyBoolean2
	// Failed: true
	// Aborted: false
	// Output: expected false to be true
}
```

```go
func ExampleT_Fatal() {
	requireTrue := func(t testing.TB, v bool) {
		if v != true {
			t.Fatal("expected false to be true")
		}
	}

	mt := mocktesting.NewT("TestMyBoolean1")
	fmt.Printf("Name: %s\n", mt.Name())
	requireTrue(mt, true)
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Aborted: %+v\n", mt.Aborted())

	mt = mocktesting.NewT("TestMyBoolean2")
	fmt.Printf("Name: %s\n", mt.Name())
	mocktesting.Go(func() {
		requireTrue(mt, false)
		fmt.Println("This is never executed.")
	})
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Aborted: %+v\n", mt.Aborted())
	fmt.Printf("Output:\n  - %s\n", strings.Join(mt.Output(), "\n  - "))

	// Output:
	// Name: TestMyBoolean1
	// Failed: false
	// Aborted: false
	// Name: TestMyBoolean2
	// Failed: true
	// Aborted: true
	// Output:
	//   - expected false to be true
}
```

```go
func ExampleT_Run() {
	requireTrue := func(t testing.TB, v bool) {
		if v != true {
			t.Fatal("expected false to be true")
		}
	}

	mt := mocktesting.NewT("TestMyBoolean")
	fmt.Printf("Name: %s\n", mt.Name())
	mt.Run("true", func(t testing.TB) {
		requireTrue(t, true)
	})
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Sub1-Name: %s\n", mt.Subtests()[0].Name())
	fmt.Printf("Sub1-Failed: %+v\n", mt.Subtests()[0].Failed())
	fmt.Printf("Sub1-Aborted: %+v\n", mt.Subtests()[0].Aborted())

	mt.Run("false", func(t testing.TB) {
		requireTrue(t, false)
		fmt.Println("This is never executed.")
	})
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Sub2-Name: %s\n", mt.Subtests()[1].Name())
	fmt.Printf("Sub2-Failed: %+v\n", mt.Subtests()[1].Failed())
	fmt.Printf("Sub2-Aborted: %+v\n", mt.Subtests()[1].Aborted())
	fmt.Printf("Sub2-Output:\n  - %s\n",
		strings.Join(mt.Subtests()[1].Output(), "\n  - "),
	)

	// Output:
	// Name: TestMyBoolean
	// Failed: false
	// Sub1-Name: TestMyBoolean/true
	// Sub1-Failed: false
	// Sub1-Aborted: false
	// Failed: true
	// Sub2-Name: TestMyBoolean/false
	// Sub2-Failed: true
	// Sub2-Aborted: true
	// Sub2-Output:
	//   - expected false to be true
}
```

## Documentation

Please see the
[Go Reference](https://pkg.go.dev/github.com/jimeh/go-mocktesting#section-documentation)
for documentation and examples.

## License

[MIT](https://github.com/jimeh/go-mocktesting/blob/main/LICENSE)
