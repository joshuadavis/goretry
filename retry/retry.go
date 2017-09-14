package retry

import (
	"errors"
	"time"
	"database/sql"
)

type StopReason int16

type Function func(ctx *Context) (interface{}, error)

const (
	Success             StopReason = iota
	NonRetryableError
	MaxAttemptsExceeded
	MaxDurationExceeded
)

type Context struct {
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
}

func ShouldRetry(err error) bool {
	return err != nil
}

func (r *Config) Execute(f Function) (interface{}, error, *Context) {
	if r.RetryError == nil {
		r.RetryError = ShouldRetry
	}

	ctx := Context{
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
}

func (r *Config) computeBackoff(ctx *Context) time.Duration {
	// TODO: Fixed and exponential backoff.
	return time.Duration(100 * time.Millisecond)
	// exponential =
}

func (r *Config) shouldRetry(ctx *Context, err error) bool {
	if !r.RetryError(ctx.Err) {
		ctx.stop(MaxAttemptsExceeded, nil)
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

func (ctx *Context) stop(reason StopReason, msg string) {
	if ctx.Err == nil {
		ctx.Err = errors.New(msg)
	}
	ctx.Reason = reason
}
