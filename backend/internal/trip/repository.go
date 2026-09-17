package trip

import (
	"context"
	"errors"
)

var (
	ErrNotFound    = errors.New("trip not found")
	ErrDuplicateID = errors.New("trip ID already exists")
)

// Repository persists trips independently of the transport and use-case layers.
type Repository interface {
	Create(ctx context.Context, trip Trip) error
	GetByID(ctx context.Context, id string) (Trip, error)
}
