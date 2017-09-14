package retry

import (
	"testing"
	"errors"
	"fmt"
)

type Foo struct {
	bar int
	baz string
}

func TestFunctionCall(t *testing.T) {
	fn := func(ctx *State) (val interface{}, err error) {
		if ctx.Attempts > 3 {
			return &Foo { 42, "blort" }, nil
		}
		e := errors.New("blort")
		return nil, e
	}

	r := Config{
		MaxAttempts: 10,
	}

	rv, e, ctx := r.Execute(fn)

	if e != nil {
		return
		}


	fmt.Printf("rv=%v ctx=%v\n", rv, ctx)
}