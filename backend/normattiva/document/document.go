package document

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Document struct {
	Name              string            `json:"name"`
	Title             string            `json:"title"`
	CodiceRedazionale string            `json:"codiceRedazionale"`
	DataGU            string            `json:"dataGU"`
	Vigenza           string            `json:"vigenza"`
	Sections          []DocumentSection `json:"sections"`
}

func NewDocument(codiceRedazionale, name, dataPubblicazioneGazzetta, vigenza string) Document {
	return Document{
		Name:              name,
		CodiceRedazionale: codiceRedazionale,
		DataGU:            dataPubblicazioneGazzetta,
		Vigenza:           vigenza,
	}
}

func (d *Document) AddSection(section DocumentSection) {
	d.Sections = append(d.Sections, section)
}

func (d *Document) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}

func (d *Document) ToMarkdown() ([]byte, error) {
	var sb strings.Builder

	if d.Vigenza != "" {
		displayDate := d.Vigenza
		parts := strings.Split(d.Vigenza, "-")
		if len(parts) == 3 {
			displayDate = fmt.Sprintf("%s-%s-%s", parts[2], parts[1], parts[0])
		}
		sb.WriteString(fmt.Sprintf("*Testo in vigore al: %s*\n\n", displayDate))
	}

	if d.Title != "" {
		sb.WriteString("\n\n<span id=\"preamble\"></span>\n\n")
		sb.WriteString(fmt.Sprintf("# %s\n\n", d.Title))
	}

	for _, section := range d.Sections {
		// Level 1 logic implies main sections start at some level.
		// If "preamble" is a section, handle it.
		// "Body" usually contains children which are the top level structure.
		section.WriteMarkdown(&sb, 1)
	}

	return []byte(sb.String()), nil
}
