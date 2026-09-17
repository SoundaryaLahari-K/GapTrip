package trip

import (
	"context"
	"errors"
	"regexp"
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
	if matched := regexp.MustCompile(`^trip_[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(created.ID); !matched {
		t.Errorf("CreateTrip() ID = %q; want a prefixed UUIDv4", created.ID)
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

func TestParseDateUsesStrictCalendarDateSemantics(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"2028-02-29", true},
		{"2027-02-29", false},
		{"2026-6-10", false},
		{"2026-06-10T12:00:00Z", false},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			date, err := ParseDate(test.value)
			if test.valid {
				if err != nil {
					t.Fatalf("ParseDate(%q) error = %v", test.value, err)
				}
				if got := date.String(); got != test.value {
					t.Errorf("ParseDate(%q).String() = %q", test.value, got)
				}
				return
			}
			if err == nil {
				t.Errorf("ParseDate(%q) succeeded; want an error", test.value)
			}
		})
	}
}

func TestTripAllowsSameStartAndEndDate(t *testing.T) {
	date, err := ParseDate("2026-06-10")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}
	if _, err := New("trip_1", "Day trip", "Lisbon", date, date, time.Now()); err != nil {
		t.Errorf("New() error = %v; want same-day trips to be valid", err)
	}
}

func TestServiceRetriesAfterDuplicateID(t *testing.T) {
	repository := NewMemoryRepository()
	date, err := ParseDate("2026-06-10")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}
	existing, err := New("trip_existing", "Existing", "Lisbon", date, date, time.Now())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := repository.Create(context.Background(), existing); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	service := NewService(repository)
	ids := []string{"trip_existing", "trip_unique"}
	service.newID = func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}
	created, err := service.CreateTrip(context.Background(), CreateInput{Name: "New trip", Destination: "Lisbon", StartsOn: "2026-06-10", EndsOn: "2026-06-15"})
	if err != nil {
		t.Fatalf("CreateTrip() error = %v", err)
	}
	if created.ID != "trip_unique" {
		t.Errorf("CreateTrip() ID = %q; want trip_unique after retry", created.ID)
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
	if err := repository.Create(context.Background(), want); !errors.Is(err, ErrDuplicateID) {
		t.Errorf("Create(duplicate) error = %v; want ErrDuplicateID", err)
	}
	if _, err := repository.GetByID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID(missing) error = %v; want ErrNotFound", err)
	}
}
