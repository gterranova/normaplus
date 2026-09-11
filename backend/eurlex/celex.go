package eurlex

// Reference parsing: turning what a lawyer writes into what CELLAR indexes.
//
// A CELEX number is the identity of an EU act. For legislation it is
// `3{year}{L|R}{number padded to 4}` — 3 is the sector (legislation), L a
// directive, R a regulation. A national transposition measure has its own CELEX
// in sector 7, built from the directive's: `7{year}{L|R}{number}{COUNTRY}_{n}`.

import (
	"fmt"
	"regexp"
	"strings"
)

// Sector 3 is legislation; sector 7 is a national implementing measure.
var (
	celexAttoRe   = regexp.MustCompile(`(?i)^3\d{4}[LR]\d{4}$`)
	celexMisuraRe = regexp.MustCompile(`(?i)^7\d{4}[LR]\d{4}[A-Z]{3}_\d+$`)

	// "direttiva 2019/790", "dir 2019/790/UE", "regolamento (UE) 2016/679",
	// or the bare "2019/790" a reader copies out of a footnote.
	riferimentoUERe = regexp.MustCompile(
		`(?i)(?:dirett?iva|dir\.?|regolamento|reg\.?)?\s*\(?(?:ue|ce|cee|euratom)?\)?\s*(?:n\.?\s*)?(\d{4})/(\d{1,4})`)

	// An Italian act cited with the date FIRST, which is how the Gazzetta and
	// Normattiva title every one of them: "DECRETO LEGISLATIVO 4 settembre
	// 2024, n. 138". The year is in the date and the number comes after it, so
	// the compact form below cannot read it — and this is the form the viewer
	// actually has to hand, since it is the document's own title.
	attoItalianoDataRe = regexp.MustCompile(`(?i)\b(\d{1,2})\s+\p{L}+\s+(\d{4})\s*,?\s*n\.?\s*(\d{1,5})`)

	// An Italian act cited compactly: "D.Lgs. 138/2024", "decreto legislativo
	// n. 138/2024", "legge 90/2024", "n. 138 del 4 settembre 2024".
	attoItalianoRe = regexp.MustCompile(`(?:n\.?\s*)?(\d{1,5})\s*(?:/|del\s+\d{1,2}\s+\w+\s+|,?\s*)(\d{4})`)

	// A regulation is only a regulation when the word is there AND the word
	// "direttiva" is not: "regolamento di attuazione della direttiva X" is about
	// a directive.
	regolamentoRe = regexp.MustCompile(`(?i)regolament|reg\.|\breg\b`)
	direttivaRe   = regexp.MustCompile(`(?i)dirett|dir\.|\bdir\b`)
)

// IsCelexAtto reports whether s is already the CELEX of an EU act.
func IsCelexAtto(s string) bool { return celexAttoRe.MatchString(strings.TrimSpace(s)) }

// IsCelexMisura reports whether s is the CELEX of a national implementing
// measure.
func IsCelexMisura(s string) bool { return celexMisuraRe.MatchString(strings.TrimSpace(s)) }

// CelexDaRiferimento builds the CELEX of an EU act from how it is cited, or
// returns it unchanged when it already is one.
//
// It returns ok=false rather than guessing: a reference with no year/number pair
// cannot be turned into a CELEX, and inventing one would query a different act
// and answer confidently about it.
func CelexDaRiferimento(rif string) (celex string, ok bool) {
	s := strings.TrimSpace(rif)
	if s == "" {
		return "", false
	}
	if IsCelexAtto(s) {
		return strings.ToUpper(s), true
	}

	lettera := "L"
	if regolamentoRe.MatchString(s) && !direttivaRe.MatchString(s) {
		lettera = "R"
	}
	m := riferimentoUERe.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	return fmt.Sprintf("3%s%s%04s", m[1], lettera, m[2]), true
}

// RiferimentoAttoItaliano splits an Italian act reference into its number and
// year, which is how CELLAR's national measures are matched.
//
// The type of act is deliberately not returned: CELLAR records it as free text
// in several spellings, so matching on it loses measures rather than narrowing
// them.
func RiferimentoAttoItaliano(rif string) (numero, anno string, ok bool) {
	s := strings.TrimSpace(rif)

	// The dated form is tried FIRST, and the order is load-bearing. Against
	// "DECRETO LEGISLATIVO 4 settembre 2024, n. 138" the compact expression
	// reads "2024, n. 138" as nothing at all, but on a longer title it can
	// match a stray pair of numbers — so the form that is certain has to win.
	if m := attoItalianoDataRe.FindStringSubmatch(s); m != nil {
		return m[3], m[2], true
	}

	m := attoItalianoRe.FindStringSubmatch(s)
	if m == nil || len(m[2]) != 4 {
		return "", "", false
	}
	return m[1], m[2], true
}

// prefissoMisura is the CELEX prefix every national measure of one directive
// shares, for one country.
//
// It exists because a measure can transpose several directives at once, so its
// own CELEX has to be checked against the directive being asked about — without
// this, a search for the measures of one directive surfaces CELEX values
// belonging to another.
func prefissoMisura(celexDirettiva, paese string) string {
	return "7" + celexDirettiva[1:] + strings.ToUpper(paese)
}
