// Package itinerary contains the itinerary planning domain.
package itinerary

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SoundaryaLahari-K/trippie/internal/trip"
)

var (
	ErrTripIDRequired          = errors.New("trip ID is required")
	ErrItineraryIDRequired     = errors.New("itinerary ID is required")
	ErrDayIDRequired           = errors.New("day ID is required")
	ErrDayDateRequired         = errors.New("date is required")
	ErrDayDateInvalid          = errors.New("date must use YYYY-MM-DD")
	ErrDayOutsideTrip          = errors.New("day date must be within the trip date range")
	ErrActivityTitleRequired   = errors.New("activity title is required")
	ErrActivityStartRequired   = errors.New("activity start time is required")
	ErrActivityEndRequired     = errors.New("activity end time is required")
	ErrActivityTimeInvalid     = errors.New("activity time must use HH:MM")
	ErrActivityEndBeforeStart  = errors.New("activity end time must be on or after start time")
	ErrActivityLocationTooLong = errors.New("activity location must not exceed 500 characters")
)

// Itinerary belongs to one trip. A trip has at most one itinerary.
type Itinerary struct{ ID, TripID string }
type Day struct {
	ID, ItineraryID string
	Date            trip.Date
}
type Activity struct {
	ID, DayID, Title, Location string
	StartTime, EndTime         TimeOfDay
	Fixed                      bool
}

func NewItinerary(id, tripID string) (Itinerary, error) {
	if strings.TrimSpace(id) == "" {
		return Itinerary{}, ErrItineraryIDRequired
	}
	if strings.TrimSpace(tripID) == "" {
		return Itinerary{}, ErrTripIDRequired
	}
	return Itinerary{ID: id, TripID: tripID}, nil
}
func NewDay(id, itineraryID string, date trip.Date, itineraryTrip trip.Trip) (Day, error) {
	if strings.TrimSpace(id) == "" {
		return Day{}, ErrDayIDRequired
	}
	if strings.TrimSpace(itineraryID) == "" {
		return Day{}, ErrItineraryIDRequired
	}
	if date.IsZero() {
		return Day{}, ErrDayDateRequired
	}
	if date.Before(itineraryTrip.StartsOn) || itineraryTrip.EndsOn.Before(date) {
		return Day{}, ErrDayOutsideTrip
	}
	return Day{ID: id, ItineraryID: itineraryID, Date: date}, nil
}
func NewActivity(id, dayID, title string, start, end TimeOfDay, location string, fixed bool) (Activity, error) {
	if strings.TrimSpace(id) == "" {
		return Activity{}, errors.New("activity ID is required")
	}
	if strings.TrimSpace(dayID) == "" {
		return Activity{}, ErrDayIDRequired
	}
	title = strings.TrimSpace(title)
	location = strings.TrimSpace(location)
	if title == "" {
		return Activity{}, ErrActivityTitleRequired
	}
	if start.IsZero() {
		return Activity{}, ErrActivityStartRequired
	}
	if end.IsZero() {
		return Activity{}, ErrActivityEndRequired
	}
	if end.Before(start) {
		return Activity{}, ErrActivityEndBeforeStart
	}
	if len([]rune(location)) > 500 {
		return Activity{}, ErrActivityLocationTooLong
	}
	return Activity{ID: id, DayID: dayID, Title: title, StartTime: start, EndTime: end, Location: location, Fixed: fixed}, nil
}

// TimeOfDay deliberately has no date or timezone. It is local schedule wall-clock time.
type TimeOfDay struct {
	hour, minute int
	valid        bool
}

func ParseTimeOfDay(value string) (TimeOfDay, error) {
	var h, m int
	if _, err := fmt.Sscanf(value, "%02d:%02d", &h, &m); err != nil || len(value) != 5 || value[2] != ':' || h > 23 || m > 59 {
		return TimeOfDay{}, ErrActivityTimeInvalid
	}
	return TimeOfDay{hour: h, minute: m, valid: true}, nil
}
func (t TimeOfDay) IsZero() bool { return !t.valid }
func (t TimeOfDay) Before(other TimeOfDay) bool {
	return t.hour < other.hour || (t.hour == other.hour && t.minute < other.minute)
}
func (t TimeOfDay) String() string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%02d:%02d", t.hour, t.minute)
}

// Minutes returns the number of minutes since midnight. It is intended for
// deterministic schedule calculations; callers should check IsZero first.
func (t TimeOfDay) Minutes() int { return t.hour*60 + t.minute }
