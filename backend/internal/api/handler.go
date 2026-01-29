package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gterranova/normattiva-search/internal/ai"
	"github.com/gterranova/normattiva-search/internal/converter"
	"github.com/gterranova/normattiva-search/internal/export"
	"github.com/gterranova/normattiva-search/internal/normattiva"
	"github.com/gterranova/normattiva-search/internal/store"
)

type Handler struct {
	client        *normattiva.Client
	store         *store.Store
	aiService     *ai.Service
	exportService *export.Service
}

func NewHandler(client *normattiva.Client, store *store.Store, aiService *ai.Service, exportService *export.Service) *Handler {
	return &Handler{
		client:        client,
		store:         store,
		aiService:     aiService,
		exportService: exportService,
	}
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
	vigenza := query.Get("vigenza")
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

	xmlContent, err := h.client.FetchXML(id, date, vigenza)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Document-Id", id)
	w.Header().Set("X-Document-Date", date)
	w.Header().Set("X-Document-Vigenza", vigenza)

	// Extract and set Title
	if title, err := converter.ExtractTitle(xmlContent); err == nil && title != "" {
		w.Header().Set("X-Document-Title", title)
	}

	if format == "markdown" {
		md, err := converter.ToMarkdown(xmlContent, vigenza)
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

// --- User Handlers ---

func (h *Handler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		return
	}

	ctx := r.Context()

	if r.Method == "GET" {
		users, err := h.store.ListUsers(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(users)
		return
	}

	if r.Method == "POST" {
		var body struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		user, err := h.store.CreateUser(ctx, body.Name, body.Color)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(user)
		return
	}

	if r.Method == "PUT" {
		var u store.User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateUser(ctx, &u); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// --- Bookmark Handlers ---

func (h *Handler) HandleBookmarks(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		return
	}

	ctx := r.Context()
	userIDStr := r.URL.Query().Get("userId")
	if userIDStr == "" {
		http.Error(w, "Missing userId", http.StatusBadRequest)
		return
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid userId", http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		bookmarks, err := h.store.ListBookmarks(ctx, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(bookmarks)
		return
	}

	if r.Method == "POST" {
		var body struct {
			DocID string `json:"doc_id"`
			Title string `json:"title"`
			Date  string `json:"date"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		bm, err := h.store.CreateBookmark(ctx, userID, body.DocID, body.Title, body.Date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(bm)
		return
	}

	if r.Method == "DELETE" {
		docID := r.URL.Query().Get("docId")
		if docID == "" {
			http.Error(w, "Missing docId", http.StatusBadRequest)
			return
		}
		if err := h.store.DeleteBookmark(ctx, userID, docID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == "PUT" {
		var body struct {
			DocID    string `json:"doc_id"`
			Category string `json:"category"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateBookmarkCategory(ctx, userID, body.DocID, body.Category); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// --- Annotation Handlers ---

func (h *Handler) HandleAnnotations(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		return
	}

	ctx := r.Context()

	if r.Method == "GET" {
		userIDStr := r.URL.Query().Get("userId")
		docID := r.URL.Query().Get("docId")

		if userIDStr == "" {
			http.Error(w, "Missing userId", http.StatusBadRequest)
			return
		}
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w, "Invalid userId", http.StatusBadRequest)
			return
		}

		annotations, err := h.store.ListAnnotations(ctx, userID, docID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(annotations)
		return
	}

	if r.Method == "POST" {
		var body struct {
			UserID          int    `json:"user_id"`
			DocID           string `json:"doc_id"`
			SelectionData   string `json:"selection_data"`
			LocationID      string `json:"location_id"`
			SelectionOffset int    `json:"selection_offset"`
			Prefix          string `json:"prefix"`
			Suffix          string `json:"suffix"`
			Comment         string `json:"comment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		ann, err := h.store.CreateAnnotation(ctx, body.UserID, body.DocID, body.SelectionData, body.LocationID, body.Prefix, body.Suffix, body.SelectionOffset, body.Comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(ann)
		return
	}

	if r.Method == "DELETE" {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "Missing id", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid id", http.StatusBadRequest)
			return
		}
		if err := h.store.DeleteAnnotation(ctx, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == "PUT" {
		var body struct {
			ID      int    `json:"id"`
			Comment string `json:"comment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateAnnotation(ctx, body.ID, body.Comment); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) HandleAIGenerate(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Action     string `json:"action"` // "summarize" or "translate"
		Text       string `json:"text"`
		TargetLang string `json:"targetLang"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var result string
	var err error

	switch body.Action {
	case "summarize":
		result, err = h.aiService.Summarize(r.Context(), body.Text)
	case "translate":
		lang := body.TargetLang
		if lang == "" {
			lang = "English"
		}
		result, err = h.aiService.Translate(r.Context(), body.Text, lang)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"result": result})
}
func (h *Handler) HandleExport(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		return
	}

	query := r.URL.Query()
	id := query.Get("id")
	date := query.Get("date")
	vigenza := query.Get("vigenza")
	format := query.Get("format") // pdf, docx, html, md

	if id == "" || date == "" {
		http.Error(w, "Missing id/date", http.StatusBadRequest)
		return
	}

	if format == "" {
		format = "pdf"
	}

	xmlContent, err := h.client.FetchXML(id, date, vigenza)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	md, err := converter.ToMarkdown(xmlContent, vigenza)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, contentType, err := h.exportService.Export(md, format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"document_%s.%s\"", id, format))
	w.Write(data)
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Expose-Headers", "X-Document-Id, X-Document-Date, X-Document-Title")
}
