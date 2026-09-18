// Package job contains the durable state machine for asynchronous work.
package job

import (
	"errors"
	"strings"
	"time"
)

const maxErrorMessageRunes = 1000

var (
	ErrIDRequired             = errors.New("job ID is required")
	ErrTypeRequired           = errors.New("job type is required")
	ErrReferenceKindRequired  = errors.New("job reference kind is required")
	ErrReferenceIDRequired    = errors.New("job reference ID is required")
	ErrMaxAttemptsInvalid     = errors.New("job max attempts must be positive")
	ErrCreatedAtRequired      = errors.New("job createdAt is required")
	ErrScheduledAtRequired    = errors.New("job scheduledAt is required")
	ErrLeaseDurationInvalid   = errors.New("job lease duration must be positive")
	ErrClaimTimeRequired      = errors.New("job claim time is required")
	ErrCompletionTimeRequired = errors.New("job completion time is required")
	ErrRetryTimeRequired      = errors.New("job retry time is required")
	ErrErrorCodeRequired      = errors.New("job error code is required")
)

// Type identifies the business operation a job handler performs.
type Type string

// Reference identifies the domain entity being processed without coupling jobs
// to a particular domain package.
type Reference struct {
	Kind string
	ID   string
}

// State describes the lifecycle of a Job.
type State string

const (
	StateQueued         State = "queued"
	StateRunning        State = "running"
	StateRetryScheduled State = "retry_scheduled"
	StateSucceeded      State = "succeeded"
	StateFailed         State = "failed"
)

func (s State) IsTerminal() bool {
	return s == StateSucceeded || s == StateFailed
}

// Error contains safe diagnostic information for a failed attempt. Message is
// deliberately a bounded, caller-supplied safe message rather than a raw error.
type Error struct {
	Code       string
	Message    string
	OccurredAt time.Time
}

// NewError creates a bounded error suitable for persisting on a Job.
func NewError(code, message string, occurredAt time.Time) (Error, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return Error{}, ErrErrorCodeRequired
	}
	return Error{
		Code:       code,
		Message:    truncateRunes(strings.TrimSpace(message), maxErrorMessageRunes),
		OccurredAt: occurredAt.UTC(),
	}, nil
}

// Job is the source of truth for asynchronous work. Broker deliveries are not
// allowed to carry or mutate this lifecycle state.
type Job struct {
	ID          string
	Type        Type
	Reference   Reference
	State       State
	Attempt     int
	MaxAttempts int

	CreatedAt   time.Time
	ScheduledAt time.Time
	AvailableAt time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	LeaseUntil  *time.Time
	LastError   *Error
}

type NewInput struct {
	ID          string
	Type        Type
	Reference   Reference
	MaxAttempts int
	CreatedAt   time.Time
	ScheduledAt time.Time
}

// New constructs a queued job ready to be claimed at ScheduledAt.
func New(input NewInput) (Job, error) {
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return Job{}, ErrIDRequired
	}
	jobType := Type(strings.TrimSpace(string(input.Type)))
	if jobType == "" {
		return Job{}, ErrTypeRequired
	}
	reference := Reference{Kind: strings.TrimSpace(input.Reference.Kind), ID: strings.TrimSpace(input.Reference.ID)}
	if reference.Kind == "" {
		return Job{}, ErrReferenceKindRequired
	}
	if reference.ID == "" {
		return Job{}, ErrReferenceIDRequired
	}
	if input.MaxAttempts <= 0 {
		return Job{}, ErrMaxAttemptsInvalid
	}
	if input.CreatedAt.IsZero() {
		return Job{}, ErrCreatedAtRequired
	}
	if input.ScheduledAt.IsZero() {
		return Job{}, ErrScheduledAtRequired
	}

	return Job{
		ID:          id,
		Type:        jobType,
		Reference:   reference,
		State:       StateQueued,
		MaxAttempts: input.MaxAttempts,
		CreatedAt:   input.CreatedAt.UTC(),
		ScheduledAt: input.ScheduledAt.UTC(),
		AvailableAt: input.ScheduledAt.UTC(),
	}, nil
}

func truncateRunes(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum])
}
