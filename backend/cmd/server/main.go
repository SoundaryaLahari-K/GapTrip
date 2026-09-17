package main

import (
	"log"
	"net/http"

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

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Trippie API listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
