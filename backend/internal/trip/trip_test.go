package trip

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceCreateTrip(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(repository)
	service.now = func() time.Time { return time.Date(2026, 5, 1, 10, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60)) }

	created, err := service.CreateTrip(context.Background(), CreateInput{
		Name:        "  Summer break  ",
		Destination: "  Lisbon  ",
		StartsOn:    "2026-06-10",
		EndsOn:      "2026-06-15",
	})
	if err != nil {
		t.Fatalf("CreateTrip() error = %v", err)
	}
	if created.ID == "" {
		t.Error("CreateTrip() returned an empty ID")
	}
	if created.Name != "Summer break" || created.Destination != "Lisbon" {
		t.Errorf("CreateTrip() = %+v; want trimmed name and destination", created)
	}
	if got := created.StartsOn.String(); got != "2026-06-10" {
		t.Errorf("StartsOn = %q; want 2026-06-10", got)
	}
	if got := created.CreatedAt.Location(); got != time.UTC {
		t.Errorf("CreatedAt location = %v; want UTC", got)
	}
}

func TestServiceCreateTripValidation(t *testing.T) {
	tests := []struct {
		name  string
		input CreateInput
		want  error
	}{
		{"missing name", CreateInput{Destination: "Lisbon", StartsOn: "2026-06-10", EndsOn: "2026-06-15"}, ErrNameRequired},
		{"missing destination", CreateInput{Name: "Summer break", StartsOn: "2026-06-10", EndsOn: "2026-06-15"}, ErrDestinationRequired},
		{"missing startsOn", CreateInput{Name: "Summer break", Destination: "Lisbon", EndsOn: "2026-06-15"}, ErrStartsOnRequired},
		{"missing endsOn", CreateInput{Name: "Summer break", Destination: "Lisbon", StartsOn: "2026-06-10"}, ErrEndsOnRequired},
		{"invalid startsOn", CreateInput{Name: "Summer break", Destination: "Lisbon", StartsOn: "June 10", EndsOn: "2026-06-15"}, ErrStartsOnInvalid},
		{"ends before starts", CreateInput{Name: "Summer break", Destination: "Lisbon", StartsOn: "2026-06-15", EndsOn: "2026-06-10"}, ErrEndsOnBeforeStart},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewService(NewMemoryRepository()).CreateTrip(context.Background(), test.input)
			if !errors.Is(err, test.want) {
				t.Errorf("CreateTrip() error = %v; want %v", err, test.want)
			}
		})
	}
}

func TestMemoryRepositoryCreateAndGet(t *testing.T) {
	startsOn, _ := ParseDate("2026-06-10")
	endsOn, _ := ParseDate("2026-06-15")
	want, err := New("trip_1", "Summer break", "Lisbon", startsOn, endsOn, time.Now())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	repository := NewMemoryRepository()
	if err := repository.Create(context.Background(), want); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, err := repository.GetByID(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got != want {
		t.Errorf("GetByID() = %+v; want %+v", got, want)
	}
	if _, err := repository.GetByID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID(missing) error = %v; want ErrNotFound", err)
	}
}
