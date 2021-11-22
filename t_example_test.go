package mocktesting_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jimeh/go-mocktesting"
)

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

func ExampleT_Errorf() {
	assertGreaterThan := func(t testing.TB, got int, min int) {
		if got <= min {
			t.Errorf("expected %d to be greater than %d", got, min)
		}
	}

	mt := mocktesting.NewT("TestMyBoolean1")
	assertGreaterThan(mt, 6, 5)
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Aborted: %+v\n", mt.Aborted())

	mt = mocktesting.NewT("TestMyBoolean2")
	assertGreaterThan(mt, 4, 5)
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Aborted: %+v\n", mt.Aborted())
	fmt.Printf("Output:\n  - %s\n", strings.Join(mt.Output(), "\n  - "))

	// Output:
	// Name: TestMyBoolean1
	// Failed: false
	// Aborted: false
	// Name: TestMyBoolean2
	// Failed: true
	// Aborted: false
	// Output:
	//   - expected 4 to be greater than 5
}

func ExampleT_Fail() {
	assertTrue := func(t testing.TB, v bool) {
		if v != true {
			t.Fail()
		}
	}

	mt := mocktesting.NewT("TestMyBoolean1")
	fmt.Printf("Name: %s\n", mt.Name())
	assertTrue(mt, false)
	fmt.Printf("Failed: %+v\n", mt.Failed())

	// Output:
	// Name: TestMyBoolean1
	// Failed: true
}

func ExampleT_FailNow() {
	requireTrue := func(t testing.TB, v bool) {
		if v != true {
			t.FailNow()
		}
	}

	mt := mocktesting.NewT("TestMyBoolean1")
	fmt.Printf("Name: %s\n", mt.Name())
	halted := true
	mocktesting.Go(func() {
		requireTrue(mt, false)
		halted = false
	})
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Halted: %+v\n", halted)

	// Output:
	// Name: TestMyBoolean1
	// Failed: true
	// Halted: true
}

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

func ExampleT_Fatalf() {
	requireGreaterThan := func(t testing.TB, got int, min int) {
		if got <= min {
			t.Fatalf("expected %d to be greater than %d", got, min)
		}
	}

	mt := mocktesting.NewT("TestMyGT1")
	fmt.Printf("Name: %s\n", mt.Name())
	halted := true
	mocktesting.Go(func() {
		requireGreaterThan(mt, 6, 5)
		halted = false
	})
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Halted: %+v\n", halted)

	mt = mocktesting.NewT("TestMyGT2")
	fmt.Printf("Name: %s\n", mt.Name())
	halted = true
	mocktesting.Go(func() {
		requireGreaterThan(mt, 4, 5)
		halted = false
	})
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Halted: %+v\n", halted)
	fmt.Printf("Output:\n  - %s\n", strings.Join(mt.Output(), "\n  - "))

	// Output:
	// Name: TestMyGT1
	// Failed: false
	// Halted: false
	// Name: TestMyGT2
	// Failed: true
	// Halted: true
	// Output:
	//   - expected 4 to be greater than 5
}

func ExampleT_Log() {
	logHello := func(t testing.TB) {
		t.Log("hello world")
	}

	mt := mocktesting.NewT("TestMyLog")
	fmt.Printf("Name: %s\n", mt.Name())
	logHello(mt)
	fmt.Printf("Output:\n  - %s\n", strings.Join(mt.Output(), "\n  - "))

	// Output:
	// Name: TestMyLog
	// Output:
	//   - hello world
}

func ExampleT_Logf() {
	logHello := func(t testing.TB, name string) {
		t.Logf("hello, %s", name)
	}

	mt := mocktesting.NewT("TestMyLogf")
	fmt.Printf("Name: %s\n", mt.Name())
	logHello(mt, "Abel")
	fmt.Printf("Output:\n  - %s\n", strings.Join(mt.Output(), "\n  - "))

	// Output:
	// Name: TestMyLogf
	// Output:
	//   - hello, Abel
}

func ExampleT_Parallel() {
	logHello := func(t testing.TB, name string) {
		mt, _ := t.(*mocktesting.T)
		mt.Parallel()
		mt.Logf("hello, %s", name)
	}

	mt := mocktesting.NewT("TestMyLogf")
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf("Parallel (before): %+v\n", mt.Paralleled())
	logHello(mt, "Abel")
	fmt.Printf("Parallel (after): %+v\n", mt.Paralleled())

	// Output:
	// Name: TestMyLogf
	// Parallel (before): false
	// Parallel (after): true
}

func ExampleT_Skip() {
	logHello := func(t testing.TB, name string) {
		if name == "" {
			t.Skip("no name given to say hello to")
		}
		t.Logf("hello, %s", name)
	}

	mt := mocktesting.NewT("TestMyLog1")
	fmt.Printf("Name: %s\n", mt.Name())
	halted := true
	mocktesting.Go(func() {
		logHello(mt, "Abel")
		halted = false
	})
	fmt.Printf("Skipped: %+v\n", mt.Skipped())
	fmt.Printf("Halted: %+v\n", halted)

	mt = mocktesting.NewT("TestMyLog2")
	fmt.Printf("Name: %s\n", mt.Name())
	halted = true
	mocktesting.Go(func() {
		logHello(mt, "")
		halted = false
	})
	fmt.Printf("Skipped: %+v\n", mt.Skipped())
	fmt.Printf("Halted: %+v\n", halted)
	fmt.Printf("Output:\n  - %s\n", strings.Join(mt.Output(), "\n  - "))

	// Output:
	// Name: TestMyLog1
	// Skipped: false
	// Halted: false
	// Name: TestMyLog2
	// Skipped: true
	// Halted: true
	// Output:
	//   - no name given to say hello to
}

func ExampleT_Skipf() {
	logHello := func(t testing.TB, name string) {
		if name != "Jane" {
			t.Skipf("I only say hello to Jane, you are %s", name)
		}
	}

	mt := mocktesting.NewT("TestMyLog1")
	fmt.Printf("Name: %s\n", mt.Name())
	halted := true
	mocktesting.Go(func() {
		logHello(mt, "Jane")
		halted = false
	})
	fmt.Printf("Skipped: %+v\n", mt.Skipped())
	fmt.Printf("Halted: %+v\n", halted)

	mt = mocktesting.NewT("TestMyLog2")
	fmt.Printf("Name: %s\n", mt.Name())
	halted = true
	mocktesting.Go(func() {
		logHello(mt, "John")
		halted = false
	})
	fmt.Printf("Skipped: %+v\n", mt.Skipped())
	fmt.Printf("Halted: %+v\n", halted)
	fmt.Printf("Output:\n  - %s\n", strings.Join(mt.Output(), "\n  - "))

	// Output:
	// Name: TestMyLog1
	// Skipped: false
	// Halted: false
	// Name: TestMyLog2
	// Skipped: true
	// Halted: true
	// Output:
	//   - I only say hello to Jane, you are John
}

func ExampleT_SkipNow() {
	logHello := func(t testing.TB, name string) {
		if name == "" {
			t.SkipNow()
		}
	}

	mt := mocktesting.NewT("TestMyLog1")
	fmt.Printf("Name: %s\n", mt.Name())
	halted := true
	mocktesting.Go(func() {
		logHello(mt, "Abel")
		halted = false
	})
	fmt.Printf("Skipped: %+v\n", mt.Skipped())
	fmt.Printf("Halted: %+v\n", halted)

	mt = mocktesting.NewT("TestMyLog2")
	fmt.Printf("Name: %s\n", mt.Name())
	halted = true
	mocktesting.Go(func() {
		logHello(mt, "")
		halted = false
	})
	fmt.Printf("Skipped: %+v\n", mt.Skipped())
	fmt.Printf("Halted: %+v\n", halted)

	// Output:
	// Name: TestMyLog1
	// Skipped: false
	// Halted: false
	// Name: TestMyLog2
	// Skipped: true
	// Halted: true
}

func ExampleT_Helper() {
	assertTrue := func(t testing.TB, v bool) {
		t.Helper()

		if v != true {
			t.Error("expected false to be true")
		}
	}

	mt := mocktesting.NewT("TestMyBoolean1")
	assertTrue(mt, true)
	assertTrue(mt, true)
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf("Helpers:\n  - %s\n", strings.Join(mt.HelperNames(), "\n  - "))

	// Output:
	// Name: TestMyBoolean1
	// Helpers:
	//   - github.com/jimeh/go-mocktesting_test.ExampleT_Helper.func1
	//   - github.com/jimeh/go-mocktesting_test.ExampleT_Helper.func1
}

func ExampleT_Cleanup() {
	cleanup1 := func() {
		fmt.Println("running cleanup1")
	}
	cleanup2 := func() {
		fmt.Println("running cleanup2")
	}

	mt := mocktesting.NewT("TestMyCleanup")
	mt.Cleanup(cleanup1)
	mt.Cleanup(cleanup2)
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf(
		"CleanupNames:\n  - %s\n", strings.Join(mt.CleanupNames(), "\n  - "),
	)
	mt.CleanupFuncs()[1]()
	mt.CleanupFuncs()[0]()

	// Output:
	// Name: TestMyCleanup
	// CleanupNames:
	//   - github.com/jimeh/go-mocktesting_test.ExampleT_Cleanup.func1
	//   - github.com/jimeh/go-mocktesting_test.ExampleT_Cleanup.func2
	// running cleanup2
	// running cleanup1
}

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

func ExampleT_Run_nested() {
	assertGreaterThan := func(t testing.TB, got int, min int) {
		if got <= min {
			t.Errorf("expected %d to be greater than %d", got, min)
		}
	}

	mt := mocktesting.NewT("TestMyBoolean")
	mt.Run("positive", func(t testing.TB) {
		subMT, _ := t.(*mocktesting.T)

		subMT.Run("greater than", func(t testing.TB) {
			assertGreaterThan(t, 5, 4)
		})
		subMT.Run("equal", func(t testing.TB) {
			assertGreaterThan(t, 5, 5)
		})
		subMT.Run("less than", func(t testing.TB) {
			assertGreaterThan(t, 4, 5)
		})
	})
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Sub1-Name: %s\n", mt.Subtests()[0].Name())
	fmt.Printf("Sub1-Failed: %+v\n", mt.Subtests()[0].Failed())
	fmt.Printf("Sub1-Aborted: %+v\n", mt.Subtests()[0].Aborted())
	fmt.Printf("Sub1-Sub1-Name: %s\n", mt.Subtests()[0].Subtests()[0].Name())
	fmt.Printf(
		"Sub1-Sub1-Failed: %+v\n", mt.Subtests()[0].Subtests()[0].Failed(),
	)
	fmt.Printf(
		"Sub1-Sub1-Aborted: %+v\n", mt.Subtests()[0].Subtests()[0].Aborted(),
	)
	fmt.Printf("Sub1-Sub1-Name: %s\n", mt.Subtests()[0].Subtests()[1].Name())
	fmt.Printf(
		"Sub1-Sub2-Failed: %+v\n", mt.Subtests()[0].Subtests()[1].Failed(),
	)
	fmt.Printf(
		"Sub1-Sub2-Aborted: %+v\n", mt.Subtests()[0].Subtests()[1].Aborted(),
	)
	fmt.Printf(
		"Sub1-Sub3-Output:\n  - %s\n", strings.TrimSpace(
			strings.Join(mt.Subtests()[0].Subtests()[1].Output(), "\n  - "),
		),
	)
	fmt.Printf("Sub1-Sub1-Name: %s\n", mt.Subtests()[0].Subtests()[2].Name())
	fmt.Printf(
		"Sub1-Sub3-Failed: %+v\n", mt.Subtests()[0].Subtests()[2].Failed(),
	)
	fmt.Printf(
		"Sub1-Sub3-Aborted: %+v\n", mt.Subtests()[0].Subtests()[2].Aborted(),
	)
	fmt.Printf(
		"Sub1-Sub3-Output:\n  - %s\n", strings.TrimSpace(
			strings.Join(mt.Subtests()[0].Subtests()[2].Output(), "\n  - "),
		),
	)

	// Output:
	// Name: TestMyBoolean
	// Failed: true
	// Sub1-Name: TestMyBoolean/positive
	// Sub1-Failed: true
	// Sub1-Aborted: false
	// Sub1-Sub1-Name: TestMyBoolean/positive/greater_than
	// Sub1-Sub1-Failed: false
	// Sub1-Sub1-Aborted: false
	// Sub1-Sub1-Name: TestMyBoolean/positive/equal
	// Sub1-Sub2-Failed: true
	// Sub1-Sub2-Aborted: false
	// Sub1-Sub3-Output:
	//   - expected 5 to be greater than 5
	// Sub1-Sub1-Name: TestMyBoolean/positive/less_than
	// Sub1-Sub3-Failed: true
	// Sub1-Sub3-Aborted: false
	// Sub1-Sub3-Output:
	//   - expected 4 to be greater than 5
}
