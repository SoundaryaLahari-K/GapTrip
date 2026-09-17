package itinerary

import (
	"encoding/json"
	"errors"
	"github.com/SoundaryaLahari-K/trippie/internal/trip"
	"io"
	"net/http"
)

type HTTPHandler struct{ service *Service }

func NewHTTPHandler(s *Service) *HTTPHandler { return &HTTPHandler{service: s} }
func (h *HTTPHandler) RegisterRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /v1/trips/{tripID}/itinerary", h.create)
	m.HandleFunc("GET /v1/trips/{tripID}/itinerary", h.get)
	m.HandleFunc("POST /v1/itineraries/{itineraryID}/days", h.addDay)
	m.HandleFunc("POST /v1/days/{dayID}/activities", h.addActivity)
}

type errResponse struct {
	Error apiError `json:"error"`
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}
type itineraryResponse struct {
	ID     string        `json:"id"`
	TripID string        `json:"tripId"`
	Days   []dayResponse `json:"days"`
}
type dayResponse struct {
	ID         string             `json:"id"`
	Date       string             `json:"date"`
	Activities []activityResponse `json:"activities"`
}
type activityResponse struct {
	ID        string `json:"id"`
	DayID     string `json:"dayId"`
	Title     string `json:"title"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Location  string `json:"location"`
	Fixed     bool   `json:"fixed"`
}

func (h *HTTPHandler) create(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.CreateItinerary(r.Context(), r.PathValue("tripID"))
	if e != nil {
		h.error(w, e)
		return
	}
	w.Header().Set("Location", "/v1/trips/"+v.TripID+"/itinerary")
	writeJSON(w, http.StatusCreated, itineraryResponse{ID: v.ID, TripID: v.TripID, Days: []dayResponse{}})
}
func (h *HTTPHandler) get(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.GetItineraryForTrip(r.Context(), r.PathValue("tripID"))
	if e != nil {
		h.error(w, e)
		return
	}
	writeJSON(w, http.StatusOK, response(v))
}
func (h *HTTPHandler) addDay(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Date string `json:"date"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.service.AddDay(r.Context(), r.PathValue("itineraryID"), in.Date)
	if e != nil {
		h.error(w, e)
		return
	}
	w.Header().Set("Location", "/v1/itineraries/"+v.ItineraryID+"/days/"+v.ID)
	writeJSON(w, http.StatusCreated, dayResponse{ID: v.ID, Date: v.Date.String(), Activities: []activityResponse{}})
}
func (h *HTTPHandler) addActivity(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title     string `json:"title"`
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
		Location  string `json:"location"`
		Fixed     bool   `json:"fixed"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, e := h.service.AddActivity(r.Context(), r.PathValue("dayID"), ActivityInput{Title: in.Title, StartTime: in.StartTime, EndTime: in.EndTime, Location: in.Location, Fixed: in.Fixed})
	if e != nil {
		h.error(w, e)
		return
	}
	w.Header().Set("Location", "/v1/days/"+v.DayID+"/activities/"+v.ID)
	writeJSON(w, http.StatusCreated, activity(v))
}
func (h *HTTPHandler) error(w http.ResponseWriter, e error) {
	if errors.Is(e, ErrNotFound) || errors.Is(e, trip.ErrNotFound) {
		writeError(w, 404, "not_found", "resource not found", "")
		return
	}
	if f := field(e); f != "" {
		writeError(w, 400, "validation_error", e.Error(), f)
		return
	}
	writeError(w, 500, "internal_error", "unable to process itinerary", "")
}
func field(e error) string {
	switch {
	case errors.Is(e, ErrDayDateInvalid), errors.Is(e, ErrDayDateRequired), errors.Is(e, ErrDayOutsideTrip):
		return "date"
	case errors.Is(e, ErrActivityTitleRequired):
		return "title"
	case errors.Is(e, ErrActivityStartRequired), errors.Is(e, ErrActivityTimeInvalid):
		return "startTime"
	case errors.Is(e, ErrActivityEndRequired), errors.Is(e, ErrActivityEndBeforeStart):
		return "endTime"
	case errors.Is(e, ErrActivityLocationTooLong):
		return "location"
	}
	return ""
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		writeError(w, 400, "invalid_request", "request body contains invalid JSON", "")
		return false
	}
	var x any
	if e := d.Decode(&x); !errors.Is(e, io.EOF) {
		writeError(w, 400, "invalid_request", "request body must contain a single JSON object", "")
		return false
	}
	return true
}
func response(v ItineraryView) itineraryResponse {
	out := itineraryResponse{ID: v.Itinerary.ID, TripID: v.Itinerary.TripID, Days: []dayResponse{}}
	for _, d := range v.Days {
		x := dayResponse{ID: d.Day.ID, Date: d.Day.Date.String(), Activities: []activityResponse{}}
		for _, a := range d.Activities {
			x.Activities = append(x.Activities, activity(a))
		}
		out.Days = append(out.Days, x)
	}
	return out
}
func activity(a Activity) activityResponse {
	return activityResponse{ID: a.ID, DayID: a.DayID, Title: a.Title, StartTime: a.StartTime.String(), EndTime: a.EndTime.String(), Location: a.Location, Fixed: a.Fixed}
}
func writeError(w http.ResponseWriter, s int, c, m, f string) {
	writeJSON(w, s, errResponse{apiError{c, m, f}})
}
func writeJSON(w http.ResponseWriter, s int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	_ = json.NewEncoder(w).Encode(v)
}
