package main

import (
	"log"
	"net/http"

	"github.com/SoundaryaLahari-K/trippie/internal/itinerary"
	"github.com/SoundaryaLahari-K/trippie/internal/place"
	"github.com/SoundaryaLahari-K/trippie/internal/planner"
	"github.com/SoundaryaLahari-K/trippie/internal/trip"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	tripRepository := trip.NewMemoryRepository()
	tripService := trip.NewService(tripRepository)
	tripHandler := trip.NewHTTPHandler(tripService)
	tripHandler.RegisterRoutes(mux)

	dayRepository := itinerary.NewMemoryDayRepository()
	activityRepository := itinerary.NewMemoryActivityRepository()
	itineraryHandler := itinerary.NewHTTPHandler(itinerary.NewService(tripRepository, itinerary.NewMemoryItineraryRepository(), dayRepository, activityRepository))
	itineraryHandler.RegisterRoutes(mux)
	planner.NewHTTPHandler(planner.NewService(dayRepository, activityRepository, place.SeedDemoRepository())).RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("GapTrip API listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
