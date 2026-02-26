package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/siherrmann/slicer/handler"
)

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := handler.NewViewerHandler()

	r.Get("/", h.HandleView)
	r.Post("/slice", h.HandleSlice)
	r.Post("/export", h.HandleExport)

	log.Println("Starting server on http://localhost:4000...")

	server := &http.Server{
		Addr:         ":4000",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
