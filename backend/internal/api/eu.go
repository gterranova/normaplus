package api

// The EU half of the API: searching EUR-Lex, reading an EU act, and the
// transposition link between a directive and the national measures that
// implement it.
//
// The design decision that shapes this file: an EU act answers in the SAME
// shapes an Italian one does — DocumentMetadata for a result, Document for the
// text — so `/api/search` and `/api/document` gained a `source` parameter rather
// than gaining EU twins. Everything downstream of them (the viewer, the table of
// contents, annotations, bookmarks, every exporter) therefore works on an EU
// directive without knowing it exists.

import (
	"encoding/json"
	"net/http"
	"strings"
)

// sorgente reads the `source` parameter.
//
// Absent means normattiva, so every URL written before EUR-Lex existed keeps
// meaning what it meant. An unknown value is an error rather than a silent
// fallback: answering a request for a source that does not exist with Italian
// legislation is worse than refusing it.
func sorgente(r *http.Request) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source")))
	switch s {
	case "", "normattiva":
		return "normattiva", true
	case "eurlex", "eur-lex", "ue":
		return "eurlex", true
	}
	return s, false
}

// SearchEU answers a search over EU legislation.
//
// It is reached from Search, not routed separately, so a caller switches source
// by changing one parameter rather than one URL.
func (h *Handler) searchEU(w http.ResponseWriter, r *http.Request, query string) {
	results, err := h.euClient.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	// An empty result here means "no EU act is TITLED that way", never "the Union
	// has not legislated on it" — only titles are indexed. The header says so,
	// because a result list cannot.
	w.Header().Set("X-Search-Scope", "titolo")
	writeJSON(w, results)
}

// HandleRecepimento answers both directions of the transposition link.
//
//	GET /api/eu/recepimento?rif=direttiva+2022/2555            → national measures
//	GET /api/eu/recepimento?rif=D.Lgs.+138/2024&verso=it-eu    → the EU acts it implements
//
// Two directions on one route rather than two routes: they answer one question
// asked from either end, and the shapes returned say which end was asked.
func (h *Handler) HandleRecepimento(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	q := r.URL.Query()
	rif := strings.TrimSpace(q.Get("rif"))
	if rif == "" {
		http.Error(w, "Parametro 'rif' mancante: indicare una direttiva (es. \"direttiva 2022/2555\") oppure un atto italiano (es. \"D.Lgs. 138/2024\")", http.StatusBadRequest)
		return
	}
	paese := strings.TrimSpace(q.Get("paese"))

	switch strings.ToLower(strings.TrimSpace(q.Get("verso"))) {
	case "", "eu-it":
		out, err := h.euClient.Recepimento(rif, paese)
		if err != nil {
			// A reference that cannot be turned into a directive is the caller's
			// mistake and says what a valid one looks like, so it is a 400; a
			// CELLAR failure is not, so it is a 502.
			http.Error(w, err.Error(), statoPerErrore(err))
			return
		}
		writeJSON(w, out)
	case "it-eu":
		out, err := h.euClient.BaseUE(rif, paese)
		if err != nil {
			http.Error(w, err.Error(), statoPerErrore(err))
			return
		}
		writeJSON(w, out)
	default:
		http.Error(w, "Parametro 'verso' non valido: usare 'eu-it' (misure nazionali di una direttiva) oppure 'it-eu' (direttive recepite da un atto italiano)", http.StatusBadRequest)
	}
}

// statoPerErrore separates "you asked for something impossible" from "the source
// did not answer", because only the first is worth rewording the request for.
func statoPerErrore(err error) int {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "non si ricava"),
		strings.Contains(msg, "nessun atto UE con CELEX"),
		strings.Contains(msg, "CELEX mancante"):
		return http.StatusBadRequest
	}
	return http.StatusBadGateway
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
