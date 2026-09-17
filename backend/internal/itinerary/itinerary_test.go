package itinerary

import (
	"context"
	"errors"
	"github.com/SoundaryaLahari-K/trippie/internal/trip"
	"testing"
	"time"
)

func setup(t *testing.T) (*Service, trip.Trip) {
	t.Helper()
	tr := trip.NewMemoryRepository()
	s := NewService(tr, NewMemoryItineraryRepository(), NewMemoryDayRepository(), NewMemoryActivityRepository())
	a, _ := trip.ParseDate("2026-06-10")
	b, _ := trip.ParseDate("2026-06-12")
	v, e := trip.New("trip_1", "Trip", "Lisbon", a, b, time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if e = tr.Create(context.Background(), v); e != nil {
		t.Fatal(e)
	}
	return s, v
}
func TestTripToItineraryDayActivity(t *testing.T) {
	s, tr := setup(t)
	c := context.Background()
	it, e := s.CreateItinerary(c, tr.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateItinerary(c, tr.ID); !errors.Is(e, ErrItineraryExists) {
		t.Fatalf("duplicate = %v", e)
	}
	d, e := s.AddDay(c, it.ID, "2026-06-11")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddDay(c, it.ID, "2026-06-15"); !errors.Is(e, ErrDayOutsideTrip) {
		t.Fatalf("outside = %v", e)
	}
	if _, e = s.AddDay(c, it.ID, "2026-06-11"); !errors.Is(e, ErrDuplicateID) {
		t.Fatalf("duplicate day = %v", e)
	}
	if _, e = s.AddActivity(c, d.ID, ActivityInput{StartTime: "09:00", EndTime: "10:00"}); !errors.Is(e, ErrActivityTitleRequired) {
		t.Fatalf("missing title = %v", e)
	}
	if _, e = s.AddActivity(c, d.ID, ActivityInput{Title: "Bad", StartTime: "11:00", EndTime: "10:00"}); !errors.Is(e, ErrActivityEndBeforeStart) {
		t.Fatalf("range = %v", e)
	}
	_, e = s.AddActivity(c, d.ID, ActivityInput{Title: "Late", StartTime: "14:00", EndTime: "15:00", Fixed: true})
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.AddActivity(c, d.ID, ActivityInput{Title: "Early", StartTime: "09:00", EndTime: "10:00", Location: "Museum"})
	if e != nil {
		t.Fatal(e)
	}
	view, e := s.GetItinerary(c, it.ID)
	if e != nil {
		t.Fatal(e)
	}
	if len(view.Days) != 1 || len(view.Days[0].Activities) != 2 || view.Days[0].Activities[0].Title != "Early" {
		t.Fatalf("view = %+v", view)
	}
}
func TestUnknownResourcesAndCanceledContext(t *testing.T) {
	s, _ := setup(t)
	if _, e := s.CreateItinerary(context.Background(), "missing"); !errors.Is(e, trip.ErrNotFound) {
		t.Fatalf("unknown trip = %v", e)
	}
	if _, e := s.AddDay(context.Background(), "missing", "2026-06-10"); !errors.Is(e, ErrNotFound) {
		t.Fatalf("unknown itinerary = %v", e)
	}
	if _, e := s.AddActivity(context.Background(), "missing", ActivityInput{}); !errors.Is(e, ErrNotFound) {
		t.Fatalf("unknown day = %v", e)
	}
	c, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.CreateItinerary(c, "trip_1"); !errors.Is(e, context.Canceled) {
		t.Fatalf("canceled = %v", e)
	}
}
func TestTimeOfDayMidnight(t *testing.T) {
	v, e := ParseTimeOfDay("00:00")
	if e != nil || v.IsZero() {
		t.Fatalf("midnight = %#v, %v", v, e)
	}
}
