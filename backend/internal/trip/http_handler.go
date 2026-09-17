package trip

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/trips", h.createTrip)
	mux.HandleFunc("GET /v1/trips/{tripID}", h.getTrip)
}

type createTripRequest struct {
	Name        string `json:"name"`
	Destination string `json:"destination"`
	StartsOn    string `json:"startsOn"`
	EndsOn      string `json:"endsOn"`
}

type tripResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Destination string `json:"destination"`
	StartsOn    string `json:"startsOn"`
	EndsOn      string `json:"endsOn"`
	CreatedAt   string `json:"createdAt"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (h *HTTPHandler) createTrip(w http.ResponseWriter, r *http.Request) {
	var request createTripRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", jsonErrorMessage(err), "")
		return
	}
	if err := ensureSingleJSONObject(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), "")
		return
	}

	createdTrip, err := h.service.CreateTrip(r.Context(), CreateInput(request))
	if err != nil {
		if field, ok := validationField(err); ok {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), field)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "unable to create trip", "")
		return
	}

	w.Header().Set("Location", "/v1/trips/"+createdTrip.ID)
	writeJSON(w, http.StatusCreated, responseFromTrip(createdTrip))
}

func (h *HTTPHandler) getTrip(w http.ResponseWriter, r *http.Request) {
	trip, err := h.service.GetTrip(r.Context(), r.PathValue("tripID"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "trip not found", "")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "unable to retrieve trip", "")
		return
	}
	writeJSON(w, http.StatusOK, responseFromTrip(trip))
}

func ensureSingleJSONObject(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}
		return errors.New("request body contains invalid JSON")
	}
	return nil
}

func jsonErrorMessage(err error) string {
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		return fmt.Sprintf("%s must be a string", typeError.Field)
	}
	if strings.HasPrefix(err.Error(), "json: unknown field ") {
		return "request body contains an unexpected field"
	}
	return "request body contains invalid JSON"
}

func validationField(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrNameRequired):
		return "name", true
	case errors.Is(err, ErrDestinationRequired):
		return "destination", true
	case errors.Is(err, ErrStartsOnRequired):
		return "startsOn", true
	case errors.Is(err, ErrEndsOnRequired):
		return "endsOn", true
	case errors.Is(err, ErrStartsOnInvalid):
		return "startsOn", true
	case errors.Is(err, ErrEndsOnInvalid):
		return "endsOn", true
	case errors.Is(err, ErrEndsOnBeforeStart):
		return "endsOn", true
	default:
		return "", false
	}
}

func responseFromTrip(trip Trip) tripResponse {
	return tripResponse{
		ID:          trip.ID,
		Name:        trip.Name,
		Destination: trip.Destination,
		StartsOn:    trip.StartsOn.String(),
		EndsOn:      trip.EndsOn.String(),
		CreatedAt:   trip.CreatedAt.Format(time.RFC3339Nano),
	}
}

func writeError(w http.ResponseWriter, status int, code, message, field string) {
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message, Field: field}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
