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
	lastBackoff    time.Duration
}

type BackoffStrategy interface {
	computeBackoff(state *State) time.Duration
}

type LinearBackoff struct {
	delay	time.Duration
}

func (l *LinearBackoff) computeBackoff(state *State) time.Duration {
	return l.delay // Same delay every time.
}

type ExponentialBackoff struct {
	initialDelay	time.Duration
	factor			float32
}

func ComputeExponentialBackoff(initial time.Duration, last time.Duration, factor float32) time.Duration {
	if last == 0 {
		return initial
	}
	nanos := float32(last.Nanoseconds()) * factor
	return time.Duration(int64(nanos))
}

func (e *ExponentialBackoff) computeBackoff(state *State) time.Duration {
	return ComputeExponentialBackoff(e.initialDelay, state.lastBackoff, e.factor)
}

type Config struct {
	MaxAttempts int
	MaxDuration time.Duration        // Retry time limit.
	RetryError  func(err error) bool // Returns true if function can be retried.
	Backoff		BackoffStrategy		 // Linear or exponential backoff.
}

func ShouldRetry(err error) bool {
	return err != nil
}

func (c *Config) Execute(f Function) (interface{}, error, *State) {
	if c.RetryError == nil {
		c.RetryError = ShouldRetry
	}

	if c.Backoff == nil {
		c.Backoff = &LinearBackoff{ delay: time.Duration(100 * time.Millisecond)}
	}

	state := State {
		StartTimestamp: time.Now(),
		Attempts:       0,
		Err:            nil,
		Reason:         NonRetryableError,
	}

	for state.Attempts = 0; true ; state.Attempts++ {
		rv, e := f(&state) // Call the function.
		if e != nil {
			state.Err = e
		}

		if !c.RetryError(e) { // Exit if there is an error or other condition.
			state.Result = rv
			if e == nil {
				state.Reason = Success
			}
			return rv, e, &state
		}

		delay := c.Backoff.computeBackoff(&state)
		time.Sleep(delay)
		state.lastBackoff = delay
	}
	panic("How did you get here?")
}

func (c *Config) shouldRetry(state *State, err error) bool {
	if !c.RetryError(state.Err) {
		state.stop(NonRetryableError, "")
		return false
	}

	if state.Attempts > c.MaxAttempts {
		state.stop(MaxAttemptsExceeded, "max attempts exceeded")
		return false
	}

	now := time.Now()
	elapsed := now.Sub(state.StartTimestamp)

	if elapsed.Nanoseconds() > c.MaxDuration.Nanoseconds() {
		state.stop(MaxDurationExceeded, "max duration exceeded")
		return false
	}

	return true
}

func (s *State) stop(reason StopReason, msg string) {
	if s.Err == nil && msg != "" {
		s.Err = errors.New(msg)
	}
	s.Reason = reason
}
