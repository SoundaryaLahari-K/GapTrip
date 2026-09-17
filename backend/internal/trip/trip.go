package trip

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

var (
	ErrNameRequired        = errors.New("name is required")
	ErrDestinationRequired = errors.New("destination is required")
	ErrStartsOnRequired    = errors.New("startsOn is required")
	ErrEndsOnRequired      = errors.New("endsOn is required")
	ErrStartsOnInvalid     = errors.New("startsOn must use YYYY-MM-DD")
	ErrEndsOnInvalid       = errors.New("endsOn must use YYYY-MM-DD")
	ErrEndsOnBeforeStart   = errors.New("endsOn must be on or after startsOn")
)

// Date represents a calendar day without a time of day or timezone.
type Date struct {
	year  int
	month time.Month
	day   int
}

func ParseDate(value string) (Date, error) {
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return Date{}, fmt.Errorf("must use YYYY-MM-DD: %w", err)
	}

	if parsed.Format(dateLayout) != value {
		return Date{}, errors.New("date must use YYYY-MM-DD")
	}

	return Date{year: parsed.Year(), month: parsed.Month(), day: parsed.Day()}, nil
}

func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)
}

func (d Date) IsZero() bool {
	return d.year == 0
}

func (d Date) Before(other Date) bool {
	if d.year != other.year {
		return d.year < other.year
	}
	if d.month != other.month {
		return d.month < other.month
	}
	return d.day < other.day
}

// Trip is a planned visit whose dates bound future itinerary items and gap filling.
type Trip struct {
	ID          string
	Name        string
	Destination string
	StartsOn    Date
	EndsOn      Date
	CreatedAt   time.Time
}

func New(id, name, destination string, startsOn, endsOn Date, createdAt time.Time) (Trip, error) {
	name = strings.TrimSpace(name)
	destination = strings.TrimSpace(destination)

	if name == "" {
		return Trip{}, ErrNameRequired
	}
	if destination == "" {
		return Trip{}, ErrDestinationRequired
	}
	if startsOn.IsZero() {
		return Trip{}, ErrStartsOnRequired
	}
	if endsOn.IsZero() {
		return Trip{}, ErrEndsOnRequired
	}
	if endsOn.Before(startsOn) {
		return Trip{}, ErrEndsOnBeforeStart
	}

	return Trip{
		ID:          id,
		Name:        name,
		Destination: destination,
		StartsOn:    startsOn,
		EndsOn:      endsOn,
		CreatedAt:   createdAt.UTC(),
	}, nil
}
