package main

import (
	"log"
	"net/http"

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
	err := http.ListenAndServe(":4000", r)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
