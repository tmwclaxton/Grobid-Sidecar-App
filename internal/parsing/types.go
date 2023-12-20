package parsing

import (
	"log"
	"strings"
)

type PDFDTO struct {
	Title    string   `json:"title"`
	DOI      string   `json:"doi"`
	Abstract string   `json:"abstract"`
	Sections []string `json:"sections"`
	Keywords []string `json:"keywords"`
	Authors  []string `json:"authors"`
	Year     string   `json:"year"`
	Journal  string   `json:"journal"`
	Notes    string   `json:"notes"`
	Date     string   `json:"date"`
}

// create a PDFDTO
func CreatePDFDTO(tidyGrobidResponse *TidyGrobidResponse, tidyCrossRefResponse *TidyCrossRefResponse) *PDFDTO {
	// prefer crossref data for title, abstract, year
	if tidyCrossRefResponse.Title != "" {
		log.Println("Using crossref title")
		tidyGrobidResponse.Title = tidyCrossRefResponse.Title
	}
	if tidyCrossRefResponse.Abstract != "" {
		log.Println("Using crossref abstract")
		tidyGrobidResponse.Abstract = tidyCrossRefResponse.Abstract
	}
	if tidyCrossRefResponse.Year != "" {
		log.Println("Using crossref year")
		tidyGrobidResponse.Year = tidyCrossRefResponse.Year
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

type Paper struct {
	ID        int64  `json:"id"`
	Slug      string `json:"slug"`
	CustomKey string `json:"custom_key"`
	ISSN      string `json:"issn"`
	DOI       string `json:"doi"`
	UserID    int64  `json:"user_id"`
	ScreenID  int64  `json:"screen_id"`
	Title     string `json:"title"`
	Abstract  string `json:"abstract"`
	Journal   string `json:"journal"`
	Year      int64  `json:"year"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Section struct {
	ID        int64  `json:"id"`
	PaperID   int64  `json:"paper_id"`
	Order     int64  `json:"order"`
	Header    string `json:"header"`
	Text      string `json:"text"`
	Embedding string `json:"embedding"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
