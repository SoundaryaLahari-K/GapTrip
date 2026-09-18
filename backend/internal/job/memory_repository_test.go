package job

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func createJob(t *testing.T, repository Repository, id string) Job {
	t.Helper()
	job := newTestJob(t, id)
	if err := repository.Create(context.Background(), job); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return job
}

func mustError(t *testing.T, code string, at time.Time) Error {
	t.Helper()
	value, err := NewError(code, "safe diagnostic", at)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestMemoryRepositoryCreateGetAndContext(t *testing.T) {
	repository := NewMemoryRepository()
	job := createJob(t, repository, "job_1")
	got, err := repository.GetByID(context.Background(), job.ID)
	if err != nil || got != job {
		t.Fatalf("GetByID() = %#v, %v; want %#v, nil", got, err, job)
	}
	if err := repository.Create(context.Background(), job); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("Create(duplicate) error = %v; want %v", err, ErrDuplicateID)
	}
	if _, err := repository.GetByID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID(missing) error = %v; want %v", err, ErrNotFound)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.Create(ctx, newTestJob(t, "job_2")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create(canceled) error = %v", err)
	}
	if _, err := repository.GetByID(ctx, job.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByID(canceled) error = %v", err)
	}
}

func TestMemoryRepositoryClaimsTransitionsAndProtectsStaleExecution(t *testing.T) {
	repository := NewMemoryRepository()
	job := createJob(t, repository, "job_1")
	now := job.ScheduledAt
	execution, outcome, err := repository.Claim(context.Background(), job.ID, now, time.Minute)
	if err != nil || outcome != ClaimAcquired || execution.Job.State != StateRunning || execution.Job.Attempt != 1 {
		t.Fatalf("Claim() = %#v, %q, %v", execution, outcome, err)
	}
	if execution.Job.LeaseUntil == nil || !execution.Job.LeaseUntil.Equal(now.Add(time.Minute)) {
		t.Fatalf("lease = %v; want %v", execution.Job.LeaseUntil, now.Add(time.Minute))
	}
	if _, outcome, err := repository.Claim(context.Background(), job.ID, now.Add(time.Second), time.Minute); err != nil || outcome != ClaimUnavailable {
		t.Fatalf("Claim(live lease) outcome=%q err=%v", outcome, err)
	}

	second, outcome, err := repository.Claim(context.Background(), job.ID, now.Add(time.Minute), time.Minute)
	if err != nil || outcome != ClaimAcquired || second.Job.Attempt != 2 {
		t.Fatalf("Claim(expired lease) = %#v, %q, %v", second, outcome, err)
	}
	if err := repository.Succeed(context.Background(), execution, now.Add(time.Minute)); !errors.Is(err, ErrStaleExecution) {
		t.Fatalf("Succeed(stale execution) error = %v; want %v", err, ErrStaleExecution)
	}
	if err := repository.Succeed(context.Background(), second, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Succeed() error = %v", err)
	}
	stored, err := repository.GetByID(context.Background(), job.ID)
	if err != nil || stored.State != StateSucceeded || stored.CompletedAt == nil || stored.LeaseUntil != nil {
		t.Fatalf("stored job = %#v, %v", stored, err)
	}
	if _, outcome, err := repository.Claim(context.Background(), job.ID, now.Add(3*time.Minute), time.Minute); err != nil || outcome != ClaimTerminal {
		t.Fatalf("Claim(terminal) outcome=%q err=%v", outcome, err)
	}
}

func TestMemoryRepositoryReschedulesAndFails(t *testing.T) {
	repository := NewMemoryRepository()
	job := createJob(t, repository, "job_1")
	now := job.ScheduledAt
	execution, _, err := repository.Claim(context.Background(), job.ID, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	retryAt := now.Add(5 * time.Minute)
	if err := repository.Reschedule(context.Background(), execution, now.Add(time.Minute), retryAt, mustError(t, "temporary", now)); err != nil {
		t.Fatalf("Reschedule() error = %v", err)
	}
	stored, err := repository.GetByID(context.Background(), job.ID)
	if err != nil || stored.State != StateRetryScheduled || !stored.AvailableAt.Equal(retryAt) || stored.LastError == nil || stored.LeaseUntil != nil {
		t.Fatalf("stored retry job = %#v, %v", stored, err)
	}
	if _, outcome, err := repository.Claim(context.Background(), job.ID, retryAt.Add(-time.Nanosecond), time.Minute); err != nil || outcome != ClaimUnavailable {
		t.Fatalf("Claim(before retry) outcome=%q err=%v", outcome, err)
	}
	second, outcome, err := repository.Claim(context.Background(), job.ID, retryAt, time.Minute)
	if err != nil || outcome != ClaimAcquired || second.Job.Attempt != 2 {
		t.Fatalf("Claim(retry) = %#v, %q, %v", second, outcome, err)
	}
	if err := repository.Fail(context.Background(), second, retryAt.Add(time.Minute), mustError(t, "permanent", retryAt)); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	stored, err = repository.GetByID(context.Background(), job.ID)
	if err != nil || stored.State != StateFailed || stored.CompletedAt == nil || stored.LastError == nil || stored.LastError.Code != "permanent" {
		t.Fatalf("stored failed job = %#v, %v", stored, err)
	}
}

func TestMemoryRepositoryRejectsInvalidTransitions(t *testing.T) {
	repository := NewMemoryRepository()
	job := createJob(t, repository, "job_1")
	now := job.ScheduledAt
	if _, _, err := repository.Claim(context.Background(), job.ID, now, 0); !errors.Is(err, ErrLeaseDurationInvalid) {
		t.Fatalf("Claim(invalid lease) error = %v", err)
	}
	if _, _, err := repository.Claim(context.Background(), job.ID, time.Time{}, time.Minute); !errors.Is(err, ErrClaimTimeRequired) {
		t.Fatalf("Claim(zero time) error = %v", err)
	}
	execution, _, err := repository.Claim(context.Background(), job.ID, now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Reschedule(context.Background(), execution, now, time.Time{}, mustError(t, "retry", now)); !errors.Is(err, ErrRetryTimeRequired) {
		t.Fatalf("Reschedule(empty availableAt) error = %v", err)
	}
	if err := repository.Fail(context.Background(), execution, now, Error{}); !errors.Is(err, ErrErrorCodeRequired) {
		t.Fatalf("Fail(no error code) error = %v", err)
	}
	if err := repository.Succeed(context.Background(), execution, time.Time{}); !errors.Is(err, ErrCompletionTimeRequired) {
		t.Fatalf("Succeed(zero time) error = %v", err)
	}
	if err := repository.Succeed(context.Background(), execution, now); err != nil {
		t.Fatal(err)
	}
	if err := repository.Fail(context.Background(), execution, now, mustError(t, "late", now)); !errors.Is(err, ErrStaleExecution) {
		t.Fatalf("Fail(terminal) error = %v", err)
	}
}

func TestMemoryRepositoryCannotRescheduleAfterLastAttempt(t *testing.T) {
	repository := NewMemoryRepository()
	job, err := New(NewInput{
		ID:          "job_1",
		Type:        "type",
		Reference:   Reference{Kind: "day", ID: "day_1"},
		MaxAttempts: 1,
		CreatedAt:   testTime(),
		ScheduledAt: testTime(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	execution, _, err := repository.Claim(context.Background(), job.ID, job.ScheduledAt, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Reschedule(context.Background(), execution, job.ScheduledAt, job.ScheduledAt.Add(time.Minute), mustError(t, "temporary", job.ScheduledAt)); !errors.Is(err, ErrRetryAttemptsExhausted) {
		t.Fatalf("Reschedule(last attempt) error = %v; want %v", err, ErrRetryAttemptsExhausted)
	}
	stored, err := repository.GetByID(context.Background(), job.ID)
	if err != nil || stored.State != StateRunning {
		t.Fatalf("failed reschedule must not mutate job: %#v, %v", stored, err)
	}
}

func TestMemoryRepositoryDoesNotExposeMutablePointers(t *testing.T) {
	repository := NewMemoryRepository()
	job := createJob(t, repository, "job_1")
	execution, _, err := repository.Claim(context.Background(), job.ID, job.ScheduledAt, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	*execution.Job.StartedAt = time.Time{}
	stored, err := repository.GetByID(context.Background(), job.ID)
	if err != nil || stored.StartedAt == nil || stored.StartedAt.IsZero() {
		t.Fatalf("repository state was mutated through returned job: %#v, %v", stored, err)
	}
}

func TestMemoryRepositoryConcurrentClaimIsAtomic(t *testing.T) {
	repository := NewMemoryRepository()
	job := createJob(t, repository, "job_1")
	const callers = 64
	results := make(chan ClaimOutcome, callers)
	errors := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			_, outcome, err := repository.Claim(context.Background(), job.ID, job.ScheduledAt, time.Minute)
			results <- outcome
			errors <- err
		}()
	}
	group.Wait()
	close(results)
	close(errors)
	acquired := 0
	for outcome := range results {
		if outcome == ClaimAcquired {
			acquired++
		} else if outcome != ClaimUnavailable {
			t.Fatalf("Claim() outcome = %q; want acquired or unavailable", outcome)
		}
	}
	for err := range errors {
		if err != nil {
			t.Fatalf("Claim() error = %v", err)
		}
	}
	if acquired != 1 {
		t.Fatalf("acquired claims = %d; want 1", acquired)
	}
	stored, err := repository.GetByID(context.Background(), job.ID)
	if err != nil || stored.Attempt != 1 || stored.State != StateRunning {
		t.Fatalf("stored job = %#v, %v", stored, err)
	}
}
