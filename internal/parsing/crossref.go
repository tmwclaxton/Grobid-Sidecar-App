package parsing

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type CrudeCrossRefDOIResponse struct {
	Raw   string `xml:",innerxml"`
	Title string `xml:"query_result>body>query>doi_record>crossref>journal>journal_article>titles>title"`
	Year  string `xml:"query_result>body>query>doi_record>crossref>journal>journal_issue>publication_date>year"`
	DOI   string `xml:"query_result>body>query>doi_record>crossref>journal>journal_article>doi_data>doi"`
}

type AbstractTemp struct {
	JATS []JATs `xml:"query_result>body>query>doi_record>crossref>journal>journal_article"`
}

type JATs struct {
	RawContent string `xml:",innerxml"`
}

type CrudeCrossRefTitleResponse struct {
	Raw   string `xml:",innerxml"`
	Title string `xml:"query_result>body>query>doi_record>crossref>journal>journal_article>titles>title"`
}

type TidyCrossRefResponse struct {
	Title    string `json:"title"`
	Year     string `json:"year"`
	Abstract string `xml:"abstract>p"`
	DOI      string `json:"doi"`
	ISSN     string `json:"issn"`
}

// CrossRefTitleResponse god i love json
type CrossRefTitleResponse struct {
	Message struct {
		Items []struct {
			DOI      string   `json:"DOI"`
			Title    []string `json:"title"`
			Abstract string   `json:"abstract"`
			ISSN     []string `json:"ISSN"`
			Issued   struct {
				DateParts [][]int `json:"date-parts"`
			} `json:"issued"`
		} `json:"items"`
	} `json:"message"`
}

func CrossRefDataDOI(doi string) (*TidyCrossRefResponse, error) {
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

	var crudeResponse CrudeCrossRefDOIResponse
	err = xml.Unmarshal(xmlBytes, &crudeResponse)
	if err != nil {
		return nil, err
	}

	tidyCrossRefResponse := TidyCrossRefDOIData(&crudeResponse, &AbstractTemp{})

	return tidyCrossRefResponse, nil
}

func TidyCrossRefDOIData(crudeResponse *CrudeCrossRefDOIResponse, AbstractTemp *AbstractTemp) *TidyCrossRefResponse {
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

	if abstract != "" {
		abstract = strings.TrimSpace(abstract)
		re := regexp.MustCompile(`\s+`)
		abstract = re.ReplaceAllString(abstract, " ")
	}

	return &TidyCrossRefResponse{
		Title:    crudeResponse.Title,
		Year:     crudeResponse.Year,
		DOI:      crudeResponse.DOI,
		Abstract: abstract,
	}
}
func CrossRefDataTitle(title string) (*TidyCrossRefResponse, error) {
	log.Printf("Cross referencing data for title: %s\n", title)

	client := &http.Client{}

	var response *http.Response
	var err error

	url := "https://api.crossref.org/works?query.bibliographic=" + title + "&rows=1&offset=0"

	// escape the url
	url = strings.ReplaceAll(url, " ", "%20")

	log.Printf("Crossref URL: %s\n", url)
	response, err = client.Get(url)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing Crossref response body:", err)
		}
	}(response.Body)

	// Parse JSON response
	var crossRefResponse CrossRefTitleResponse
	err = json.NewDecoder(response.Body).Decode(&crossRefResponse)
	if err != nil {
		return nil, err
	}

	if len(crossRefResponse.Message.Items) == 0 {
		return nil, fmt.Errorf("No matching items found for title: %s", title)
	}

	// Extract data from the response
	item := crossRefResponse.Message.Items[0]
	tidyResponse := &TidyCrossRefResponse{
		Title:    item.Title[0],
		Year:     fmt.Sprintf("%d", item.Issued.DateParts[0][0]), // assuming the date-parts contain the year
		Abstract: item.Abstract,
		DOI:      item.DOI,
		ISSN:     item.ISSN[0],
	}

	return tidyResponse, nil
}
