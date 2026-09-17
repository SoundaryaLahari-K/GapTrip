package itinerary

import (
	"context"
	"sync"
)

type MemoryItineraryRepository struct {
	mu      sync.RWMutex
	values  map[string]Itinerary
	tripIDs map[string]string
}

func NewMemoryItineraryRepository() *MemoryItineraryRepository {
	return &MemoryItineraryRepository{values: map[string]Itinerary{}, tripIDs: map[string]string{}}
}
func (r *MemoryItineraryRepository) Create(c context.Context, v Itinerary) error {
	if e := c.Err(); e != nil {
		return e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.values[v.ID]; ok {
		return ErrDuplicateID
	}
	if _, ok := r.tripIDs[v.TripID]; ok {
		return ErrItineraryExists
	}
	r.values[v.ID] = v
	r.tripIDs[v.TripID] = v.ID
	return nil
}
func (r *MemoryItineraryRepository) GetByID(c context.Context, id string) (Itinerary, error) {
	if e := c.Err(); e != nil {
		return Itinerary{}, e
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.values[id]
	if !ok {
		return Itinerary{}, ErrNotFound
	}
	return v, nil
}
func (r *MemoryItineraryRepository) GetByTripID(c context.Context, id string) (Itinerary, error) {
	if e := c.Err(); e != nil {
		return Itinerary{}, e
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.values[r.tripIDs[id]]
	if !ok {
		return Itinerary{}, ErrNotFound
	}
	return v, nil
}

type MemoryDayRepository struct {
	mu     sync.RWMutex
	values map[string]Day
}

func NewMemoryDayRepository() *MemoryDayRepository {
	return &MemoryDayRepository{values: map[string]Day{}}
}
func (r *MemoryDayRepository) Create(c context.Context, v Day) error {
	if e := c.Err(); e != nil {
		return e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.values[v.ID]; ok {
		return ErrDuplicateID
	}
	for _, d := range r.values {
		if d.ItineraryID == v.ItineraryID && d.Date.String() == v.Date.String() {
			return ErrDuplicateID
		}
	}
	r.values[v.ID] = v
	return nil
}
func (r *MemoryDayRepository) GetByID(c context.Context, id string) (Day, error) {
	if e := c.Err(); e != nil {
		return Day{}, e
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.values[id]
	if !ok {
		return Day{}, ErrNotFound
	}
	return v, nil
}
func (r *MemoryDayRepository) ListByItineraryID(c context.Context, id string) (out []Day, e error) {
	if e = c.Err(); e != nil {
		return nil, e
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.values {
		if v.ItineraryID == id {
			out = append(out, v)
		}
	}
	return out, nil
}

type MemoryActivityRepository struct {
	mu     sync.RWMutex
	values map[string]Activity
}

func NewMemoryActivityRepository() *MemoryActivityRepository {
	return &MemoryActivityRepository{values: map[string]Activity{}}
}
func (r *MemoryActivityRepository) Create(c context.Context, v Activity) error {
	if e := c.Err(); e != nil {
		return e
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.values[v.ID]; ok {
		return ErrDuplicateID
	}
	r.values[v.ID] = v
	return nil
}
func (r *MemoryActivityRepository) ListByDayID(c context.Context, id string) (out []Activity, e error) {
	if e = c.Err(); e != nil {
		return nil, e
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.values {
		if v.DayID == id {
			out = append(out, v)
		}
	}
	return out, nil
}
