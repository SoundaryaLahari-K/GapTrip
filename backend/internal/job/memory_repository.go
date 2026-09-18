package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// MemoryRepository is a concurrency-safe, process-local job repository for
// early development and deterministic tests.
type MemoryRepository struct {
	mu            sync.RWMutex
	jobs          map[string]storedJob
	newClaimToken func() (string, error)
}

type storedJob struct {
	job        Job
	claimToken string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{jobs: make(map[string]storedJob), newClaimToken: newClaimToken}
}

func (r *MemoryRepository) Create(ctx context.Context, job Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.jobs[job.ID]; exists {
		return ErrDuplicateID
	}
	r.jobs[job.ID] = storedJob{job: cloneJob(job)}
	return nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id string) (Job, error) {
	if err := ctx.Err(); err != nil {
		return Job{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return Job{}, err
	}
	stored, exists := r.jobs[id]
	if !exists {
		return Job{}, ErrNotFound
	}
	return cloneJob(stored.job), nil
}

func (r *MemoryRepository) Claim(ctx context.Context, id string, now time.Time, leaseDuration time.Duration) (Execution, ClaimOutcome, error) {
	if err := ctx.Err(); err != nil {
		return Execution{}, "", err
	}
	if leaseDuration <= 0 {
		return Execution{}, "", ErrLeaseDurationInvalid
	}
	if now.IsZero() {
		return Execution{}, "", ErrClaimTimeRequired
	}
	now = now.UTC()

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Execution{}, "", err
	}
	stored, exists := r.jobs[id]
	if !exists {
		return Execution{}, "", ErrNotFound
	}
	if stored.job.State.IsTerminal() {
		return Execution{}, ClaimTerminal, nil
	}
	if !claimable(stored.job, now) {
		return Execution{}, ClaimUnavailable, nil
	}
	if stored.job.Attempt >= stored.job.MaxAttempts {
		return Execution{}, ClaimUnavailable, nil
	}
	token, err := r.newClaimToken()
	if err != nil {
		return Execution{}, "", err
	}
	startedAt := now
	leaseUntil := now.Add(leaseDuration).UTC()
	stored.job.State = StateRunning
	stored.job.Attempt++
	stored.job.StartedAt = &startedAt
	stored.job.LeaseUntil = &leaseUntil
	stored.claimToken = token
	r.jobs[id] = stored
	return Execution{Job: cloneJob(stored.job), attempt: stored.job.Attempt, claimToken: token}, ClaimAcquired, nil
}

func (r *MemoryRepository) Succeed(ctx context.Context, execution Execution, now time.Time) error {
	if now.IsZero() {
		return ErrCompletionTimeRequired
	}
	return r.transition(ctx, execution, func(job *Job) error {
		completedAt := now.UTC()
		job.State = StateSucceeded
		job.CompletedAt = &completedAt
		job.LeaseUntil = nil
		return nil
	})
}

func (r *MemoryRepository) Reschedule(ctx context.Context, execution Execution, now, availableAt time.Time, jobError Error) error {
	if now.IsZero() {
		return ErrCompletionTimeRequired
	}
	if availableAt.IsZero() {
		return ErrRetryTimeRequired
	}
	if err := validateError(jobError); err != nil {
		return err
	}
	return r.transition(ctx, execution, func(job *Job) error {
		if execution.attempt >= job.MaxAttempts {
			return ErrRetryAttemptsExhausted
		}
		job.State = StateRetryScheduled
		job.AvailableAt = availableAt.UTC()
		job.LeaseUntil = nil
		value := normalizedError(jobError, now)
		job.LastError = &value
		return nil
	})
}

func (r *MemoryRepository) Fail(ctx context.Context, execution Execution, now time.Time, jobError Error) error {
	if now.IsZero() {
		return ErrCompletionTimeRequired
	}
	if err := validateError(jobError); err != nil {
		return err
	}
	return r.transition(ctx, execution, func(job *Job) error {
		completedAt := now.UTC()
		job.State = StateFailed
		job.CompletedAt = &completedAt
		job.LeaseUntil = nil
		value := normalizedError(jobError, now)
		job.LastError = &value
		return nil
	})
}

func (r *MemoryRepository) transition(ctx context.Context, execution Execution, mutate func(*Job) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	stored, exists := r.jobs[execution.Job.ID]
	if !exists {
		return ErrNotFound
	}
	if stored.job.State != StateRunning || stored.job.Attempt != execution.attempt || stored.claimToken != execution.claimToken {
		return ErrStaleExecution
	}
	if err := mutate(&stored.job); err != nil {
		return err
	}
	stored.claimToken = ""
	r.jobs[stored.job.ID] = stored
	return nil
}

func claimable(job Job, now time.Time) bool {
	switch job.State {
	case StateQueued, StateRetryScheduled:
		return !job.AvailableAt.After(now)
	case StateRunning:
		return job.LeaseUntil != nil && !job.LeaseUntil.After(now)
	default:
		return false
	}
}

func validateError(value Error) error {
	if strings.TrimSpace(value.Code) == "" {
		return ErrErrorCodeRequired
	}
	return nil
}

func normalizedError(value Error, fallbackTime time.Time) Error {
	value.Code = truncateRunes(strings.TrimSpace(value.Code), maxErrorMessageRunes)
	value.Message = truncateRunes(strings.TrimSpace(value.Message), maxErrorMessageRunes)
	if value.OccurredAt.IsZero() {
		value.OccurredAt = fallbackTime
	}
	value.OccurredAt = value.OccurredAt.UTC()
	return value
}

func cloneJob(value Job) Job {
	copy := value
	if value.StartedAt != nil {
		timestamp := *value.StartedAt
		copy.StartedAt = &timestamp
	}
	if value.CompletedAt != nil {
		timestamp := *value.CompletedAt
		copy.CompletedAt = &timestamp
	}
	if value.LeaseUntil != nil {
		timestamp := *value.LeaseUntil
		copy.LeaseUntil = &timestamp
	}
	if value.LastError != nil {
		err := *value.LastError
		copy.LastError = &err
	}
	return copy
}

func newClaimToken() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}
