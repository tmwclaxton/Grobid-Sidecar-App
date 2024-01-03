package parsing

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type PubMedResponse struct {
	Header struct {
		Type    string `json:"type"`
		Version string `json:"version"`
	}
	ESearchResult struct {
		Count    string   `json:"count"`
		RetMax   string   `json:"retmax"`
		RetStart string   `json:"retstart"`
		IDList   []string `json:"idlist"`
	}
}

func GetPubMedIDFromDOI(doi string) (any, error) {
	if doi == "" {
		return nil, fmt.Errorf("DOI is empty")
	}

	// Make a GET request to the PubMed service endpoint
	resp, err := http.Get("https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&term=" + doi + "&retmode=json")
	if err != nil {
		log.Println("Error getting PubMed ID:", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing PubMed response body:", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Println("PubMed service returned non-OK status:", resp.Status)
		return nil, fmt.Errorf("PubMed service returned non-OK status: %v", resp.Status)
	}

	// Read PubMed service response
	pubMedResponse, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading PubMed response:", err)
		return nil, err
	}

	// Parse JSON response
	var parsedPubMedResponse PubMedResponse
	err = json.Unmarshal(pubMedResponse, &parsedPubMedResponse)
	if err != nil {
		log.Println("Error parsing PubMed response:", err)
		return nil, err
	}

	if len(parsedPubMedResponse.ESearchResult.IDList) > 0 {
		pubMedID, err := strconv.Atoi(parsedPubMedResponse.ESearchResult.IDList[0])
		if err != nil {
			log.Println("Error converting PubMed ID to integer:", err)
			return nil, err
		}
		log.Printf("PubMed ID: %v\n", pubMedID)
		return pubMedID, nil
	}

	return nil, nil
}
