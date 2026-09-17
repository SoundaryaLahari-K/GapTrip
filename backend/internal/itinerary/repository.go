package itinerary

import (
	"context"
	"errors"
)

var (
	ErrNotFound        = errors.New("itinerary entity not found")
	ErrDuplicateID     = errors.New("itinerary entity ID already exists")
	ErrItineraryExists = errors.New("itinerary already exists for trip")
)

type ItineraryRepository interface {
	Create(context.Context, Itinerary) error
	GetByID(context.Context, string) (Itinerary, error)
	GetByTripID(context.Context, string) (Itinerary, error)
}
type DayRepository interface {
	Create(context.Context, Day) error
	GetByID(context.Context, string) (Day, error)
	ListByItineraryID(context.Context, string) ([]Day, error)
}
type ActivityRepository interface {
	Create(context.Context, Activity) error
	ListByDayID(context.Context, string) ([]Activity, error)
}
