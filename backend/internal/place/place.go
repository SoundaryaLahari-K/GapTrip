// Package place contains the small, provider-independent place catalog domain.
package place

import (
	"context"
	"errors"
	"sort"
	"sync"
)

type Category string

const (
	Cafe       Category = "CAFE"
	Restaurant Category = "RESTAURANT"
	Spa        Category = "SPA"
	Market     Category = "MARKET"
	Experience Category = "EXPERIENCE"
)

type Location struct{ Latitude, Longitude float64 }
type Place struct {
	ID, Name            string
	Category            Category
	Location            Location
	AverageVisitMinutes int
	Rating              float64
	PriceLevel          int
}

var ErrNotFound = errors.New("place not found")
var ErrDuplicateID = errors.New("place ID already exists")

type Repository interface {
	Create(context.Context, Place) error
	GetByID(context.Context, string) (Place, error)
	List(context.Context) ([]Place, error)
}

type MemoryRepository struct {
	mu     sync.RWMutex
	places map[string]Place
}

func NewMemoryRepository() *MemoryRepository { return &MemoryRepository{places: map[string]Place{}} }
func (r *MemoryRepository) Create(ctx context.Context, p Place) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.places[p.ID]; ok {
		return ErrDuplicateID
	}
	r.places[p.ID] = p
	return nil
}
func (r *MemoryRepository) GetByID(ctx context.Context, id string) (Place, error) {
	if err := ctx.Err(); err != nil {
		return Place{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.places[id]
	if !ok {
		return Place{}, ErrNotFound
	}
	return p, nil
}
func (r *MemoryRepository) List(ctx context.Context) ([]Place, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Place, 0, len(r.places))
	for _, p := range r.places {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// SeedDemoRepository is an intentionally local, deterministic mock catalog near Paris.
func SeedDemoRepository() *MemoryRepository {
	r := NewMemoryRepository()
	for _, p := range []Place{
		{"place_cafe_lumiere", "Cafe Lumiere", Cafe, Location{48.8566, 2.3522}, 45, 4.7, 2},
		{"place_bistro_etoile", "Bistro Etoile", Restaurant, Location{48.8582, 2.3498}, 90, 4.6, 3},
		{"place_serene_spa", "Serene Spa", Spa, Location{48.8535, 2.3570}, 120, 4.8, 4},
		{"place_marais_market", "Marais Market", Market, Location{48.8601, 2.3622}, 60, 4.4, 1},
		{"place_river_walk", "River Walk Experience", Experience, Location{48.8519, 2.3499}, 75, 4.5, 2},
	} {
		_ = r.Create(context.Background(), p)
	}
	return r
}
