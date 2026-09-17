package itinerary

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/SoundaryaLahari-K/trippie/internal/trip"
)

type Service struct {
	trips       trip.Repository
	itineraries ItineraryRepository
	days        DayRepository
	activities  ActivityRepository
	newID       func(string) (string, error)
}
type ItineraryView struct {
	Itinerary Itinerary
	Days      []DayView
}
type DayView struct {
	Day        Day
	Activities []Activity
}
type ActivityInput struct {
	Title, StartTime, EndTime, Location string
	Fixed                               bool
}

func NewService(t trip.Repository, i ItineraryRepository, d DayRepository, a ActivityRepository) *Service {
	return &Service{trips: t, itineraries: i, days: d, activities: a, newID: newID}
}
func (s *Service) CreateItinerary(c context.Context, tripID string) (Itinerary, error) {
	if _, e := s.trips.GetByID(c, tripID); e != nil {
		return Itinerary{}, e
	}
	id, e := s.newID("itinerary")
	if e != nil {
		return Itinerary{}, e
	}
	v, e := NewItinerary(id, tripID)
	if e != nil {
		return Itinerary{}, e
	}
	if e = s.itineraries.Create(c, v); e != nil {
		return Itinerary{}, e
	}
	return v, nil
}
func (s *Service) GetItinerary(c context.Context, id string) (ItineraryView, error) {
	v, e := s.itineraries.GetByID(c, id)
	if e != nil {
		return ItineraryView{}, e
	}
	ds, e := s.days.ListByItineraryID(c, id)
	if e != nil {
		return ItineraryView{}, e
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].Date.Before(ds[j].Date) })
	out := ItineraryView{Itinerary: v}
	for _, d := range ds {
		as, e := s.activities.ListByDayID(c, d.ID)
		if e != nil {
			return ItineraryView{}, e
		}
		sort.Slice(as, func(i, j int) bool { return as[i].StartTime.Before(as[j].StartTime) })
		out.Days = append(out.Days, DayView{Day: d, Activities: as})
	}
	return out, nil
}
func (s *Service) GetItineraryForTrip(c context.Context, tripID string) (ItineraryView, error) {
	if _, e := s.trips.GetByID(c, tripID); e != nil {
		return ItineraryView{}, e
	}
	v, e := s.itineraries.GetByTripID(c, tripID)
	if e != nil {
		return ItineraryView{}, e
	}
	return s.GetItinerary(c, v.ID)
}
func (s *Service) AddDay(c context.Context, itineraryID, dateValue string) (Day, error) {
	v, e := s.itineraries.GetByID(c, itineraryID)
	if e != nil {
		return Day{}, e
	}
	tr, e := s.trips.GetByID(c, v.TripID)
	if e != nil {
		return Day{}, e
	}
	date, e := trip.ParseDate(dateValue)
	if e != nil {
		return Day{}, ErrDayDateInvalid
	}
	id, e := s.newID("day")
	if e != nil {
		return Day{}, e
	}
	d, e := NewDay(id, itineraryID, date, tr)
	if e != nil {
		return Day{}, e
	}
	if e = s.days.Create(c, d); e != nil {
		return Day{}, e
	}
	return d, nil
}
func (s *Service) AddActivity(c context.Context, dayID string, input ActivityInput) (Activity, error) {
	if _, e := s.days.GetByID(c, dayID); e != nil {
		return Activity{}, e
	}
	start, e := parseTime(input.StartTime, ErrActivityStartRequired)
	if e != nil {
		return Activity{}, e
	}
	end, e := parseTime(input.EndTime, ErrActivityEndRequired)
	if e != nil {
		return Activity{}, e
	}
	id, e := s.newID("activity")
	if e != nil {
		return Activity{}, e
	}
	a, e := NewActivity(id, dayID, input.Title, start, end, input.Location, input.Fixed)
	if e != nil {
		return Activity{}, e
	}
	if e = s.activities.Create(c, a); e != nil {
		return Activity{}, e
	}
	return a, nil
}
func parseTime(v string, required error) (TimeOfDay, error) {
	if v == "" {
		return TimeOfDay{}, required
	}
	t, e := ParseTimeOfDay(v)
	if e != nil {
		return TimeOfDay{}, ErrActivityTimeInvalid
	}
	return t, nil
}
func newID(kind string) (string, error) {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return fmt.Sprintf("%s_%s", kind, hex.EncodeToString(b[:])), nil
}
