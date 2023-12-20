package parsing

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type CrudeCrossRefResponse struct {
	Raw   string `xml:",innerxml"`
	Title string `xml:"query_result>body>query>doi_record>crossref>journal>journal_article>titles>title"`
	Year  string `xml:"query_result>body>query>doi_record>crossref>journal>journal_issue>publication_date>year"`
}

type AbstractTemp struct {
	JATS []JATs `xml:"query_result>body>query>doi_record>crossref>journal>journal_article"`
}

type JATs struct {
	RawContent string `xml:",innerxml"`
}

type TidyCrossRefResponse struct {
	Title    string `json:"title"`
	Year     string `json:"year"`
	Abstract string `xml:"abstract>p"`
}

func CrossReferenceData(doi string) (*TidyCrossRefResponse, error) {
	log.Printf("Cross referencing data for DOI: %s\n", doi)

	client := &http.Client{}

	var response *http.Response
	var err error

	response, err = client.Get("https://api.crossref.org/works/" + doi + "/transform/application/vnd.crossref.unixsd+xml")
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing Crossref response body:", err)
		}
	}(response.Body)

	xmlBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	//log.Printf("Crossref response: %s\n", xmlBytes)

	var crudeResponse CrudeCrossRefResponse
	err = xml.Unmarshal(xmlBytes, &crudeResponse)
	if err != nil {
		return nil, err
	}

	tidyCrossRefResponse := TidyCrossRefData(&crudeResponse, &AbstractTemp{})

	return tidyCrossRefResponse, nil
}

func TidyCrossRefData(crudeResponse *CrudeCrossRefResponse, AbstractTemp *AbstractTemp) *TidyCrossRefResponse {
	// if there is more than one journal article find the one with the namespace or prefix of http://www.ncbi.nlm.nih.gov/JATS1
	var abstract string
	if len(AbstractTemp.JATS) > 1 {
		for _, journalArticle := range AbstractTemp.JATS {
			if journalArticle.RawContent[:len("http://www.ncbi.nlm.nih.gov/JATS1")] == "http://www.ncbi.nlm.nih.gov/JATS1" {
				log.Printf("Found journal article with namespace: %s\n", journalArticle.RawContent[:len("http://www.ncbi.nlm.nih.gov/JATS1")])
				abstract = journalArticle.RawContent
			}
		}
	}

	if abstract == "" {
		abstract = strings.TrimSpace(abstract)
		re := regexp.MustCompile(`\s+`)
		abstract = re.ReplaceAllString(abstract, " ")
	}

	return &TidyCrossRefResponse{
		Title:    crudeResponse.Title,
		Year:     crudeResponse.Year,
		Abstract: abstract,
	}
}
