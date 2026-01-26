package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gterranova/normattiva-search/internal/converter"
	"github.com/gterranova/normattiva-search/internal/normattiva"
)

type Handler struct {
	client *normattiva.Client
}

func NewHandler(client *normattiva.Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	results, err := h.client.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		return
	}

	query := r.URL.Query()
	id := query.Get("id")
	date := query.Get("date")
	urn := query.Get("urn")
	format := query.Get("format")

	if urn != "" {
		var err error
		id, date, err = h.client.ResolveURN(urn)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to resolve URN: %v", err), http.StatusNotFound)
			return
		}
	}

	if id == "" || date == "" {
		http.Error(w, "Missing 'id' and 'date' or 'urn' parameters", http.StatusBadRequest)
		return
	}

	xmlContent, err := h.client.FetchXML(id, date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Document-Id", id)
	w.Header().Set("X-Document-Date", date)

	// Extract and set Title
	if title, err := converter.ExtractTitle(xmlContent); err == nil && title != "" {
		w.Header().Set("X-Document-Title", title)
	}

	if format == "markdown" {
		md, err := converter.ToMarkdown(xmlContent)
		if err != nil {
			http.Error(w, "Conversion failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte(md))
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write(xmlContent)
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Expose-Headers", "X-Document-Id, X-Document-Date, X-Document-Title")
}
