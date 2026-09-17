package planner

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/SoundaryaLahari-K/trippie/internal/itinerary"
	"github.com/SoundaryaLahari-K/trippie/internal/place"
)

type HTTPHandler struct{ service *Service }

func NewHTTPHandler(s *Service) *HTTPHandler { return &HTTPHandler{service: s} }
func (h *HTTPHandler) RegisterRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /v1/days/{dayID}/gaps", h.gaps)
	m.HandleFunc("GET /v1/days/{dayID}/suggestions", h.suggestions)
}

type gapResponse struct {
	DayID           string               `json:"dayId"`
	StartTime       string               `json:"startTime"`
	EndTime         string               `json:"endTime"`
	DurationMinutes int                  `json:"durationMinutes"`
	Suggestions     []suggestionResponse `json:"suggestions,omitempty"`
}
type suggestionResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Score      float64  `json:"score"`
	DistanceKM float64  `json:"distanceKm"`
	Reasons    []string `json:"reasons"`
}

func (h *HTTPHandler) gaps(w http.ResponseWriter, r *http.Request) {
	gaps, err := h.service.Gaps(r.Context(), r.PathValue("dayID"))
	if err != nil {
		h.error(w, err)
		return
	}
	write(w, http.StatusOK, map[string]any{"gaps": gapResponses(gaps, nil)})
}
func (h *HTTPHandler) suggestions(w http.ResponseWriter, r *http.Request) {
	req, err := parseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	gaps, err := h.service.Gaps(r.Context(), r.PathValue("dayID"))
	if err != nil {
		h.error(w, err)
		return
	}
	all, err := h.service.Suggestions(r.Context(), r.PathValue("dayID"), req)
	if err != nil {
		h.error(w, err)
		return
	}
	byGap := map[string][]Suggestion{}
	for _, s := range all {
		byGap[s.Gap.StartTime.String()+s.Gap.EndTime.String()] = append(byGap[s.Gap.StartTime.String()+s.Gap.EndTime.String()], s)
	}
	write(w, http.StatusOK, map[string]any{"gaps": gapResponses(gaps, byGap)})
}
func parseRequest(r *http.Request) (DiscoveryRequest, error) {
	q := r.URL.Query()
	req := DiscoveryRequest{Origin: place.Location{Latitude: 48.8566, Longitude: 2.3522}}
	for _, key := range []string{"latitude", "longitude", "maxDistanceKm"} {
		if q.Get(key) != "" {
			v, e := strconv.ParseFloat(q.Get(key), 64)
			if e != nil {
				return req, errors.New("invalid " + key)
			}
			switch key {
			case "latitude":
				req.Origin.Latitude = v
			case "longitude":
				req.Origin.Longitude = v
			case "maxDistanceKm":
				if v < 0 {
					return req, errors.New("invalid maxDistanceKm")
				}
				req.MaximumDistanceKM = v
			}
		}
	}
	if raw := q.Get("categories"); raw != "" {
		for _, v := range strings.Split(raw, ",") {
			c := place.Category(strings.ToUpper(strings.TrimSpace(v)))
			if c != place.Cafe && c != place.Restaurant && c != place.Spa && c != place.Market && c != place.Experience {
				return req, errors.New("invalid categories")
			}
			req.Categories = append(req.Categories, c)
		}
	}
	return req, nil
}
func gapResponses(gaps []Gap, grouped map[string][]Suggestion) []gapResponse {
	out := make([]gapResponse, 0, len(gaps))
	for _, g := range gaps {
		end := g.EndTime.String()
		if end == "" {
			end = "24:00"
		}
		x := gapResponse{DayID: g.DayID, StartTime: g.StartTime.String(), EndTime: end, DurationMinutes: g.DurationMinutes}
		for _, s := range grouped[g.StartTime.String()+g.EndTime.String()] {
			x.Suggestions = append(x.Suggestions, suggestionResponse{s.ID, s.Place.Name, string(s.Place.Category), s.Score, s.DistanceKM, s.Reasons})
		}
		out = append(out, x)
	}
	return out
}
func (h *HTTPHandler) error(w http.ResponseWriter, err error) {
	if errors.Is(err, itinerary.ErrNotFound) {
		http.Error(w, "resource not found", http.StatusNotFound)
		return
	}
	http.Error(w, "unable to process planner", http.StatusInternalServerError)
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
