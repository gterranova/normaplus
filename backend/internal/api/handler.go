package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gterranova/normaplus/backend/eurlex"
	"github.com/gterranova/normaplus/backend/internal/ai"
	"github.com/gterranova/normaplus/backend/internal/export"
	"github.com/gterranova/normaplus/backend/internal/store"
	"github.com/gterranova/normaplus/backend/normattiva"
	"github.com/gterranova/normaplus/backend/normattiva/document"
)

type Handler struct {
	client        *normattiva.Client
	euClient      *eurlex.Client
	store         *store.Store
	aiService     *ai.Service
	exportService *export.Service
}

func NewHandler(client *normattiva.Client, euClient *eurlex.Client, store *store.Store, aiService *ai.Service, exportService *export.Service) *Handler {
	return &Handler{
		client:        client,
		euClient:      euClient,
		store:         store,
		aiService:     aiService,
		exportService: exportService,
	}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	src, ok := sorgente(r)
	if !ok {
		http.Error(w, "Parametro 'source' non valido: usare 'normattiva' oppure 'eurlex'", http.StatusBadRequest)
		return
	}
	if src == "eurlex" {
		h.searchEU(w, r, query)
		return
	}

	results, err := h.client.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Stamped on every result so the client can tell an Italian act from an EU
	// one without inspecting the identifier — the two live side by side in the
	// history, in bookmarks and in annotations.
	out := make([]searchResult, 0, len(results))
	for _, r := range results {
		out = append(out, searchResult{
			Title:                     r.Title,
			DataPubblicazioneGazzetta: r.DataPubblicazioneGazzetta,
			CodiceRedazionale:         r.CodiceRedazionale,
			Link:                      r.Link,
			Source:                    "normattiva",
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// searchResult is normattiva.DocumentMetadata plus the source.
//
// It exists so both sources answer in one shape: the frontend keys history,
// bookmarks and annotations on codice_redazionale, and the CELEX takes that slot
// for an EU act — so `source` is the only thing that has to travel alongside.
type searchResult struct {
	Title                     string `json:"title"`
	DataPubblicazioneGazzetta string `json:"data_pubblicazione_gazzetta"`
	CodiceRedazionale         string `json:"codice_redazionale"`
	Link                      string `json:"link,omitempty"`
	Source                    string `json:"source"`
}

func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		return
	}

	query := r.URL.Query()
	id := query.Get("id")
	date := query.Get("date")
	vigenza := query.Get("vigenza")
	urn := query.Get("urn")
	format := query.Get("format")
	name := ""

	src, ok := sorgente(r)
	if !ok {
		http.Error(w, "Parametro 'source' non valido: usare 'normattiva' oppure 'eurlex'", http.StatusBadRequest)
		return
	}

	var doc *document.Document
	var err error

	switch {
	case src == "eurlex":
		// An EU act has no vigenza and no URN: its identity is the CELEX, which
		// travels in `id` exactly as a codice redazionale does.
		if id == "" {
			http.Error(w, "Parametro 'id' mancante: per source=eurlex indicare il numero CELEX (es. 32022L2555)", http.StatusBadRequest)
			return
		}
		doc, err = h.euClient.Fetch(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	case urn != "":
		doc, err = h.client.FetchByURN(urn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case id != "":
		doc, err = h.client.Fetch(id, name, date, vigenza)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Missing/wrong 'id' or 'urn' parameters", http.StatusBadRequest)
		return
	}

	w.Header().Set("X-Document-Id", doc.CodiceRedazionale)
	w.Header().Set("X-Document-Date", doc.DataGU)
	w.Header().Set("X-Document-Vigenza", doc.Vigenza)
	w.Header().Set("X-Document-Name", doc.Name)
	// The client navigates from a link inside a document and has to know which
	// source to ask next; without this it would have to guess from the shape of
	// the identifier.
	w.Header().Set("X-Document-Source", src)

	switch format {
	case "json":
		jsonBytes, err := doc.ToJSON()
		if err != nil {
			http.Error(w, "Conversion failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBytes)
	case "markdown":
		md, err := doc.ToMarkdown()
		if err != nil {
			http.Error(w, "Conversion failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte(md))
	default:
		http.Error(w, "Unsupported format: "+format, http.StatusBadRequest)
		return
	}
}

// --- User Handlers ---

func (h *Handler) HandleUsers(w http.ResponseWriter, r *http.Request) {
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
			// Absent means normattiva, so a client written before EUR-Lex
			// existed goes on bookmarking Italian acts unchanged.
			Source string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		bm, err := h.store.CreateBookmark(ctx, userID, body.DocID, body.Title, body.Date, body.Source)
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
	if r.Method == "OPTIONS" {
		return
	}

	query := r.URL.Query()
	id := query.Get("id")
	date := query.Get("date")
	vigenza := query.Get("vigenza")
	format := query.Get("format") // pdf, docx, html, md

	src, ok := sorgente(r)
	if !ok {
		http.Error(w, "Parametro 'source' non valido: usare 'normattiva' oppure 'eurlex'", http.StatusBadRequest)
		return
	}

	if format == "" {
		format = "pdf"
	}

	// An EU act is identified by its CELEX alone: it has no gazzetta date and no
	// vigenza, so requiring them here would make every export of one fail — and
	// fail as a Normattiva lookup, which names the wrong archive in the error.
	var (
		doc *document.Document
		err error
	)
	switch {
	case src == "eurlex":
		if id == "" {
			http.Error(w, "Parametro 'id' mancante: per source=eurlex indicare il numero CELEX (es. 32022L2555)", http.StatusBadRequest)
			return
		}
		doc, err = h.euClient.Fetch(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
	default:
		if id == "" || date == "" {
			http.Error(w, "Missing id/date", http.StatusBadRequest)
			return
		}
		doc, err = h.client.Fetch(id, "", date, vigenza)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	md, err := doc.ToMarkdown()
	if err != nil {
		http.Error(w, "Conversion failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data, contentType, err := h.exportService.Export(string(md), format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"document_%s.%s\"", id, format))
	w.Write(data)
}
