package parsing

import (
	"log"
	"strings"
)

type PDFDTO struct {
	Title     string       `json:"title"`
	DOI       string       `json:"doi"`
	CustomKey string       `json:"custom_key"`
	ISSN      string       `json:"issn"`
	Abstract  string       `json:"abstract"`
	Sections  []SectionRaw `json:"sections"`
	Keywords  []string     `json:"keywords"`
	Authors   []AuthorsRaw `json:"authors"`
	Year      string       `json:"year"`
	Journal   string       `json:"journal"`
	Notes     string       `json:"notes"`
	Date      string       `json:"date"`
}

// create a PDFDTO
func CreatePDFDTO(tidyGrobidResponse *TidyGrobidResponse, tidyCrossRefResponse *TidyCrossRefResponse) *PDFDTO {

	if tidyCrossRefResponse != nil {
		// prefer crossref data for title, abstract, year
		if tidyCrossRefResponse.Title != "" {
			log.Println("Using crossref title")
			tidyGrobidResponse.Title = tidyCrossRefResponse.Title
		}
		if tidyCrossRefResponse.Abstract != "" {
			// I do not have much faith in this
			log.Println("Using crossref abstract")
			tidyGrobidResponse.Abstract = tidyCrossRefResponse.Abstract
		}
		if tidyCrossRefResponse.Year != "" {
			log.Println("Using crossref year")
			tidyGrobidResponse.Year = tidyCrossRefResponse.Year
		}
	}

	// trim title and replace '-' with ' '
	tidyGrobidResponse.Title = strings.TrimSpace(tidyGrobidResponse.Title)
	tidyGrobidResponse.Title = strings.ReplaceAll(tidyGrobidResponse.Title, "-", " ")

	// trim abstract
	tidyGrobidResponse.Abstract = strings.TrimSpace(tidyGrobidResponse.Abstract)

	return &PDFDTO{
		Title:    tidyGrobidResponse.Title,
		DOI:      tidyGrobidResponse.Doi,
		Abstract: tidyGrobidResponse.Abstract,
		Sections: tidyGrobidResponse.Sections,
		Keywords: tidyGrobidResponse.Keywords,
		Authors:  tidyGrobidResponse.Authors,
		Year:     tidyGrobidResponse.Year,
		Date:     tidyGrobidResponse.Date,
		Journal:  tidyGrobidResponse.Journal,
		Notes:    tidyGrobidResponse.Notes,
	}
}
