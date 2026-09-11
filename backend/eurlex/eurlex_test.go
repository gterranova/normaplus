package eurlex

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gterranova/normaplus/backend/normattiva/document"
)

// --- references --------------------------------------------------------------

func TestCelexDaRiferimento(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"32022L2555", "32022L2555"},
		{"direttiva 2022/2555", "32022L2555"},
		{"dir. 2019/790/UE", "32019L0790"},
		{"2019/790", "32019L0790"},
		// A regulation is R, and the number is padded to four digits.
		{"regolamento (UE) 2016/679", "32016R0679"},
		{"reg. 2016/679", "32016R0679"},
	}
	for _, c := range cases {
		got, ok := CelexDaRiferimento(c.in)
		if !ok {
			t.Errorf("CelexDaRiferimento(%q) non riconosciuto", c.in)
			continue
		}
		if got != c.want {
			t.Errorf("CelexDaRiferimento(%q) = %q, atteso %q", c.in, got, c.want)
		}
	}
}

// "regolamento di attuazione della direttiva X" is about a directive: the word
// "regolamento" appearing somewhere does not make the cited act one.
func TestARegulationIsOnlyARegulationWhenNoDirectiveIsNamed(t *testing.T) {
	got, ok := CelexDaRiferimento("regolamento di esecuzione della direttiva 2022/2555")
	if !ok {
		t.Fatal("riferimento non riconosciuto")
	}
	if !strings.Contains(got, "L") {
		t.Errorf("= %q, atteso un CELEX di direttiva", got)
	}
}

// Inventing a CELEX would query a different act and answer confidently about it.
func TestAReferenceWithNoYearAndNumberIsRefused(t *testing.T) {
	if got, ok := CelexDaRiferimento("direttiva macchine"); ok {
		t.Errorf("riferimento senza anno/numero accettato come %q", got)
	}
}

func TestRiferimentoAttoItaliano(t *testing.T) {
	cases := []struct{ in, num, anno string }{
		{"D.Lgs. 138/2024", "138", "2024"},
		{"decreto legislativo n. 138/2024", "138", "2024"},
		{"legge 90/2024", "90", "2024"},
		{"n. 138 del 4 settembre 2024", "138", "2024"},
		// The form the viewer actually has to hand: a document's own title, as
		// the Gazzetta writes it, with the date before the number. Only the
		// compact form was accepted, so every lookup started from an open act —
		// which is the only way a reader reaches this — was refused with a
		// message telling them to write the citation the other way round.
		{"DECRETO LEGISLATIVO 4 settembre 2024, n. 138", "138", "2024"},
		{"LEGGE 28 dicembre 2000, n. 445", "445", "2000"},
		{"DECRETO DEL PRESIDENTE DELLA REPUBBLICA 28 Dicembre 2000, n. 445", "445", "2000"},
	}
	for _, c := range cases {
		num, anno, ok := RiferimentoAttoItaliano(c.in)
		if !ok || num != c.num || anno != c.anno {
			t.Errorf("RiferimentoAttoItaliano(%q) = (%q,%q,%v), atteso (%q,%q,true)", c.in, num, anno, ok, c.num, c.anno)
		}
	}
}

// A measure can transpose several directives at once, so its CELEX has to be
// checked against the directive being asked about — otherwise a search for the
// measures of one directive surfaces CELEX values belonging to another.
func TestPrefissoMisuraDerivesFromTheDirective(t *testing.T) {
	if got := prefissoMisura("32022L2555", "ITA"); got != "72022L2555ITA" {
		t.Errorf("= %q", got)
	}
}

func TestCelexShapes(t *testing.T) {
	if !IsCelexAtto("32022L2555") || IsCelexAtto("72022L2555ITA_1") {
		t.Error("il riconoscimento di un CELEX di atto è sbagliato")
	}
	if !IsCelexMisura("72022L2555ITA_1") || IsCelexMisura("32022L2555") {
		t.Error("il riconoscimento di un CELEX di misura è sbagliato")
	}
	// A corrigendum is not the act: its CELEX carries a suffix.
	if IsCelexAtto("32024R2847R(04)") {
		t.Error("una rettifica è stata riconosciuta come atto")
	}
}

// --- SPARQL literals ---------------------------------------------------------

// The term reaches the endpoint as query syntax, so an unescaped quotation mark
// is a malformed query and a 400 the caller cannot explain.
func TestLiteralClosesTheWayOutOfTheString(t *testing.T) {
	got := literal(`a" } UNION { ?x ?y ?z`)
	if strings.Contains(got, `"`) && !strings.Contains(got, `\"`) {
		t.Errorf("= %q: le virgolette non sono state protette", got)
	}
}

// Apostrophes are legal inside "..." and ordinary in Italian legal prose;
// removing them would make a phrase match nothing.
func TestLiteralKeepsApostrophes(t *testing.T) {
	if got := literal("tutela dell'ambiente"); got != "tutela dell'ambiente" {
		t.Errorf("= %q", got)
	}
}

// --- the parser --------------------------------------------------------------

// The shape the Official Journal actually publishes: the chapter carries an id
// and NO class, while the article carries both.
const attoXHTML = `<html><body>
<div class="eli-container" id="cons_1">
  <div class="eli-subdivision" id="pbl_1">
    <p class="oj-normal">visto il trattato…</p>
    <div class="eli-subdivision" id="rct_1"><p class="oj-normal">(1) considerando che…</p></div>
  </div>
  <div class="eli-subdivision" id="enc_1">
    <div id="cpt_I">
      <p class="oj-ti-section-1">CAPO I</p>
      <div class="eli-title" id="cpt_I.tit_1"><p class="oj-ti-section-2">DISPOSIZIONI GENERALI</p></div>
      <div class="eli-subdivision" id="art_1">
        <p class="oj-ti-art">Articolo 1</p>
        <div class="eli-title" id="art_1.tit_1"><p class="oj-sti-art">Oggetto</p></div>
        <div id="001.001"><p class="oj-normal">1. La presente direttiva stabilisce misure.</p></div>
      </div>
      <div class="eli-subdivision" id="art_2">
        <p class="oj-ti-art">Articolo 2</p>
        <div id="002.001"><p class="oj-normal">2. Ai fini della presente direttiva.</p></div>
      </div>
    </div>
  </div>
</div>
<hr class="oj-doc-sep"/>
<div id="L_2022333IT.01014301">
  <div class="eli-container" id="anx_I">
    <p class="oj-doc-ti">ALLEGATO I</p>
    <p class="oj-doc-ti">SETTORI AD ALTA CRITICITÀ</p>
    <table class="oj-table"><tr><td><p class="oj-normal">Energia</p></td></tr></table>
  </div>
</div></body></html>`

func parseFixture(t *testing.T) *document.Document {
	t.Helper()
	doc := document.NewDocument("32022L2555", "Direttiva di prova", "2022-12-14", "")
	if err := parseAtto([]byte(attoXHTML), &doc); err != nil {
		t.Fatal(err)
	}
	return &doc
}

// The trap this parser exists for. Chapters carry an id and no class, so walking
// `div.eli-subdivision` children finds the enacting terms and then nothing —
// the articles are grandchildren, behind a chapter the selector does not match.
// Measured on the real directive, that yielded an act with ZERO articles.
func TestArticlesAreFoundBehindAClasslessChapter(t *testing.T) {
	doc := parseFixture(t)

	var articoli []document.DocumentSection
	var walk func([]document.DocumentSection)
	walk = func(secs []document.DocumentSection) {
		for _, s := range secs {
			if s.Type == "article" {
				articoli = append(articoli, s)
			}
			walk(s.Children)
		}
	}
	walk(doc.Sections)

	if len(articoli) != 2 {
		t.Fatalf("%d articoli, attesi 2 — il capo senza classe li ha nascosti", len(articoli))
	}
	if articoli[0].ID != "art_1" {
		t.Errorf("id = %q: è l'ancora usata dall'indice e dalle annotazioni", articoli[0].ID)
	}
}

// "Articolo 1" alone says nothing and "Oggetto" alone cannot be cited.
func TestAnArticleHeadingJoinsNumberAndRubric(t *testing.T) {
	doc := parseFixture(t)
	art := doc.Sections[1].Children[0].Children[0]
	if art.Title != "Articolo 1 — Oggetto" {
		t.Errorf("titolo = %q", art.Title)
	}
	// An article with no rubric keeps its number rather than coming out untitled.
	art2 := doc.Sections[1].Children[0].Children[1]
	if art2.Title != "Articolo 2" {
		t.Errorf("titolo senza rubrica = %q", art2.Title)
	}
}

// A chapter that repeated the text of every article under it would triple the
// length of the act in every export.
func TestAChapterDoesNotRepeatTheTextOfItsArticles(t *testing.T) {
	doc := parseFixture(t)
	capo := doc.Sections[1].Children[0]
	for _, c := range capo.Content {
		if strings.Contains(c, "stabilisce misure") {
			t.Fatalf("il capo contiene il testo di un articolo: %q", c)
		}
	}
}

// The heading is the section's Title; emitting it as content too would print
// every article's number twice.
func TestTheHeadingIsNotAlsoContent(t *testing.T) {
	doc := parseFixture(t)
	art := doc.Sections[1].Children[0].Children[0]
	for _, c := range art.Content {
		if c == "Articolo 1" || c == "Oggetto" {
			t.Errorf("il titolo è finito nel contenuto: %q", c)
		}
	}
	if len(art.Content) != 1 || !strings.Contains(art.Content[0], "stabilisce misure") {
		t.Errorf("contenuto = %v", art.Content)
	}
}

// A directive has a hundred and fifty recitals. Each as a section buries the
// articles in the table of contents, so they stay as the preamble's text.
func TestRecitalsStayInThePreambleRatherThanBecomingSections(t *testing.T) {
	doc := parseFixture(t)
	pre := doc.Sections[0]
	if pre.Type != "preamble" {
		t.Fatalf("prima sezione = %q", pre.Type)
	}
	if len(pre.Children) != 0 {
		t.Errorf("il preambolo ha %d sottosezioni: i considerando devono restare testo", len(pre.Children))
	}
	joined := strings.Join(pre.Content, " ")
	if !strings.Contains(joined, "considerando che") {
		t.Errorf("il testo dei considerando è andato perso: %v", pre.Content)
	}
}

// An act published without the ELI containers must still be readable, rather
// than coming back empty — which looks exactly like a fetch that failed.
func TestAnActWithoutEliContainersIsStillReadable(t *testing.T) {
	doc := document.NewDocument("31968L0001", "Atto antico", "1968-01-01", "")
	err := parseAtto([]byte(`<html><body><p>Il Consiglio ha adottato la presente direttiva.</p></body></html>`), &doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) != 1 || len(doc.Sections[0].Content) == 0 {
		t.Fatalf("sezioni = %+v", doc.Sections)
	}
}

// Scripts are text to a naive extractor, so a page whose body is short and whose
// script block is long comes out as minified JavaScript.
func TestScriptsAreNotReadAsTheAct(t *testing.T) {
	doc := document.NewDocument("x", "", "", "")
	_ = parseAtto([]byte(`<html><body><script>var x="non è il testo";</script><p>Il testo vero.</p></body></html>`), &doc)
	for _, s := range doc.Sections {
		for _, c := range s.Content {
			if strings.Contains(c, "var x") {
				t.Fatalf("script nel testo: %q", c)
			}
		}
	}
}

func TestTipoSezione(t *testing.T) {
	cases := map[string]string{
		"art_1": "article", "cpt_I": "chapter", "sct_2": "section",
		"anx_I": "annex", "pbl_1": "preamble", "enc_1": "body", "fnp_1": "closing",
	}
	for id, want := range cases {
		if got := tipoSezione(id); got != want {
			t.Errorf("tipoSezione(%q) = %q, atteso %q", id, got, want)
		}
	}
}

// --- client ------------------------------------------------------------------

func fakeCellar(t *testing.T, handle func(query string) string) *int {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		form, _ := url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/sparql-results+json")
		w.Write([]byte(handle(form.Get("query"))))
	}))
	t.Cleanup(srv.Close)

	prev := cellarSPARQL
	cellarSPARQL = srv.URL + "/sparql"
	t.Cleanup(func() { cellarSPARQL = prev })
	return &calls
}

func sparqlJSON(rows ...map[string]string) string {
	var bindings []map[string]map[string]string
	for _, r := range rows {
		b := map[string]map[string]string{}
		for k, v := range r {
			b[k] = map[string]string{"value": v}
		}
		bindings = append(bindings, b)
	}
	out, _ := json.Marshal(map[string]any{"results": map[string]any{"bindings": bindings}})
	return string(out)
}

// A reader pasting a CELEX means "open this", and a title search for it finds
// nothing at all.
func TestSearchingACelexIsALookupNotATitleSearch(t *testing.T) {
	var seen string
	fakeCellar(t, func(q string) string {
		seen = q
		return sparqlJSON(map[string]string{"title": "Direttiva NIS2", "date": "2022-12-14"})
	})

	got, err := NewClient(0).Search("32022L2555")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].CodiceRedazionale != "32022L2555" {
		t.Fatalf("= %+v", got)
	}
	if strings.Contains(seen, "CONTAINS(LCASE(STR(?title))") {
		t.Error("un CELEX è stato cercato fra i titoli anziché aperto")
	}
	if got[0].Link == "" || !strings.Contains(got[0].Link, "eur-lex.europa.eu") {
		t.Errorf("manca il link citabile: %q", got[0].Link)
	}
}

// One work has several Italian expressions in CELLAR, so it comes back once per
// expression: undeduplicated a result list shows the same directive three times.
func TestTheSameActIsNotListedOncePerExpression(t *testing.T) {
	fakeCellar(t, func(q string) string {
		return sparqlJSON(
			map[string]string{"celex": "32022L2555", "title": "Direttiva NIS2", "date": "2022-12-14"},
			map[string]string{"celex": "32022L2555", "title": "Direttiva NIS2", "date": "2022-12-14"},
			map[string]string{"celex": "32016R0679", "title": "GDPR", "date": "2016-04-27"},
		)
	})
	got, err := NewClient(0).Search("protezione dati personali")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%d risultati, attesi 2 atti distinti su 3 espressioni", len(got))
	}
}

// The search is an AND over the words, and the corrigenda are excluded by an
// anchored CELEX pattern.
func TestTheSearchQueryIsAnchoredAndConjunctive(t *testing.T) {
	var seen string
	fakeCellar(t, func(q string) string {
		seen = q
		return sparqlJSON()
	})
	_, _ = NewClient(0).Search("sicurezza reti")

	if !strings.Contains(seen, `"^3[0-9]{4}[LR][0-9]{4}$"`) {
		t.Errorf("il CELEX non è ancorato, le rettifiche entreranno nei risultati:\n%s", seen)
	}
	if !strings.Contains(seen, "&&") {
		t.Errorf("i termini non sono in AND:\n%s", seen)
	}
}

// nil rather than an error: "this CELEX names nothing" is an answer, and an
// error would make the fall-back search impossible to write.
func TestAnUnknownCelexIsNotAnError(t *testing.T) {
	fakeCellar(t, func(q string) string { return sparqlJSON() })
	got, err := NewClient(0).Metadata("39999L9999")
	if err != nil {
		t.Fatalf("un CELEX inesistente è stato segnalato come errore: %v", err)
	}
	if got != nil {
		t.Errorf("= %+v", got)
	}
}

// --- transposition -----------------------------------------------------------

// The join is anchored on the directive FIRST: an unconstrained measure join
// walks every national measure of every member state and times out.
func TestTheTranspositionQueryIsAnchoredOnTheDirective(t *testing.T) {
	var queries []string
	fakeCellar(t, func(q string) string {
		queries = append(queries, q)
		if strings.Contains(q, "measure_national_implementing_implements") {
			return sparqlJSON(map[string]string{
				"mne": "http://x/1", "tipo": "http://x/Decreto legislativo",
				"id_local": "Decreto legislativo 4 settembre 2024, n. 138,",
			})
		}
		return sparqlJSON(map[string]string{"titolo": "Direttiva NIS2", "termine": "2024-10-17"})
	})

	got, err := NewClient(0).Recepimento("direttiva 2022/2555", "ITA")
	if err != nil {
		t.Fatal(err)
	}
	if got.Direttiva.TermineRecepimento != "2024-10-17" {
		t.Errorf("termine = %q", got.Direttiva.TermineRecepimento)
	}
	if len(got.Misure) != 1 {
		t.Fatalf("%d misure", len(got.Misure))
	}
	// The trailing comma of the citation is the archive's, not the act's.
	if got.Misure[0].Riferimento != "Decreto legislativo 4 settembre 2024, n. 138" {
		t.Errorf("riferimento = %q", got.Misure[0].Riferimento)
	}
	last := queries[len(queries)-1]
	if !strings.Contains(last, `?dir cdm:resource_legal_id_celex "32022L2555"`) {
		t.Errorf("la join non parte dalla direttiva:\n%s", last)
	}
}

// An empty list has three very different meanings here and only the caller can
// act on the difference.
func TestNoMeasuresIsExplainedRatherThanLeftEmpty(t *testing.T) {
	fakeCellar(t, func(q string) string {
		if strings.Contains(q, "measure_national_implementing_implements") {
			return sparqlJSON()
		}
		return sparqlJSON(map[string]string{"titolo": "Regolamento GDPR"})
	})
	got, err := NewClient(0).Recepimento("regolamento 2016/679", "ITA")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Misure) != 0 {
		t.Fatal("attese zero misure")
	}
	joined := strings.Join(got.Note, " ")
	if !strings.Contains(joined, "regolamento") {
		t.Errorf("la nota non spiega le cause possibili: %v", got.Note)
	}
}

// A directive CELLAR does not hold is an error naming it, not an empty answer
// that reads as "nobody has transposed it".
func TestAnUnknownDirectiveIsRefusedByName(t *testing.T) {
	fakeCellar(t, func(q string) string { return sparqlJSON() })
	_, err := NewClient(0).Recepimento("direttiva 9999/9999", "ITA")
	if err == nil {
		t.Fatal("una direttiva inesistente non è stata segnalata")
	}
	if !strings.Contains(err.Error(), "39999L9999") {
		t.Errorf("l'errore non nomina il CELEX cercato: %v", err)
	}
}

// Both shapes of id_local are matched AND both are constrained by the year —
// otherwise a bare "138" of any year matches, and "n. 90" matches "n. 190".
func TestTheItalianActMatchIsConstrainedByTheYear(t *testing.T) {
	var seen string
	fakeCellar(t, func(q string) string {
		seen = q
		return sparqlJSON()
	})
	if _, err := NewClient(0).BaseUE("D.Lgs. 138/2024", "ITA"); err != nil {
		t.Fatal(err)
	}
	if strings.Count(seen, "2024") < 3 {
		t.Errorf("l'anno non vincola tutti i rami del match:\n%s", seen)
	}
	if !strings.Contains(seen, `"n. 138,"`) || !strings.Contains(seen, `"n. 138 "`) {
		t.Errorf("una delle due forme di id_local non è cercata:\n%s", seen)
	}
}

func TestAnUnparseableActReferenceIsRefusedWithAnExample(t *testing.T) {
	_, err := NewClient(0).BaseUE("il decreto trasparenza", "ITA")
	if err == nil {
		t.Fatal("riferimento non interpretabile accettato")
	}
	if !strings.Contains(err.Error(), "138/2024") {
		t.Errorf("il rifiuto non mostra una forma valida: %v", err)
	}
}

// The absence of a link is not proof that an act has no EU origin: the link
// exists only where the member state notified the measure.
func TestNoDirectiveIsExplainedRatherThanLeftEmpty(t *testing.T) {
	fakeCellar(t, func(q string) string { return sparqlJSON() })
	got, err := NewClient(0).BaseUE("legge 90/2024", "ITA")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Note) == 0 || !strings.Contains(strings.Join(got.Note, " "), "notificato") {
		t.Errorf("l'assenza di collegamenti non è spiegata: %v", got.Note)
	}
}

// The dated form must win over the compact one, which is why it is tried first:
// against a title carrying both a date and a number, the compact expression
// reads a pair that is not the citation.
func TestTheDatedFormWinsOverAStrayNumberPair(t *testing.T) {
	numero, anno, ok := RiferimentoAttoItaliano("Testo unico 2015 art. 3 - DECRETO LEGISLATIVO 4 settembre 2024, n. 138")
	if !ok || numero != "138" || anno != "2024" {
		t.Fatalf("numero=%q anno=%q ok=%v; atteso 138/2024", numero, anno, ok)
	}
}

// A reference with no year/number pair yields nothing rather than a guess: a
// fabricated citation queries a different act and answers confidently about it.
func TestAReferenceWithNoCitationIsRefused(t *testing.T) {
	if _, _, ok := RiferimentoAttoItaliano("codice civile"); ok {
		t.Fatal("una citazione inventata da \"codice civile\"")
	}
}

// An annex is published as its own eli-container, a SIBLING of the one holding
// the articles — so a parser reading only the first container drops every annex.
//
// Measured on directive 32022L2555 before this: three annexes missing, including
// the tables that decide which entities the directive applies to, which is the
// most consulted part of it. Nothing said so; the act simply ended at its last
// article, which is what an act looks like.
func TestAnAnnexPublishedAsItsOwnContainerIsNotDropped(t *testing.T) {
	doc := parseFixture(t)

	var annex *document.DocumentSection
	for i := range doc.Sections {
		if doc.Sections[i].ID == "anx_I" {
			annex = &doc.Sections[i]
		}
	}
	if annex == nil {
		var ids []string
		for _, s := range doc.Sections {
			ids = append(ids, s.ID)
		}
		t.Fatalf("allegato assente; sezioni di primo livello: %v", ids)
	}
	if annex.Type != "annex" {
		t.Errorf("tipo %q, atteso annex", annex.Type)
	}
	if annex.Title != "ALLEGATO I" {
		t.Errorf("titolo %q, atteso \"ALLEGATO I\"", annex.Title)
	}
}

// The annex's NAME is a second paragraph of the same class as its number, and
// only the first is the heading. Excluding the class threw the name away, so the
// annex arrived titled "ALLEGATO I" and said nothing about what it contains.
func TestAnAnnexKeepsItsNameAndItsTable(t *testing.T) {
	doc := parseFixture(t)

	var annex *document.DocumentSection
	for i := range doc.Sections {
		if doc.Sections[i].ID == "anx_I" {
			annex = &doc.Sections[i]
		}
	}
	if annex == nil {
		t.Fatal("allegato assente")
	}
	testo := strings.Join(annex.Content, " ")
	if !strings.Contains(testo, "SETTORI AD ALTA CRITICIT") {
		t.Errorf("il nome dell'allegato non è nel testo: %q", testo)
	}
	if !strings.Contains(testo, "Energia") {
		t.Errorf("la tabella dell'allegato non è nel testo: %q", testo)
	}
	// A negative control: the number must not be repeated as body text, or every
	// heading in the act appears twice.
	if strings.Count(testo, "ALLEGATO I") != 0 {
		t.Errorf("il numero dell'allegato è ripetuto nel testo: %q", testo)
	}
}
