package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gterranova/normattiva-search/internal/api"
	"github.com/gterranova/normattiva-search/internal/normattiva"
)

// corsMiddleware adds CORS headers to allow frontend access
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from frontend
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	client := normattiva.NewClient(30 * time.Second)
	handler := api.NewHandler(client)

	http.HandleFunc("/api/search", corsMiddleware(handler.Search))
	http.HandleFunc("/api/document", corsMiddleware(handler.GetDocument))

	port := "8080"
	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
