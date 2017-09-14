package retry

import (
	"errors"
	"time"
)

type StopReason int16

type Function func(state *State) (interface{}, error)

const (
	Success             StopReason = iota
	NonRetryableError
	MaxAttemptsExceeded
	MaxDurationExceeded
)

type State struct {
	StartTimestamp time.Time
	Attempts       int
	Reason         StopReason
	Result         interface{}
	Err            error
}

type Config struct {
	MaxAttempts int
	MaxDuration time.Duration        // Retry time limit.
	RetryError  func(err error) bool // Returns true if function can be retried.
	// backoffConfig SomeBackoffInterfaceUnionThing?
}

func ShouldRetry(err error) bool {
	return err != nil
}

func (r *Config) Execute(f Function) (interface{}, error, *State) {
	if r.RetryError == nil {
		r.RetryError = ShouldRetry
	}

	ctx := State{
		StartTimestamp: time.Now(),
		Attempts:       0,
		Err:            nil,
		Reason:         NonRetryableError,
	}

	for ctx.Attempts = 0; true ; ctx.Attempts++ {
		rv, e := f(&ctx)	// Call the function.
		if e != nil {
			ctx.Err = e
		}

		if !r.RetryError(e) {	// Exit if there is an error or other condition.
			ctx.Result = rv
			return rv, e, &ctx
		}

		backoffDuration := r.computeBackoff(&ctx)
		if backoffDuration.Nanoseconds() > 10 {
			time.Sleep(backoffDuration)
		}
	}
	panic("How did you get here?")
}

func (r *Config) computeBackoff(ctx *State) time.Duration {
	// TODO: Fixed and exponential backoff.
	return time.Duration(100 * time.Millisecond)
	// exponential =
}

func (r *Config) shouldRetry(ctx *State, err error) bool {
	if !r.RetryError(ctx.Err) {
		ctx.stop(NonRetryableError, "")
		return false
	}

	if ctx.Attempts > r.MaxAttempts {
		ctx.stop(MaxAttemptsExceeded, "max attempts exceeded")
		return false
	}

	now := time.Now()
	elapsed := now.Sub(ctx.StartTimestamp)

	if elapsed.Nanoseconds() > r.MaxDuration.Nanoseconds() {
		ctx.stop(MaxDurationExceeded, "max duration exceeded")
		return false
	}

	return true
}

func (ctx *State) stop(reason StopReason, msg string) {
	if ctx.Err == nil && msg != "" {
		ctx.Err = errors.New(msg)
	}
	ctx.Reason = reason
}
