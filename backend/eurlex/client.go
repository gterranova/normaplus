// Package eurlex reads EU legislation and the national measures that transpose
// it, from the Publications Office's own endpoints.
//
// It is the sibling of package normattiva and answers in the same shapes, so the
// API, the viewer, the exporter and the annotation layer work on an EU act
// exactly as they do on an Italian one.
//
// # Two endpoints, and only one of them is usable
//
// **eur-lex.europa.eu cannot be read by a program.** Measured 2026-09-11: a
// request for the HTML of CELEX 32022L2555 answers **HTTP 202 with an empty
// body** — a WAF challenge meant for a browser. So every address on that host
// here is emitted for a human to open and never fetched.
//
// What is read instead is CELLAR (publications.europa.eu): a SPARQL endpoint
// over the CDM ontology to find acts and their transpositions, and content
// negotiation on an expression URI to read one. That is the whole of the
// difference between this package working and returning nothing.
package eurlex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gterranova/normaplus/backend/normattiva/document"
)

// cellarSPARQL is a var for one reason: a test points it at an httptest server.
// Every other use is read-only.
var cellarSPARQL = "https://publications.europa.eu/webapi/rdf/sparql"

const (
	// eurLexBase builds the citable page for a reader. NEVER fetched: see the
	// package comment.
	eurLexBase = "https://eur-lex.europa.eu/legal-content/IT/TXT/?uri=CELEX:"

	langITA = "http://publications.europa.eu/resource/authority/language/ITA"

	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// The SPARQL endpoint is slow in proportion to how common the words are, not
	// to how many rows come back: a keyword filter is a CONTAINS over every
	// Italian title in the corpus, which it cannot serve from an index.
	defaultTimeout = 90 * time.Second
)

// Client reads EU acts. Its zero value is not usable; call NewClient.
type Client struct {
	httpClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{httpClient: &http.Client{Timeout: timeout}}
}

// DocumentMetadata is one EU act in a result list.
//
// The field names mirror normattiva.DocumentMetadata on purpose: the CELEX *is*
// the redactional code of an EU act, and the OJ date is its publication date, so
// the existing history, bookmarks and annotations key on them unchanged. Source
// is what tells the two apart where it matters.
type DocumentMetadata struct {
	Title                     string `json:"title"`
	DataPubblicazioneGazzetta string `json:"data_pubblicazione_gazzetta"`
	CodiceRedazionale         string `json:"codice_redazionale"`
	Link                      string `json:"link,omitempty"`
	Source                    string `json:"source"`
	Tipo                      string `json:"tipo,omitempty"`
}

// --- SPARQL ------------------------------------------------------------------

type sparqlBinding map[string]struct {
	Value string `json:"value"`
}

func (b sparqlBinding) get(k string) string { return b[k].Value }

type sparqlResponse struct {
	Results struct {
		Bindings []sparqlBinding `json:"bindings"`
	} `json:"results"`
}

// literal escapes a caller's term into a SPARQL string literal.
//
// The term reaches the endpoint as query *syntax*, so an unescaped quotation
// mark is not a bad search — it is a malformed query and a 400 the caller cannot
// explain. An apostrophe is legal inside "..." and is left alone: it is ordinary
// in Italian legal prose, and removing it would make a phrase match nothing.
func literal(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	).Replace(s)
}

func (c *Client) sparql(query string) ([]sparqlBinding, error) {
	form := url.Values{"query": {query}}
	req, err := http.NewRequest(http.MethodPost, cellarSPARQL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("endpoint SPARQL CELLAR non raggiungibile: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("endpoint SPARQL CELLAR: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out sparqlResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("risposta SPARQL non interpretabile: %w", err)
	}
	return out.Results.Bindings, nil
}

// --- search ------------------------------------------------------------------

const prefissi = `PREFIX cdm: <http://publications.europa.eu/ontology/cdm#>
PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
`

// Search finds EU acts whose Italian title contains the words given.
//
// WHAT IT SEARCHES: the title, not the text. An EU act's title states its
// subject in full, so this is a real index — but an empty result means "no act
// is titled that way", never "the Union has not legislated on it". The caller is
// expected to say so; the API and the tool both do.
//
// A reference that is already a CELEX short-circuits to a lookup, because that
// is what a reader pasting "32022L2555" means and a title search for it finds
// nothing.
func (c *Client) Search(query string) ([]DocumentMetadata, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("la ricerca richiede almeno un termine")
	}

	// A CELEX, or a citation that yields one, is an address rather than a query.
	if celex, ok := CelexDaRiferimento(q); ok {
		meta, err := c.Metadata(celex)
		if err != nil {
			return nil, err
		}
		if meta != nil {
			return []DocumentMetadata{*meta}, nil
		}
		// Not an act CELLAR holds: fall through and search the words instead, so
		// a reference that merely looked like a citation still finds something.
	}

	var conds []string
	for _, term := range strings.Fields(q) {
		conds = append(conds, fmt.Sprintf(`CONTAINS(LCASE(STR(?title)), "%s")`, literal(strings.ToLower(term))))
	}

	// Two decisions in this query, and each was measured.
	//
	// The CELEX pattern is ANCHORED to a directive or a regulation and nothing
	// else. Unanchored, a search for "cibersicurezza" comes back led by
	// "Rettifica del regolamento…" — corrigenda carry a CELEX like
	// 32024R2847R(04), they are the most recently dated documents about any act,
	// and they are not the act. Measured: six of the first ten results were
	// corrigenda, and anchoring also cut the query from 30s to 20s.
	//
	// ORDER BY stays despite costing about ten of those seconds. Without it the
	// LIMIT returns an arbitrary twenty-five rows rather than the twenty-five
	// most recent, and sorting those afterwards would present an arbitrary sample
	// as if it were the latest — which is the kind of wrong a date column makes
	// invisible.
	sparqlQuery := prefissi + fmt.Sprintf(`
SELECT DISTINCT ?celex ?title ?date ?tipo
WHERE {
  ?work cdm:resource_legal_id_celex ?celex .
  ?work cdm:work_date_document ?date .
  OPTIONAL { ?work cdm:work_has_resource-type ?tipo . }
  ?exp cdm:expression_belongs_to_work ?work .
  ?exp cdm:expression_uses_language <%s> .
  ?exp cdm:expression_title ?title .
  FILTER(REGEX(STR(?celex), "^3[0-9]{4}[LR][0-9]{4}$"))
  FILTER(%s)
}
ORDER BY DESC(?date)
LIMIT 25`, langITA, strings.Join(conds, " && "))

	bindings, err := c.sparql(sparqlQuery)
	if err != nil {
		return nil, err
	}

	// One work has several Italian expressions in CELLAR — different
	// manifestations of one act — so it comes back once per expression.
	// Undeduplicated, a result list shows the same directive three times.
	seen := map[string]bool{}
	var out []DocumentMetadata
	for _, b := range bindings {
		celex := b.get("celex")
		if celex == "" || seen[celex] {
			continue
		}
		seen[celex] = true
		out = append(out, DocumentMetadata{
			Title:                     strings.TrimSpace(b.get("title")),
			DataPubblicazioneGazzetta: b.get("date"),
			CodiceRedazionale:         celex,
			Link:                      eurLexBase + celex,
			Source:                    "eurlex",
			Tipo:                      tipoDaURI(b.get("tipo")),
		})
	}
	return out, nil
}

// Metadata returns what CELLAR holds about one act, or nil when it holds none.
//
// nil rather than an error: "this CELEX names nothing" is an answer, and an
// error here would make a fall-back search impossible to write.
func (c *Client) Metadata(celex string) (*DocumentMetadata, error) {
	celex = strings.ToUpper(strings.TrimSpace(celex))
	if celex == "" {
		return nil, fmt.Errorf("CELEX mancante")
	}
	q := prefissi + fmt.Sprintf(`
SELECT ?title ?date ?tipo
WHERE {
  ?work cdm:resource_legal_id_celex "%s"^^xsd:string .
  OPTIONAL { ?work cdm:work_date_document ?date . }
  OPTIONAL { ?work cdm:work_has_resource-type ?tipo . }
  OPTIONAL {
    ?exp cdm:expression_belongs_to_work ?work .
    ?exp cdm:expression_uses_language <%s> .
    ?exp cdm:expression_title ?title .
  }
}
LIMIT 1`, literal(celex), langITA)

	bindings, err := c.sparql(q)
	if err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	b := bindings[0]
	return &DocumentMetadata{
		Title:                     strings.TrimSpace(b.get("title")),
		DataPubblicazioneGazzetta: b.get("date"),
		CodiceRedazionale:         celex,
		Link:                      eurLexBase + celex,
		Source:                    "eurlex",
		Tipo:                      tipoDaURI(b.get("tipo")),
	}, nil
}

// tipoDaURI reduces a CDM resource-type URI to its last segment, which is the
// readable name ("DIR", "REG", "DIR_IMPL").
func tipoDaURI(uri string) string {
	if uri == "" {
		return ""
	}
	return uri[strings.LastIndexAny(uri, "/#")+1:]
}

// --- fetch -------------------------------------------------------------------

// Fetch returns an EU act as a Document — the same type package normattiva
// returns, so the viewer, the table of contents and every exporter work on it
// unchanged.
func (c *Client) Fetch(celex string) (*document.Document, error) {
	celex = strings.ToUpper(strings.TrimSpace(celex))
	if celex == "" {
		return nil, fmt.Errorf("CELEX mancante")
	}

	meta, err := c.Metadata(celex)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("nessun atto UE con CELEX %s", celex)
	}

	expr, err := c.espressioneItaliana(celex)
	if err != nil {
		return nil, err
	}
	if expr == "" {
		return nil, fmt.Errorf("l'atto %s non ha una versione in lingua italiana in CELLAR", celex)
	}

	html, err := c.contenuto(expr)
	if err != nil {
		return nil, err
	}

	doc := document.NewDocument(celex, meta.Title, meta.DataPubblicazioneGazzetta, "")
	if err := parseAtto(html, &doc); err != nil {
		return nil, err
	}
	doc.Title = meta.Title
	return &doc, nil
}

func (c *Client) espressioneItaliana(celex string) (string, error) {
	q := prefissi + fmt.Sprintf(`
SELECT ?exp WHERE {
  ?work cdm:resource_legal_id_celex "%s"^^xsd:string .
  ?exp cdm:expression_belongs_to_work ?work .
  ?exp cdm:expression_uses_language <%s> .
} LIMIT 1`, literal(celex), langITA)

	bindings, err := c.sparql(q)
	if err != nil {
		return "", err
	}
	if len(bindings) == 0 {
		return "", nil
	}
	return bindings[0].get("exp"), nil
}

// contenuto asks CELLAR for the expression as XHTML.
//
// Redirects are followed because CELLAR answers an expression URI with one to
// the manifestation that actually holds the bytes.
func (c *Client) contenuto(expr string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, expr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "it")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("recupero del testo da CELLAR: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("recupero del testo da CELLAR: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// LinkEurLex is the citable page for a human reader.
//
// Returned and never followed: see the package comment. A caller that fetches it
// gets an empty 202 and would report a perfectly good act as unavailable.
func LinkEurLex(celex string) string {
	celex = strings.ToUpper(strings.TrimSpace(celex))
	if celex == "" {
		return ""
	}
	return eurLexBase + celex
}
