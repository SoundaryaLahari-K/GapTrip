package job

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound               = errors.New("job not found")
	ErrDuplicateID            = errors.New("job ID already exists")
	ErrStaleExecution         = errors.New("job execution is stale")
	ErrInvalidTransition      = errors.New("invalid job state transition")
	ErrRetryAttemptsExhausted = errors.New("job retry attempts exhausted")
)

// ClaimOutcome explains why a delivery did or did not acquire a job.
type ClaimOutcome string

const (
	ClaimAcquired    ClaimOutcome = "acquired"
	ClaimUnavailable ClaimOutcome = "unavailable"
	ClaimTerminal    ClaimOutcome = "terminal"
)

// Execution is an opaque, single-attempt capability returned by Claim. Its
// token is intentionally unexported so callers can pass it only to the
// repository methods that complete that same attempt.
type Execution struct {
	Job Job

	attempt    int
	claimToken string
}

// Repository persists jobs and owns their atomic lifecycle transitions.
type Repository interface {
	Create(context.Context, Job) error
	GetByID(context.Context, string) (Job, error)
	Claim(context.Context, string, time.Time, time.Duration) (Execution, ClaimOutcome, error)
	Succeed(context.Context, Execution, time.Time) error
	Reschedule(context.Context, Execution, time.Time, time.Time, Error) error
	Fail(context.Context, Execution, time.Time, Error) error
}
