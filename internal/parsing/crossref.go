package parsing

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type TidyCrossRefResponse struct {
	Title    string `json:"title"`
	Year     string `json:"year"`
	Abstract string `json:"abstract"`
	DOI      string `json:"doi"`
	ISSN     string `json:"issn"`
}

// CrossRefDOIResponse god i love json
type CrossRefDOIResponse struct {
	Message struct {
		DOI      string   `json:"DOI"`
		Title    []string `json:"title"`
		Abstract string   `json:"abstract"`
		ISSN     []string `json:"ISSN"`
		Issued   struct {
			DateParts [][]int `json:"date-parts"`
		}
	} `json:"message"`
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

	response, err = client.Get("https://api.crossref.org/works/" + doi)
	if err != nil {
		return &TidyCrossRefResponse{}, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing Crossref response body:", err)
		}
	}(response.Body)

	// Parse JSON response
	var crossRefResponse CrossRefDOIResponse
	err = json.NewDecoder(response.Body).Decode(&crossRefResponse)
	if err != nil {
		return &TidyCrossRefResponse{}, err
	}

	// Extract data from the response
	item := crossRefResponse.Message
	tidyCrossRefResponse := &TidyCrossRefResponse{
		Title:    item.Title[0],
		Year:     fmt.Sprintf("%d", item.Issued.DateParts[0][0]), // assuming the date-parts contain the year
		Abstract: item.Abstract,
		DOI:      item.DOI,
		ISSN:     item.ISSN[0],
	}

	if tidyCrossRefResponse.Abstract != "" {
		tidyCrossRefResponse.Abstract = strings.TrimSpace(tidyCrossRefResponse.Abstract)
		re := regexp.MustCompile(`\s+`)
		tidyCrossRefResponse.Abstract = re.ReplaceAllString(tidyCrossRefResponse.Abstract, " ")
	}

	return tidyCrossRefResponse, nil
}

func CrossRefDataTitle(title string) (*TidyCrossRefResponse, error) {
	log.Printf("Cross referencing data for title: %s\n", title)

	client := &http.Client{}

	var response *http.Response
	var err error

	url := "https://api.crossref.org/works?query.bibliographic=" + title + "&rows=1&offset=0"

	// escape the url
	url = strings.ReplaceAll(url, " ", "%20")

	//log.Printf("Crossref URL: %s\n", url)
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
		DOI:      item.DOI,
		Abstract: item.Abstract,
	}

	// Check if Title array is not empty before accessing the first element
	if len(item.Title) > 0 {
		tidyResponse.Title = item.Title[0]
	}

	// Check if ISSN array is not empty before accessing the first element
	if len(item.ISSN) > 0 {
		tidyResponse.ISSN = item.ISSN[0]
	}

	// Check if Issued array is not empty and contains the expected date-parts before accessing the first element
	if len(item.Issued.DateParts) > 0 && len(item.Issued.DateParts[0]) > 0 {
		tidyResponse.Year = fmt.Sprintf("%d", item.Issued.DateParts[0][0])
	}

	return tidyResponse, nil
}
