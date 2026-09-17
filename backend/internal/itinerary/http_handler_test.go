package itinerary

import (
	"context"
	"encoding/json"
	"github.com/SoundaryaLahari-K/trippie/internal/trip"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func httpSetup(t *testing.T) (*Service, http.Handler, trip.Trip) {
	s, tr := setup(t)
	m := http.NewServeMux()
	NewHTTPHandler(s).RegisterRoutes(m)
	return s, m, tr
}
func TestHTTPItinerarySlice(t *testing.T) {
	_, h, tr := httpSetup(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("POST", "/v1/trips/"+tr.ID+"/itinerary", nil))
	if r.Code != 201 || r.Header().Get("Location") == "" {
		t.Fatalf("create %d %s", r.Code, r.Body.String())
	}
	var it itineraryResponse
	json.NewDecoder(r.Body).Decode(&it)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("POST", "/v1/itineraries/"+it.ID+"/days", strings.NewReader(`{"date":"2026-06-11"}`)))
	if r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	var d dayResponse
	json.NewDecoder(r.Body).Decode(&d)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("POST", "/v1/days/"+d.ID+"/activities", strings.NewReader(`{"title":"Lunch","startTime":"12:00","endTime":"13:00","location":"Cafe","fixed":true}`)))
	if r.Code != 201 || r.Header().Get("Location") == "" {
		t.Fatalf("activity %d %s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/v1/trips/"+tr.ID+"/itinerary", nil))
	if r.Code != 200 {
		t.Fatal(r.Body.String())
	}
	var out itineraryResponse
	json.NewDecoder(r.Body).Decode(&out)
	if len(out.Days) != 1 || len(out.Days[0].Activities) != 1 || out.Days[0].Activities[0].Title != "Lunch" {
		t.Fatalf("response=%+v", out)
	}
}
func TestHTTPItineraryErrors(t *testing.T) {
	_, h, tr := httpSetup(t)
	tests := []struct {
		url, body string
		want      int
	}{{"/v1/trips/missing/itinerary", "", 404}, {"/v1/itineraries/missing/days", `{"date":"2026-06-11"}`, 404}, {"/v1/itineraries/missing/days", `{"date":`, 400}, {"/v1/itineraries/missing/days", `{"date":"2026-06-11","x":true}`, 400}, {"/v1/itineraries/missing/days", `{"date":"2026-06-11"} {}`, 400}, {"/v1/trips/" + tr.ID + "/itinerary", "", 201}}
	for _, x := range tests {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("POST", x.url, strings.NewReader(x.body)))
		if r.Code != x.want {
			t.Errorf("%s: %d want %d: %s", x.url, r.Code, x.want, r.Body.String())
		}
	}
}

var _ = context.Background
