// Package planner detects free itinerary time and turns a local place catalog
// into explainable suggestions. Geographic distance is straight-line distance,
// not a routing or travel-time estimate.
package planner

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/SoundaryaLahari-K/trippie/internal/itinerary"
	"github.com/SoundaryaLahari-K/trippie/internal/place"
)

const dayMinutes = 24 * 60

type Gap struct {
	DayID              string
	StartTime, EndTime itinerary.TimeOfDay
	DurationMinutes    int
}
type Detector struct{ MinimumDurationMinutes int }

func (d Detector) Detect(dayID string, activities []itinerary.Activity) []Gap {
	minimum := d.MinimumDurationMinutes
	if minimum <= 0 {
		minimum = 1
	}
	items := append([]itinerary.Activity(nil), activities...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].StartTime.Minutes() != items[j].StartTime.Minutes() {
			return items[i].StartTime.Minutes() < items[j].StartTime.Minutes()
		}
		return items[i].ID < items[j].ID
	})
	result := []Gap{}
	cursor := 0
	for _, a := range items {
		start, end := a.StartTime.Minutes(), a.EndTime.Minutes()
		if start > cursor {
			result = appendGap(result, dayID, cursor, start, minimum)
		}
		if end > cursor {
			cursor = end
		}
	}
	return appendGap(result, dayID, cursor, dayMinutes, minimum)
}
func appendGap(gaps []Gap, dayID string, start, end, minimum int) []Gap {
	if end-start < minimum {
		return gaps
	}
	s, _ := itinerary.ParseTimeOfDay(fmt.Sprintf("%02d:%02d", start/60, start%60))
	e := itinerary.TimeOfDay{}
	if end < dayMinutes {
		e, _ = itinerary.ParseTimeOfDay(fmt.Sprintf("%02d:%02d", end/60, end%60))
	}
	return append(gaps, Gap{dayID, s, e, end - start})
}

type DiscoveryRequest struct {
	Categories        []place.Category
	Origin            place.Location
	MaximumDistanceKM float64
}
type Candidate struct {
	Place      place.Place
	DistanceKM float64
}
type Discovery struct{ Places place.Repository }

func (d Discovery) Find(ctx context.Context, gap Gap, request DiscoveryRequest) ([]Candidate, error) {
	ps, err := d.Places.List(ctx)
	if err != nil {
		return nil, err
	}
	allowed := map[place.Category]bool{}
	for _, c := range request.Categories {
		allowed[c] = true
	}
	out := []Candidate{}
	for _, p := range ps {
		if p.AverageVisitMinutes > gap.DurationMinutes || (len(allowed) > 0 && !allowed[p.Category]) {
			continue
		}
		distance := DistanceKM(request.Origin, p.Location)
		if request.MaximumDistanceKM > 0 && distance > request.MaximumDistanceKM {
			continue
		}
		out = append(out, Candidate{p, distance})
	}
	return out, nil
}
func DistanceKM(a, b place.Location) float64 {
	const radius = 6371.0
	radians := func(v float64) float64 { return v * math.Pi / 180 }
	dLat, dLon := radians(b.Latitude-a.Latitude), radians(b.Longitude-a.Longitude)
	x := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(radians(a.Latitude))*math.Cos(radians(b.Latitude))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return radius * 2 * math.Atan2(math.Sqrt(x), math.Sqrt(1-x))
}

type Suggestion struct {
	ID         string
	Gap        Gap
	Place      place.Place
	Score      float64
	Reasons    []string
	DistanceKM float64
}
type Ranker struct{}

// Rank scores rating (x10), closeness (up to 10), duration fit (up to 10), and
// a five-point category preference bonus. Ties sort by place ID.
func (Ranker) Rank(gap Gap, candidates []Candidate, preferred []place.Category) []Suggestion {
	prefs := map[place.Category]bool{}
	for _, p := range preferred {
		prefs[p] = true
	}
	out := make([]Suggestion, 0, len(candidates))
	for _, c := range candidates {
		fit := float64(c.Place.AverageVisitMinutes) / float64(gap.DurationMinutes)
		score := c.Place.Rating*10 + math.Max(0, 10-c.DistanceKM) + fit*10
		reasons := []string{fmt.Sprintf("Fits %d-minute gap", gap.DurationMinutes), fmt.Sprintf("%.1f rating", c.Place.Rating), fmt.Sprintf("%.1f km away", c.DistanceKM)}
		if prefs[c.Place.Category] {
			score += 5
			reasons = append(reasons, "Preferred category")
		}
		out = append(out, Suggestion{"suggestion_" + gap.DayID + "_" + c.Place.ID, gap, c.Place, score, reasons, c.DistanceKM})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Place.ID < out[j].Place.ID
		}
		return out[i].Score > out[j].Score
	})
	return out
}
