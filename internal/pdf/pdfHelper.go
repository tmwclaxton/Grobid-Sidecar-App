package helpers

import (
	"encoding/xml"
	"fmt"
	"log"
)

//var DOIRegex = regexp.MustCompile(`10\.\d{4,9}\/[-._;()/:A-Z0-9]+`)

type PDFDTO struct {
	Title    string      `json:"title"`
	DOI      string      `json:"doi"`
	Abstract string      `json:"abstract"`
	Sections interface{} `json:"sections"`
	Keywords interface{} `json:"keywords"`
	Authors  interface{} `json:"authors"`
	Year     interface{} `json:"year"`
	Data     interface{} `json:"data"`
}

// TeIHeader Define structs to mirror the structure of the Grobid XML response
type TeIHeader struct {
	FileDesc    FileDesc    `xml:"fileDesc"`
	ProfileDesc ProfileDesc `xml:"profileDesc"`
}

type FileDesc struct {
	SourceDesc      SourceDesc      `xml:"sourceDesc"`
	TitleStmt       TitleStmt       `xml:"titleStmt"`
	PublicationStmt PublicationStmt `xml:"publicationStmt"`
}

type SourceDesc struct {
	BiblStruct BiblStruct `xml:"biblStruct"`
}

type BiblStruct struct {
	Idno     []string `xml:"idno"`
	Analytic Analytic `xml:"analytic"`
}

type Analytic struct {
	Author []string `xml:"author"`
}

type TitleStmt struct {
	Title string `xml:"title"`
}

type ProfileDesc struct {
	TextClass TextClass `xml:"textClass"`
	Abstract  Abstract  `xml:"abstract"`
}

type TextClass struct {
	Keywords Keywords `xml:"keywords"`
}

type Keywords struct {
	Term []string `xml:"term"`
}

type Abstract struct {
	Div Div `xml:"div"`
}

type Div struct {
	P string `xml:"p"`
}

type PublicationStmt struct {
	Date string `xml:"date"`
}

func ParsePDF(response []byte) (PDFDTO, error) {
	fmt.Println(string(response[100:200]))

	log.Println("Parsing PDF...")
	var dto PDFDTO
	var parsedResponse TeIHeader

	// Takes a well-formed XML string and returns a parsed response

	err := xml.Unmarshal([]byte(response), &parsedResponse)
	if err != nil {
		log.Println("Error parsing XML response:", err)
		return dto, err
	}
	//log.Printf(" response: %+v\n", response)
	log.Printf("Parsed response: %+v\n", parsedResponse)

	//dto.DOI = getDOIFromString(fmt.Sprintf("%v", parsedResponse.FileDesc.SourceDesc.BiblStruct.Idno[1]))
	dto.Keywords = parsedResponse.ProfileDesc.TextClass.Keywords.Term
	dto.Title = parsedResponse.FileDesc.TitleStmt.Title
	dto.Abstract = parsedResponse.ProfileDesc.Abstract.Div.P
	log.Printf("dto.Abstract: %+v\n", parsedResponse.FileDesc.TitleStmt.Title)
	// Extracting authors
	var authors []string
	for _, author := range parsedResponse.FileDesc.SourceDesc.BiblStruct.Analytic.Author {
		authors = append(authors, author)
	}
	dto.Authors = authors

	// Extracting year
	year := parsedResponse.FileDesc.PublicationStmt.Date
	if year != "" {
		//dto.Year = carbon.Parse(year).Year
		dto.Year = year
	}

	log.Printf("DTO: %+v\n", dto)
	// Add other fields mapping here

	log.Println("PDF parsed successfully.")
	return dto, nil
}

//func getDOIFromString(text string) string {
//	matches := DOIRegex.FindStringSubmatch(text)
//	if len(matches) > 0 {
//		return matches[0]
//	}
//	return ""
//}
