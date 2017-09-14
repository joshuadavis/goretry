package retry

import (
	"testing"
	"errors"
	"fmt"
	"time"
)

type Foo struct {
	bar int
	baz string
}

func TestFunctionCall(t *testing.T) {
	fn := func(state *State) (val interface{}, err error) {
		if state.Attempts > 3 {
			return &Foo { 42, "blort" }, nil
		}
		e := errors.New("blort")
		return nil, e
	}

	r := Config{
		MaxAttempts: 10,
		Backoff: &ExponentialBackoff{
			initialDelay: time.Duration(100 * time.Millisecond),
			factor: 2.0,
			},
	}

	rv, e, state := r.Execute(fn)

	fmt.Printf("rv=%v state=%v\n", rv, state)

	if e != nil {
		t.Error("Expected nil error!")
	}

	if state.Reason != Success {
		t.Error("Expected Success!")
	}
}

func TestCalculateExponentialBackoff(t *testing.T) {
	initial := time.Duration(5)
	var last time.Duration
	for i := 0; i < 10 ; i++ {
		last = ComputeExponentialBackoff(initial, last, 2.0)
	}
	if last.Nanoseconds() != 2560 {
		t.Errorf("Expected 2560, got %v", last)
	}
}