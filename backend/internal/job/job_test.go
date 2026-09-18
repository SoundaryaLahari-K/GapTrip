package job

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func testTime() time.Time {
	return time.Date(2026, 9, 18, 12, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
}

func newTestJob(t *testing.T, id string) Job {
	t.Helper()
	created, err := New(NewInput{
		ID:          id,
		Type:        "planner.generate_suggestions.v1",
		Reference:   Reference{Kind: "day", ID: "day_1"},
		MaxAttempts: 3,
		CreatedAt:   testTime(),
		ScheduledAt: testTime().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return created
}

func TestNewValidatesAndNormalizesJob(t *testing.T) {
	job, err := New(NewInput{
		ID:          " job_1 ",
		Type:        " planner.generate_suggestions.v1 ",
		Reference:   Reference{Kind: " day ", ID: " day_1 "},
		MaxAttempts: 2,
		CreatedAt:   testTime(),
		ScheduledAt: testTime().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if job.ID != "job_1" || job.Type != "planner.generate_suggestions.v1" || job.Reference != (Reference{Kind: "day", ID: "day_1"}) {
		t.Fatalf("New() = %#v; want normalized values", job)
	}
	if job.State != StateQueued || job.Attempt != 0 || !job.AvailableAt.Equal(job.ScheduledAt) {
		t.Fatalf("New() = %#v; want queued initial state", job)
	}
	if job.CreatedAt.Location() != time.UTC || job.ScheduledAt.Location() != time.UTC {
		t.Fatalf("timestamps must be UTC: %#v", job)
	}
}

func TestNewValidation(t *testing.T) {
	valid := NewInput{ID: "job_1", Type: "type", Reference: Reference{Kind: "day", ID: "day_1"}, MaxAttempts: 1, CreatedAt: testTime(), ScheduledAt: testTime()}
	tests := []struct {
		name string
		edit func(*NewInput)
		want error
	}{
		{"ID", func(in *NewInput) { in.ID = " " }, ErrIDRequired},
		{"type", func(in *NewInput) { in.Type = " " }, ErrTypeRequired},
		{"reference kind", func(in *NewInput) { in.Reference.Kind = " " }, ErrReferenceKindRequired},
		{"reference ID", func(in *NewInput) { in.Reference.ID = " " }, ErrReferenceIDRequired},
		{"attempts", func(in *NewInput) { in.MaxAttempts = 0 }, ErrMaxAttemptsInvalid},
		{"createdAt", func(in *NewInput) { in.CreatedAt = time.Time{} }, ErrCreatedAtRequired},
		{"scheduledAt", func(in *NewInput) { in.ScheduledAt = time.Time{} }, ErrScheduledAtRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.edit(&input)
			_, err := New(input)
			if !errors.Is(err, test.want) {
				t.Fatalf("New() error = %v; want %v", err, test.want)
			}
		})
	}
}

func TestNewErrorTruncatesSafeMessage(t *testing.T) {
	message := strings.Repeat("世", maxErrorMessageRunes+1)
	value, err := NewError(" retryable ", "  "+message+"  ", testTime())
	if err != nil {
		t.Fatalf("NewError() error = %v", err)
	}
	if value.Code != "retryable" || len([]rune(value.Message)) != maxErrorMessageRunes || value.OccurredAt.Location() != time.UTC {
		t.Fatalf("NewError() = %#v", value)
	}
	if _, err := NewError(" ", "safe", testTime()); !errors.Is(err, ErrErrorCodeRequired) {
		t.Fatalf("NewError(empty code) error = %v; want %v", err, ErrErrorCodeRequired)
	}
}

func TestStateIsTerminal(t *testing.T) {
	if !StateSucceeded.IsTerminal() || !StateFailed.IsTerminal() || StateQueued.IsTerminal() || StateRunning.IsTerminal() || StateRetryScheduled.IsTerminal() {
		t.Fatal("unexpected terminal state classification")
	}
}
