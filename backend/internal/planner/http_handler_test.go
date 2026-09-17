package planner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SoundaryaLahari-K/trippie/internal/itinerary"
	"github.com/SoundaryaLahari-K/trippie/internal/place"
)

func plannerHandler(t *testing.T) http.Handler {
	t.Helper()
	days := itinerary.NewMemoryDayRepository()
	activities := itinerary.NewMemoryActivityRepository()
	if err := days.Create(context.Background(), itinerary.Day{ID: "day_1"}); err != nil {
		t.Fatal(err)
	}
	breakfast, err := itinerary.NewActivity("activity_1", "day_1", "Breakfast", timeValue(t, "09:00"), timeValue(t, "10:00"), "", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := activities.Create(context.Background(), breakfast); err != nil {
		t.Fatal(err)
	}
	a, err := itinerary.NewActivity("activity_2", "day_1", "Lunch", timeValue(t, "12:00"), timeValue(t, "13:00"), "", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := activities.Create(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewHTTPHandler(NewService(days, activities, place.SeedDemoRepository())).RegisterRoutes(mux)
	return mux
}
func TestGapsEndpoint(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/v1/days/day_1/gaps", nil)
	w := httptest.NewRecorder()
	plannerHandler(t).ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"startTime":"00:00"`) || !strings.Contains(w.Body.String(), `"endTime":"24:00"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Gaps []struct {
			StartTime, EndTime string
			DurationMinutes    int
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Gaps) != 3 || body.Gaps[1].StartTime != "10:00" || body.Gaps[1].EndTime != "12:00" || body.Gaps[1].DurationMinutes != 120 {
		t.Fatalf("expected middle 10:00-12:00 gap, got %#v", body.Gaps)
	}
}
func TestSuggestionsEndpointAndErrors(t *testing.T) {
	h := plannerHandler(t)
	for _, tc := range []struct {
		path string
		want int
	}{{"/v1/days/day_1/suggestions?categories=CAFE", 200}, {"/v1/days/missing/suggestions", 404}, {"/v1/days/day_1/suggestions?maxDistanceKm=no", 400}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != tc.want {
			t.Errorf("%s status=%d body=%s", tc.path, w.Code, w.Body.String())
		}
	}
}
