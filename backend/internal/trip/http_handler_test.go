package trip

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() (*Service, http.Handler) {
	service := NewService(NewMemoryRepository())
	mux := http.NewServeMux()
	NewHTTPHandler(service).RegisterRoutes(mux)
	return service, mux
}

func TestHTTPCreateTrip(t *testing.T) {
	_, handler := newTestServer()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/trips", strings.NewReader(`{"name":"Summer break","destination":"Lisbon","startsOn":"2026-06-10","endsOn":"2026-06-15"}`))

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d; want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if got := recorder.Header().Get("Location"); !strings.HasPrefix(got, "/v1/trips/trip_") {
		t.Errorf("Location = %q; want a trip resource location", got)
	}
	var response tripResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Name != "Summer break" || response.StartsOn != "2026-06-10" {
		t.Errorf("response = %+v; want created trip", response)
	}
}

func TestHTTPGetTrip(t *testing.T) {
	service, handler := newTestServer()
	created, err := service.CreateTrip(context.Background(), CreateInput{Name: "Summer break", Destination: "Lisbon", StartsOn: "2026-06-10", EndsOn: "2026-06-15"})
	if err != nil {
		t.Fatalf("CreateTrip() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/trips/"+created.ID, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response tripResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ID != created.ID || response.Destination != "Lisbon" {
		t.Errorf("response = %+v; want %+v", response, created)
	}
}

func TestHTTPGetTripUnknownID(t *testing.T) {
	_, handler := newTestServer()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/trips/missing", nil))
	assertErrorResponse(t, recorder, http.StatusNotFound, "not_found")
}

func TestHTTPCreateTripRejectsMalformedJSON(t *testing.T) {
	_, handler := newTestServer()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/trips", strings.NewReader(`{"name":`)))
	assertErrorResponse(t, recorder, http.StatusBadRequest, "invalid_request")
}

func TestHTTPCreateTripRejectsUnexpectedFields(t *testing.T) {
	_, handler := newTestServer()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/trips", strings.NewReader(`{"name":"Summer break","destination":"Lisbon","startsOn":"2026-06-10","endsOn":"2026-06-15","extra":true}`)))
	assertErrorResponse(t, recorder, http.StatusBadRequest, "invalid_request")
}

func TestHTTPCreateTripRejectsMultipleJSONObjects(t *testing.T) {
	_, handler := newTestServer()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/trips", strings.NewReader(`{"name":"Summer break","destination":"Lisbon","startsOn":"2026-06-10","endsOn":"2026-06-15"} {}`)))
	assertErrorResponse(t, recorder, http.StatusBadRequest, "invalid_request")
}

func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d; want %d; body = %s", recorder.Code, wantStatus, recorder.Body.String())
	}
	var response errorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != wantCode {
		t.Errorf("error code = %q; want %q", response.Error.Code, wantCode)
	}
}
