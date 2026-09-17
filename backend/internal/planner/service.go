package planner

import (
	"context"
	"github.com/SoundaryaLahari-K/trippie/internal/itinerary"
	"github.com/SoundaryaLahari-K/trippie/internal/place"
)

type Service struct {
	days       itinerary.DayRepository
	activities itinerary.ActivityRepository
	detector   Detector
	discovery  Discovery
	ranker     Ranker
}

func NewService(days itinerary.DayRepository, activities itinerary.ActivityRepository, places place.Repository) *Service {
	return &Service{days: days, activities: activities, detector: Detector{MinimumDurationMinutes: 15}, discovery: Discovery{Places: places}}
}
func (s *Service) Gaps(ctx context.Context, dayID string) ([]Gap, error) {
	if _, err := s.days.GetByID(ctx, dayID); err != nil {
		return nil, err
	}
	a, err := s.activities.ListByDayID(ctx, dayID)
	if err != nil {
		return nil, err
	}
	return s.detector.Detect(dayID, a), nil
}
func (s *Service) Suggestions(ctx context.Context, dayID string, request DiscoveryRequest) ([]Suggestion, error) {
	gaps, err := s.Gaps(ctx, dayID)
	if err != nil {
		return nil, err
	}
	out := []Suggestion{}
	for _, g := range gaps {
		c, err := s.discovery.Find(ctx, g, request)
		if err != nil {
			return nil, err
		}
		out = append(out, s.ranker.Rank(g, c, request.Categories)...)
	}
	return out, nil
}
