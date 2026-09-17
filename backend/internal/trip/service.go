package trip

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const maxIDGenerationAttempts = 3

type CreateInput struct {
	Name        string
	Destination string
	StartsOn    string
	EndsOn      string
}

type Service struct {
	repository Repository
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now, newID: newTripID}
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

	createdAt := s.now().UTC()
	for range maxIDGenerationAttempts {
		id, err := s.newID()
		if err != nil {
			return Trip{}, fmt.Errorf("generate trip ID: %w", err)
		}
		newTrip, err := New(id, input.Name, input.Destination, startsOn, endsOn, createdAt)
		if err != nil {
			return Trip{}, err
		}
		if err := s.repository.Create(ctx, newTrip); err == nil {
			return newTrip, nil
		} else if !errors.Is(err, ErrDuplicateID) {
			return Trip{}, err
		}
	}
	return Trip{}, errors.New("could not generate a unique trip ID")
}

func newTripID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	// Set UUID version (4) and variant (RFC 4122) bits.
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes[:])
	return fmt.Sprintf("trip_%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32]), nil
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
