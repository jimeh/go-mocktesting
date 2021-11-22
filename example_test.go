package mocktesting_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jimeh/go-mocktesting"
)

func Example_basic() {
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

func Example_fatal() {
	requireTrue := func(t testing.TB, v bool) {
		if v != true {
			t.Fatal("expected false to be true")
		}
	}

	mt := mocktesting.NewT("TestMyBoolean1")
	requireTrue(mt, true)
	fmt.Printf("Name: %s\n", mt.Name())
	fmt.Printf("Failed: %+v\n", mt.Failed())
	fmt.Printf("Aborted: %+v\n", mt.Aborted())

	mt = mocktesting.NewT("TestMyBoolean2")
	mocktesting.Go(func() {
		requireTrue(mt, false)
		fmt.Println("This is never executed.")
	})
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
	// Aborted: true
	// Output: expected false to be true
}

func Example_subtests() {
	requireTrue := func(t testing.TB, v bool) {
		if v != true {
			t.Fatal("expected false to be true")
		}
	}

	mt := mocktesting.NewT("TestMyBoolean")
	mt.Run("true", func(t testing.TB) {
		requireTrue(t, true)
	})
	fmt.Printf("Name: %s\n", mt.Name())
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
	fmt.Printf("Sub2-Output: %s\n", strings.Join(mt.Subtests()[1].Output(), ""))

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
	// Sub2-Output: expected false to be true
}

func Example_subtests_in_subtests() {
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
		"Sub1-Sub3-Output: %s\n", strings.TrimSpace(
			strings.Join(mt.Subtests()[0].Subtests()[1].Output(), ""),
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
		"Sub1-Sub3-Output: %s\n", strings.TrimSpace(
			strings.Join(mt.Subtests()[0].Subtests()[2].Output(), ""),
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
	// Sub1-Sub3-Output: expected 5 to be greater than 5
	// Sub1-Sub1-Name: TestMyBoolean/positive/less_than
	// Sub1-Sub3-Failed: true
	// Sub1-Sub3-Aborted: false
	// Sub1-Sub3-Output: expected 4 to be greater than 5
}
