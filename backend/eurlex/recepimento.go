package eurlex

// Transposition: which Italian measures implement an EU directive, and which
// directive an Italian act implements.
//
// CELLAR records this as a first-class relation
// (cdm:measure_national_implementing_implements_resource_legal), which is why
// this can be answered precisely rather than by searching titles for a
// citation — the one place in this application where an EU↔IT link is a fact in
// the data rather than an inference.

import (
	"fmt"
	"strings"
)

const paeseBase = "http://publications.europa.eu/resource/authority/country/"

// Misura is one national measure transposing a directive.
type Misura struct {
	Celex           string `json:"celex,omitempty"`
	Tipo            string `json:"tipo,omitempty"`
	Riferimento     string `json:"riferimento,omitempty"`
	Titolo          string `json:"titolo,omitempty"`
	EntrataInVigore string `json:"entrata_in_vigore,omitempty"`
	Paese           string `json:"paese"`
}

// Direttiva is one EU act, as the other end of a transposition link.
type Direttiva struct {
	Celex              string `json:"celex"`
	Titolo             string `json:"titolo,omitempty"`
	TermineRecepimento string `json:"termine_recepimento,omitempty"`
	Link               string `json:"link,omitempty"`
}

// Recepimento is the answer to "how was this directive implemented here".
type Recepimento struct {
	Direttiva Direttiva `json:"direttiva"`
	Misure    []Misura  `json:"misure"`
	Paese     string    `json:"paese"`
	Note      []string  `json:"note,omitempty"`
}

// BaseUE is the answer to "what does this Italian act implement".
type BaseUE struct {
	Atto      string      `json:"atto"`
	Direttive []Direttiva `json:"direttive"`
	Note      []string    `json:"note,omitempty"`
}

// Recepimento returns the national measures that transpose a directive.
//
// rif is written the way a lawyer writes it — "direttiva 2022/2555", "NIS2" will
// not do, but "2022/2555" will — or is already a CELEX.
func (c *Client) Recepimento(rif, paese string) (*Recepimento, error) {
	celex, ok := CelexDaRiferimento(rif)
	if !ok {
		return nil, fmt.Errorf("da %q non si ricava una direttiva: scrivila come \"direttiva 2022/2555\", \"regolamento 2016/679\" oppure con il CELEX (32022L2555)", rif)
	}
	if paese == "" {
		paese = "ITA"
	}
	paese = strings.ToUpper(paese)

	out := &Recepimento{Paese: paese}

	// The directive first, so an answer with no measures can tell "nothing has
	// been notified yet" from "this directive does not exist".
	dir, err := c.direttiva(celex)
	if err != nil {
		return nil, err
	}
	if dir == nil {
		return nil, fmt.Errorf("nessun atto UE con CELEX %s: verifica anno e numero", celex)
	}
	out.Direttiva = *dir

	// The join is anchored on the directive FIRST. An unconstrained ?mne join
	// walks every national measure of every member state and times out.
	q := prefissi + fmt.Sprintf(`
SELECT DISTINCT ?mne ?mne_celex ?tipo ?id_local ?eif ?titolo
WHERE {
  ?dir cdm:resource_legal_id_celex "%s"^^xsd:string .
  ?mne cdm:measure_national_implementing_implements_resource_legal ?dir .
  ?mne cdm:measure_national_implementing_implemented_by_country <%s%s> .
  OPTIONAL { ?mne cdm:measure_national_implementing_type_act ?tipo . }
  OPTIONAL { ?mne cdm:resource_legal_id_local ?id_local . }
  OPTIONAL { ?mne cdm:resource_legal_date_entry-into-force ?eif . }
  OPTIONAL { ?mne cdm:work_title ?titolo . }
  OPTIONAL {
    ?mne cdm:resource_legal_id_celex ?mne_celex .
    FILTER(STRSTARTS(STR(?mne_celex), "%s"))
  }
}
LIMIT 50`, literal(celex), paeseBase, paese, prefissoMisura(celex, paese))

	bindings, err := c.sparql(q)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, b := range bindings {
		// One measure fans out across the OPTIONAL clauses, so it returns as
		// several rows differing only in an optional value.
		key := b.get("mne")
		if key != "" && seen[key] {
			continue
		}
		seen[key] = true
		out.Misure = append(out.Misure, Misura{
			Celex:           b.get("mne_celex"),
			Tipo:            tipoDaURI(b.get("tipo")),
			Riferimento:     strings.TrimRight(strings.TrimSpace(b.get("id_local")), ","),
			Titolo:          oneLine(b.get("titolo")),
			EntrataInVigore: b.get("eif"),
			Paese:           paese,
		})
	}

	if len(out.Misure) == 0 {
		// Three very different situations land here, and only the caller can act
		// on the difference — so it is stated rather than left as an empty list.
		out.Note = append(out.Note,
			"nessuna misura nazionale notificata alla Commissione per questa direttiva e questo Paese: "+
				"può significare che il termine di recepimento non è ancora scaduto, che il recepimento non è "+
				"stato notificato, oppure che l'atto è un regolamento, che non si recepisce")
	}
	out.Note = append(out.Note,
		"fonte: le misure nazionali di esecuzione notificate dagli Stati membri e registrate in CELLAR; "+
			"non è un elenco delle norme interne che incidono sulla materia")
	return out, nil
}

func (c *Client) direttiva(celex string) (*Direttiva, error) {
	q := prefissi + fmt.Sprintf(`
SELECT ?titolo ?termine
WHERE {
  ?dir cdm:resource_legal_id_celex "%s"^^xsd:string .
  OPTIONAL { ?dir cdm:directive_date_transposition ?termine . }
  OPTIONAL {
    ?exp cdm:expression_belongs_to_work ?dir .
    ?exp cdm:expression_uses_language <%s> .
    ?exp cdm:expression_title ?titolo .
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
	return &Direttiva{
		Celex:              celex,
		Titolo:             oneLine(b.get("titolo")),
		TermineRecepimento: b.get("termine"),
		Link:               LinkEurLex(celex),
	}, nil
}

// BaseUE returns the EU acts an Italian act transposes.
//
// The Italian act is matched on number and year, because CELLAR's
// resource_legal_id_local is best-effort and arrives in two shapes: a bare
// number ("138"), and the full citation ("Decreto legislativo 4 settembre 2024,
// n. 138,"). Both are matched, and BOTH are constrained by the year — otherwise
// a bare "138" of any year matches, and "n. 90" matches "n. 190".
func (c *Client) BaseUE(rif, paese string) (*BaseUE, error) {
	if celex := strings.ToUpper(strings.TrimSpace(rif)); IsCelexMisura(celex) {
		return c.baseDaCelexMisura(celex)
	}
	numero, anno, ok := RiferimentoAttoItaliano(rif)
	if !ok {
		return nil, fmt.Errorf("da %q non si ricavano numero e anno: scrivilo come \"D.Lgs. 138/2024\" oppure \"legge n. 90 del 2024\"", rif)
	}
	if paese == "" {
		paese = "ITA"
	}
	paese = strings.ToUpper(paese)

	q := prefissi + fmt.Sprintf(`
SELECT DISTINCT ?dir_celex ?titolo ?termine
WHERE {
  ?mne cdm:measure_national_implementing_implemented_by_country <%s%s> .
  ?mne cdm:resource_legal_id_local ?id_local .
  OPTIONAL { ?mne cdm:work_title ?wtitle . }
  OPTIONAL { ?mne cdm:resource_legal_date_entry-into-force ?eif . }
  FILTER(
    (STR(?id_local) = "%s" && (CONTAINS(STR(COALESCE(?wtitle, "")), "%s") || CONTAINS(STR(COALESCE(?eif, "")), "%s")))
    || (CONTAINS(LCASE(STR(?id_local)), "n. %s,") && CONTAINS(STR(?id_local), "%s"))
    || (CONTAINS(LCASE(STR(?id_local)), "n. %s ") && CONTAINS(STR(?id_local), "%s"))
  )
  ?mne cdm:measure_national_implementing_implements_resource_legal ?dir .
  ?dir cdm:resource_legal_id_celex ?dir_celex .
  OPTIONAL { ?dir cdm:directive_date_transposition ?termine . }
  OPTIONAL {
    ?exp cdm:expression_belongs_to_work ?dir .
    ?exp cdm:expression_uses_language <%s> .
    ?exp cdm:expression_title ?titolo .
  }
}
LIMIT 30`,
		paeseBase, paese,
		literal(numero), literal(anno), literal(anno),
		literal(numero), literal(anno),
		literal(numero), literal(anno),
		langITA)

	bindings, err := c.sparql(q)
	if err != nil {
		return nil, err
	}
	return c.raccogliDirettive(strings.TrimSpace(rif), bindings), nil
}

func (c *Client) baseDaCelexMisura(celex string) (*BaseUE, error) {
	q := prefissi + fmt.Sprintf(`
SELECT DISTINCT ?dir_celex ?titolo ?termine
WHERE {
  ?mne cdm:resource_legal_id_celex "%s"^^xsd:string .
  ?mne cdm:measure_national_implementing_implements_resource_legal ?dir .
  ?dir cdm:resource_legal_id_celex ?dir_celex .
  OPTIONAL { ?dir cdm:directive_date_transposition ?termine . }
  OPTIONAL {
    ?exp cdm:expression_belongs_to_work ?dir .
    ?exp cdm:expression_uses_language <%s> .
    ?exp cdm:expression_title ?titolo .
  }
}
LIMIT 20`, literal(celex), langITA)

	bindings, err := c.sparql(q)
	if err != nil {
		return nil, err
	}
	return c.raccogliDirettive(celex, bindings), nil
}

func (c *Client) raccogliDirettive(atto string, bindings []sparqlBinding) *BaseUE {
	out := &BaseUE{Atto: atto}
	seen := map[string]bool{}
	for _, b := range bindings {
		celex := b.get("dir_celex")
		if celex == "" || seen[celex] {
			continue
		}
		seen[celex] = true
		out.Direttive = append(out.Direttive, Direttiva{
			Celex:              celex,
			Titolo:             oneLine(b.get("titolo")),
			TermineRecepimento: b.get("termine"),
			Link:               LinkEurLex(celex),
		})
	}
	if len(out.Direttive) == 0 {
		out.Note = append(out.Note,
			"nessuna direttiva risulta recepita da questo atto: il collegamento esiste solo se lo Stato "+
				"ha notificato l'atto alla Commissione come misura di esecuzione, quindi l'assenza non prova "+
				"che l'atto non abbia origine europea")
	}
	return out
}
