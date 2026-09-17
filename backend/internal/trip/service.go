package trip

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

type CreateInput struct {
	Name        string
	Destination string
	StartsOn    string
	EndsOn      string
}

type Service struct {
	repository Repository
	counter    atomic.Uint64
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) CreateTrip(ctx context.Context, input CreateInput) (Trip, error) {
	startsOn, err := parseRequiredDate(input.StartsOn, ErrStartsOnRequired, ErrStartsOnInvalid)
	if err != nil {
		return Trip{}, err
	}
	endsOn, err := parseRequiredDate(input.EndsOn, ErrEndsOnRequired, ErrEndsOnInvalid)
	if err != nil {
		return Trip{}, err
	}

	sequence := s.counter.Add(1)
	createdAt := s.now().UTC()
	newTrip, err := New(
		fmt.Sprintf("trip_%d_%d", createdAt.UnixNano(), sequence),
		input.Name,
		input.Destination,
		startsOn,
		endsOn,
		createdAt,
	)
	if err != nil {
		return Trip{}, err
	}
	if err := s.repository.Create(ctx, newTrip); err != nil {
		return Trip{}, err
	}
	return newTrip, nil
}

func (s *Service) GetTrip(ctx context.Context, id string) (Trip, error) {
	return s.repository.GetByID(ctx, id)
}

func parseRequiredDate(value string, requiredError, invalidError error) (Date, error) {
	if value == "" {
		return Date{}, requiredError
	}
	date, err := ParseDate(value)
	if err != nil {
		return Date{}, invalidError
	}
	return date, nil
}
