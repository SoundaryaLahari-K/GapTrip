package trip

import (
	"context"
	"sync"
)

// MemoryRepository is a concurrency-safe repository intended for local use and early development.
type MemoryRepository struct {
	mu    sync.RWMutex
	trips map[string]Trip
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{trips: make(map[string]Trip)}
}

func (r *MemoryRepository) Create(ctx context.Context, trip Trip) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.trips[trip.ID] = trip
	return nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id string) (Trip, error) {
	if err := ctx.Err(); err != nil {
		return Trip{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	trip, ok := r.trips[id]
	if !ok {
		return Trip{}, ErrNotFound
	}
	return trip, nil
}
