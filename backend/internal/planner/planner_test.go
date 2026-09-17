package planner

import (
	"context"
	"testing"

	"github.com/SoundaryaLahari-K/trippie/internal/itinerary"
	"github.com/SoundaryaLahari-K/trippie/internal/place"
)

func timeValue(t *testing.T, s string) itinerary.TimeOfDay {
	t.Helper()
	v, e := itinerary.ParseTimeOfDay(s)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func activity(t *testing.T, id, start, end string) itinerary.Activity {
	t.Helper()
	a, e := itinerary.NewActivity(id, "day", id, timeValue(t, start), timeValue(t, end), "", true)
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func TestDetectorRules(t *testing.T) {
	cases := []struct {
		name       string
		activities []itinerary.Activity
		minimum    int
		want       []int
	}{
		{"no activities", nil, 15, []int{1440}}, {"one activity", []itinerary.Activity{activity(t, "a", "09:00", "10:00")}, 15, []int{540, 840}},
		{"between", []itinerary.Activity{activity(t, "a", "09:00", "10:00"), activity(t, "b", "12:00", "13:00")}, 15, []int{540, 120, 660}},
		{"adjacent", []itinerary.Activity{activity(t, "a", "09:00", "10:00"), activity(t, "b", "10:00", "11:00")}, 15, []int{540, 780}},
		{"overlap", []itinerary.Activity{activity(t, "a", "09:00", "12:00"), activity(t, "b", "10:00", "13:00")}, 15, []int{540, 660}},
		{"minimum", []itinerary.Activity{activity(t, "a", "09:00", "10:00"), activity(t, "b", "10:10", "11:00")}, 15, []int{540, 780}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detector{tc.minimum}.Detect("day", tc.activities)
			if len(got) != len(tc.want) {
				t.Fatalf("gaps=%v", got)
			}
			for i := range got {
				if got[i].DurationMinutes != tc.want[i] {
					t.Fatalf("gap %d = %d want %d", i, got[i].DurationMinutes, tc.want[i])
				}
			}
		})
	}
}
func TestDetectorDoesNotMutateAndSortsDeterministically(t *testing.T) {
	as := []itinerary.Activity{activity(t, "b", "12:00", "13:00"), activity(t, "a", "09:00", "10:00")}
	got := Detector{15}.Detect("day", as)
	if as[0].ID != "b" || got[1].StartTime.String() != "10:00" {
		t.Fatalf("input mutated or result unsorted: %#v", got)
	}
}
func TestDiscoveryHardConstraintsAndOrdering(t *testing.T) {
	r := place.NewMemoryRepository()
	for _, p := range []place.Place{{ID: "z", Category: place.Cafe, Location: place.Location{}, AverageVisitMinutes: 30}, {ID: "a", Category: place.Restaurant, Location: place.Location{}, AverageVisitMinutes: 91}, {ID: "b", Category: place.Cafe, Location: place.Location{Latitude: 2}, AverageVisitMinutes: 30}} {
		_ = r.Create(context.Background(), p)
	}
	got, e := (Discovery{r}).Find(context.Background(), Gap{DurationMinutes: 90}, DiscoveryRequest{Categories: []place.Category{place.Cafe}, MaximumDistanceKM: 10})
	if e != nil || len(got) != 1 || got[0].Place.ID != "z" {
		t.Fatalf("got %#v err %v", got, e)
	}
}
func TestRankerInputsAndTieBreak(t *testing.T) {
	g := Gap{DayID: "d", DurationMinutes: 120}
	candidates := []Candidate{{place.Place{ID: "b", Category: place.Cafe, AverageVisitMinutes: 60, Rating: 4}, 2}, {place.Place{ID: "a", Category: place.Cafe, AverageVisitMinutes: 60, Rating: 4}, 2}, {place.Place{ID: "high", Category: place.Restaurant, AverageVisitMinutes: 120, Rating: 4.8}, 1}}
	got := (Ranker{}).Rank(g, candidates, []place.Category{place.Cafe})
	if got[0].Place.ID != "high" || got[1].Place.ID != "a" || len(got[1].Reasons) != 4 {
		t.Fatalf("unexpected ranking: %#v", got)
	}
}
func TestRankerUsesDistinctIDsForSamePlaceAcrossGaps(t *testing.T) {
	candidate := Candidate{Place: place.Place{ID: "cafe", AverageVisitMinutes: 30, Rating: 4.5}}
	first := (Ranker{}).Rank(Gap{DayID: "day", StartTime: timeValue(t, "09:00"), DurationMinutes: 60}, []Candidate{candidate}, nil)
	second := (Ranker{}).Rank(Gap{DayID: "day", StartTime: timeValue(t, "12:00"), DurationMinutes: 60}, []Candidate{candidate}, nil)
	if first[0].ID == second[0].ID {
		t.Fatalf("suggestion IDs must be unique per gap and place: %q", first[0].ID)
	}
}
func TestDistanceKM(t *testing.T) {
	d := DistanceKM(place.Location{Latitude: 0, Longitude: 0}, place.Location{Latitude: 0, Longitude: 1})
	if d < 111 || d > 112 {
		t.Fatalf("distance=%f", d)
	}
}
