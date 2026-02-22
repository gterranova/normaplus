package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gterranova/normaplus/backend/internal/ai"
	"github.com/gterranova/normaplus/backend/internal/api"
	"github.com/gterranova/normaplus/backend/internal/assets"
	"github.com/gterranova/normaplus/backend/internal/export"
	"github.com/gterranova/normaplus/backend/internal/store"
	"github.com/gterranova/normaplus/backend/normattiva"
)

// corsMiddleware adds CORS headers to allow frontend access
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from frontend
		allowedOrigin := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigin == "" {
			allowedOrigin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
		//w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "X-Document-Id, X-Document-Date, X-Document-Name")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	// Initialize Store
	store, err := store.NewStore("")
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	// Initialize AI
	aiService := ai.NewService()

	// Initialize Export
	exportService := export.NewService()

	client := normattiva.NewClient(30 * time.Second)
	handler := api.NewHandler(client, store, aiService, exportService)

	http.HandleFunc("/api/search", corsMiddleware(handler.Search))
	http.HandleFunc("/api/document", corsMiddleware(handler.GetDocument))

	// New routes
	http.HandleFunc("/api/users", corsMiddleware(handler.HandleUsers))
	http.HandleFunc("/api/bookmarks", corsMiddleware(handler.HandleBookmarks))
	http.HandleFunc("/api/annotations", corsMiddleware(handler.HandleAnnotations))
	http.HandleFunc("/api/ai/generate", corsMiddleware(handler.HandleAIGenerate))
	http.HandleFunc("/api/export", corsMiddleware(handler.HandleExport))

	// Serve static files from the embedded filesystem
	staticFS := assets.GetFileSystem()
	fileServer := http.FileServer(staticFS)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API routes are handled by specific handlers due to longer prefixes.
		// This handler catches everything else.

		// For the root path, serve index.html
		if r.URL.Path == "/" || r.URL.Path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Check if the file exists in the static FS
		// We need to strip the leading slash for the FS
		filePath := r.URL.Path[1:]
		f, err := staticFS.Open(filePath)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// If file doesn't exist, it might be an SPA route
		// Fallback to index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	port := "8080"
	log.Printf("Server running on http://0.0.0.0:%s", port)
	if err := http.ListenAndServe("0.0.0.0:"+port, nil); err != nil {
		log.Fatal(err)
	}
}
