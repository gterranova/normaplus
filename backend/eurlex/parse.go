package eurlex

// Turning CELLAR's XHTML into the same Document the Normattiva parser produces.
//
// This is what makes the rest of the application work on an EU act for free: the
// viewer, the table of contents, the annotation anchors and every exporter read
// Document and DocumentSection, so an EU act that arrives in that shape needs no
// new UI to be read, navigated, annotated or exported.
//
// # The structure is in the id, not in the class
//
// That is the trap, and it is easy to get backwards. The markup looks like this:
//
//	div.eli-subdivision id="enc_1"          the enacting terms
//	  div            id="cpt_I"             a chapter — NO class at all
//	    p.oj-ti-section-1                     "CAPO I"
//	    div.eli-title                         its name
//	    div.eli-subdivision id="art_1"      an article
//	      p.oj-ti-art                         "Articolo 1"
//	      div.eli-title p.oj-sti-art          its rubric
//	      div id="001.001"                    a numbered paragraph container
//	        p.oj-normal                         the text
//
// Chapters carry an id and no class, while articles carry both. So walking
// `div.eli-subdivision` children finds the enacting terms and then nothing: the
// articles are grandchildren, behind a chapter the selector does not match.
// Measured on directive 32022L2555, that yielded an act of four sections and
// zero articles.
//
// What identifies a structural node is therefore its id prefix. A container with
// a numeric id (001.001) is a paragraph wrapper, not a division, and its text
// belongs to the article around it.
//
// The `id` is carried through to DocumentSection.ID because the frontend uses it
// as the anchor for the table of contents and for annotations — so `art_1` is
// the link target, exactly as a Normattiva article id is.

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gterranova/normaplus/backend/normattiva/document"
)

// prefissiStrutturali are the id prefixes that mark a division of the act.
//
// `cit_` (citations) and `rct_` (recitals) are deliberately absent: they are
// divisions too, but a directive has a hundred and fifty of them, and turning
// each into a section buries the articles in the table of contents. They stay as
// the preamble's text, which is where a reader looks for them.
var prefissiStrutturali = []string{"pbl_", "enc_", "cpt_", "tit_", "sct_", "art_", "anx_", "fnp_"}

func isStrutturale(id string) bool {
	for _, p := range prefissiStrutturali {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	return false
}

func idDi(s *goquery.Selection) string {
	id, _ := s.Attr("id")
	return id
}

// parseAtto fills doc with the structure of one EU act.
func parseAtto(html []byte, doc *document.Document) error {
	d, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return fmt.Errorf("testo dell'atto non interpretabile: %w", err)
	}

	// Scripts and styles are text to a naive extractor, so a page whose body is
	// short and whose script block is long comes out as minified JavaScript —
	// which reads, to anyone downstream, as the act.
	d.Find("script, style, noscript").Remove()

	// EVERY eli-container, not the first. An annex is published as a container
	// of its own, a sibling of the one holding the articles:
	//
	//	<div id="L_2022333IT.01008001"><div class="eli-container">…articles…</div></div>
	//	<hr class="oj-doc-sep"/>
	//	<div id="L_2022333IT.01014301"><div class="eli-container" id="anx_I">…</div></div>
	//
	// Reading only the first therefore dropped every annex, silently and without
	// shortening the act enough to notice: measured on directive 32022L2555, the
	// three annexes went missing — including the tables that decide which
	// entities the directive applies to, which is the most consulted part of it.
	containers := d.Find("div.eli-container")
	if containers.Length() == 0 {
		// Some older acts are published without the ELI containers. Falling back
		// to the body keeps them readable as one section rather than returning an
		// act with no text at all.
		containers = d.Find("body").First()
	}
	if containers.Length() == 0 {
		return fmt.Errorf("il documento non contiene un corpo leggibile")
	}

	if t := oneLine(containers.First().Find("div.eli-main-title").First().Text()); t != "" && doc.Title == "" {
		doc.Title = t
	}

	containers.Each(func(_ int, c *goquery.Selection) {
		// An annex container carries the division id ITSELF rather than holding
		// one, so descending into it would look for a division inside a division
		// and find nothing.
		if isStrutturale(idDi(c)) {
			doc.AddSection(sezione(c, doc))
			return
		}
		for _, s := range divisioniProprie(c) {
			doc.AddSection(sezione(s, doc))
		}
	})

	if len(doc.Sections) == 0 {
		// No structural ids at all: one section holding the readable text, rather
		// than an empty document that looks like a fetch which silently failed.
		sec := document.NewDocumentSection("body", "", doc)
		for _, p := range testoDi(containers.First(), nil) {
			sec.AddContent(p)
		}
		if len(sec.Content) == 0 {
			return fmt.Errorf("il documento non contiene testo leggibile")
		}
		doc.AddSection(sec)
	}
	return nil
}

// divisioniProprie returns the NEAREST structural descendants of s — the ones
// not nested inside another structural descendant of s.
//
// This is what crosses the class-less chapter: it descends through anything that
// is not itself a division, and stops as soon as it finds one.
func divisioniProprie(s *goquery.Selection) []*goquery.Selection {
	var out []*goquery.Selection
	s.Children().Each(func(_ int, child *goquery.Selection) {
		// An eli-title is the HEADING of the node around it, never a division —
		// and its id is derived from that node's, so `art_1.tit_1` passes the
		// prefix test and would be read as a third article sitting between the
		// two real ones, with the rubric as its title and the article left
		// untitled. Checked before the prefix for exactly that reason.
		if child.HasClass("eli-title") {
			return
		}
		if isStrutturale(idDi(child)) {
			out = append(out, child)
			return
		}
		out = append(out, divisioniProprie(child)...)
	})
	return out
}

// sezione maps one division and recurses into the divisions inside it.
func sezione(s *goquery.Selection, root *document.Document) document.DocumentSection {
	id := idDi(s)
	sec := document.NewDocumentSection(tipoSezione(id), titolo(s), root)
	sec.ID = id

	figlie := divisioniProprie(s)
	// The heading is excluded by identity rather than by class, because a
	// division can carry SEVERAL paragraphs of the heading class and only the
	// first is the heading: an annex is titled "ALLEGATO I" and then named
	// "SETTORI AD ALTA CRITICITÀ", both as p.oj-doc-ti. Excluding the class threw
	// the name away.
	escluse := figlie
	if h := intestazione(s); h != nil {
		escluse = append(append([]*goquery.Selection{}, figlie...), h)
	}
	for _, p := range testoDi(s, escluse) {
		sec.AddContent(p)
	}
	for _, child := range figlie {
		sec.AddSection(sezione(child, root))
	}
	return sec
}

// tipoSezione reads the kind of division off its id, which is how the markup
// states it.
func tipoSezione(id string) string {
	switch {
	case strings.HasPrefix(id, "art_"):
		return "article"
	case strings.HasPrefix(id, "cpt_"):
		return "chapter"
	case strings.HasPrefix(id, "sct_"):
		return "section"
	case strings.HasPrefix(id, "tit_"):
		return "title"
	case strings.HasPrefix(id, "anx_"):
		return "annex"
	case strings.HasPrefix(id, "pbl_"):
		return "preamble"
	case strings.HasPrefix(id, "enc_"):
		return "body"
	case strings.HasPrefix(id, "fnp_"):
		return "closing"
	}
	return "section"
}

// titolo assembles the heading a reader expects: the number and the rubric
// joined, because "Articolo 1" alone says nothing and "Oggetto e ambito di
// applicazione" alone cannot be cited.
// selIntestazione are the paragraphs that can BE a division's heading.
const selIntestazione = "p.oj-ti-art, p.oj-ti-section-1, p.oj-ti-grseq-1, p.oj-doc-ti"

// intestazione returns the node used as the heading, or nil.
func intestazione(s *goquery.Selection) *goquery.Selection {
	h := s.ChildrenFiltered(selIntestazione).First()
	if h.Length() == 0 {
		return nil
	}
	return h
}

func titolo(s *goquery.Selection) string {
	numero := oneLine(s.ChildrenFiltered(selIntestazione).First().Text())
	rubrica := oneLine(s.ChildrenFiltered("div.eli-title").First().Text())
	switch {
	case numero != "" && rubrica != "":
		return numero + " — " + rubrica
	case numero != "":
		return numero
	}
	return rubrica
}

// classiTitolo are the paragraphs that ARE the heading, so they must not also be
// emitted as the section's text.
//
// `oj-doc-ti` is deliberately absent: an annex uses it twice, once for the
// number and once for the name, and only the first is the heading — see
// `intestazione`, which excludes that one node by identity.
var classiTitolo = []string{"oj-ti-art", "oj-ti-section-1", "oj-ti-section-2", "oj-sti-art", "oj-ti-grseq-1"}

// testoDi collects the readable paragraphs that belong to s rather than to one
// of the divisions inside it.
//
// escluse is what divisioniProprie found, so a chapter does not repeat the whole
// text of every article under it — which is what a naive Find would produce, and
// which would triple the length of an act in every export.
func testoDi(s *goquery.Selection, escluse []*goquery.Selection) []string {
	var out []string
	s.Find("p, td").Each(func(_ int, p *goquery.Selection) {
		// A <p> inside a <td> is emitted as the paragraph; emitting the cell too
		// would duplicate it.
		if p.Is("td") && p.Find("p").Length() > 0 {
			return
		}
		if dentroUnaDi(p, escluse) {
			return
		}
		for _, c := range classiTitolo {
			if p.HasClass(c) {
				return
			}
		}
		if p.Closest("div.eli-title").Length() > 0 {
			return
		}
		if t := oneLine(p.Text()); t != "" {
			out = append(out, t)
		}
	})
	return out
}

// dentroUnaDi reports whether p sits inside any of the given divisions.
//
// Compared by node identity rather than by selector: two divisions can share a
// class, and an id-based selector would need every id to be a valid one.
func dentroUnaDi(p *goquery.Selection, divisioni []*goquery.Selection) bool {
	if len(divisioni) == 0 {
		return false
	}
	for _, d := range divisioni {
		if d.Nodes[0] == p.Nodes[0] {
			return true
		}
		inside := false
		p.Parents().Each(func(_ int, parent *goquery.Selection) {
			if parent.Nodes[0] == d.Nodes[0] {
				inside = true
			}
		})
		if inside {
			return true
		}
	}
	return false
}

// oneLine collapses every run of whitespace, including the non-breaking spaces
// the Official Journal's typesetting is full of.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, " ", " ")
	return strings.Join(strings.Fields(s), " ")
}
