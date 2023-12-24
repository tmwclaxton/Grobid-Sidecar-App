package parsing

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"simple-go-app/internal/helpers"
	"strconv"
	"time"

	//"github.com/uniplaces/carbon"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// CrudeGrobidResponse represents the structure of the Grobid service response.
type CrudeGrobidResponse struct {
	Raw      string       `xml:",innerxml"`
	IDNOs    []IdnosRaw   `xml:"teiHeader>fileDesc>sourceDesc>biblStruct>idno"`
	Keywords KeywordsRaw  `xml:"teiHeader>profileDesc>textClass>keywords"`
	Title    string       `xml:"teiHeader>fileDesc>titleStmt>title"`
	Date     string       `xml:"teiHeader>fileDesc>publicationStmt>date"`
	Abstract string       `xml:"teiHeader>profileDesc>abstract>div>p"`
	Sections []SectionRaw `xml:"text>body>div"`
	Authors  []AuthorsRaw `xml:"teiHeader>fileDesc>sourceDesc>biblStruct>analytic>author"`
}

type TidyGrobidResponse struct {
	Doi      string       `json:"doi"`
	Keywords []string     `json:"keywords"`
	Title    string       `json:"title"`
	Date     string       `json:"date"`
	Year     string       `json:"year"`
	Abstract string       `json:"abstract"`
	Sections []SectionRaw `json:"sections"`
	Authors  []AuthorsRaw `json:"authors"`
	Journal  string       `json:"journal"`
	Notes    string       `json:"notes"`
}

type IdnosRaw struct {
	RawContent string `xml:",innerxml"`
}

type SectionRaw struct {
	RawContent string   `xml:",innerxml"`
	Head       string   `xml:"head"`
	P          []string `xml:"p"`
}

type KeywordsRaw struct {
	Term       []string `xml:"term"`
	RawContent string   `xml:",innerxml"`
}

type AuthorsRaw struct {
	RawContent string `xml:",innerxml"`
}

func CheckGrobidHealth(healthStatus *bool, healthMutex *sync.Mutex, fn ...func()) {
	fmt.Printf("Waiting %s seconds before starting up sidecar...\n", helpers.GetEnvVariable("START_DELAY_SECONDS"))
	// Introduce a 15-second delay before updating healthStatus to true
	startDelay := helpers.GetEnvVariable("START_DELAY_SECONDS")
	// Convert the startDelay string to an int
	startDelayInt, _ := strconv.Atoi(startDelay)
	time.Sleep(time.Duration(startDelayInt) * time.Second)

	log.Println("Checking Grobid health...")
	healthMutex.Lock()
	healthMutex.Unlock()
	GrobidURL := helpers.GetEnvVariable("GROBID_URL")
	healthEndpoint := "/api/isalive"
	// Attempt to make a GET request to the Grobid health endpoint
	resp, err := http.Get(GrobidURL + healthEndpoint)
	if err != nil {
		fmt.Println("Error checking Grobid health:", err)
		*healthStatus = false
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing Grobid response body:", err)
		}
	}(resp.Body)

	// Check if the response status code is within the 2xx range
	isHealthy := resp.StatusCode >= 200 && resp.StatusCode < 300

	if isHealthy {
		// start up workers
		if len(fn) > 0 {
			fn[0]()
		}
	}
	fmt.Println("Setting Grobid health status to", isHealthy)
	*healthStatus = isHealthy
}

func SendPDF2Grobid(fileContent []byte) (*CrudeGrobidResponse, error) {
	// Create a buffer to store the multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add the file field to the request
	fileField, err := writer.CreateFormField("input")
	if err != nil {
		return nil, err
	}

	// Copy the file content to the form file field
	_, err = io.Copy(fileField, bytes.NewReader(fileContent))
	if err != nil {
		return nil, err
	}

	// Add other form fields
	err = writer.WriteField("consolidateHeader", "1")
	if err != nil {
		return nil, err
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	GrobidURL := helpers.GetEnvVariable("GROBID_URL")
	// Make a POST request to the Grobid service endpoint
	resp, err := http.Post(GrobidURL+"/api/processFulltextDocument", writer.FormDataContentType(), &requestBody)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing Grobid response body:", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grobid service returned non-OK status: %v", resp.Status)
	}

	// Read Grobid service response
	grobidResponse, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println("Grobid successfully processed the file")
	//println(string(grobidResponse))

	// Parse XML response
	var parsedGrobidResponse CrudeGrobidResponse
	err = xml.Unmarshal(grobidResponse, &parsedGrobidResponse)
	if err != nil {
		return nil, err
	}

	return &parsedGrobidResponse, nil
}

func TidyUpGrobidResponse(crudeResponse *CrudeGrobidResponse) (*TidyGrobidResponse, error) {
	var tidyResponse TidyGrobidResponse

	//log.Printf("Crude IDNOs: %s\n", crudeResponse.IDNOs[1].RawContent)

	if len(crudeResponse.IDNOs) > 1 {
		tidyResponse.Doi = GetDOIFromString(crudeResponse.IDNOs[1].RawContent)
	} else {
		tidyResponse.Doi = GetDOIFromString(crudeResponse.IDNOs[0].RawContent)
	}
	tidyResponse.Keywords = crudeResponse.Keywords.Term

	// if keywords are empty, try to extract them from raw content
	if len(tidyResponse.Keywords) == 0 {
		tidyResponse.Keywords = extractKeywordsFromRawContent(crudeResponse.Keywords.RawContent)
	}

	tidyResponse.Title = crudeResponse.Title
	tidyResponse.Date = crudeResponse.Date
	if tidyResponse.Date != "" {
		// Hopefully the date is in this format  4 July 2020
		tidyResponse.Year = tidyResponse.Date[len(tidyResponse.Date)-4:]
	}
	tidyResponse.Abstract = crudeResponse.Abstract
	for _, section := range crudeResponse.Sections {
		tidyResponse.Sections = append(tidyResponse.Sections, section)
	}
	for _, author := range crudeResponse.Authors {
		tidyResponse.Authors = append(tidyResponse.Authors, author)
	}
	return &tidyResponse, nil
}

func extractKeywordsFromRawContent(content string) []string {
	// Check if the content is empty
	if content == "" {
		return nil
	}

	// Use regular expression to extract keywords
	keywordRegex := regexp.MustCompile(`(?i)<term>(.*?)</term>`)
	matches := keywordRegex.FindAllStringSubmatch(content, -1)

	// If matches are found, extract keywords
	if len(matches) > 0 {
		var keywords []string
		for _, match := range matches {
			if len(match) > 1 {
				keywords = append(keywords, match[1])
			}
		}

		// Split keywords by space if after that space there is a capital letter
		var finalKeywords []string
		for _, keyword := range keywords {
			splitKeywords := splitKeywordsBySpace(keyword)
			finalKeywords = append(finalKeywords, splitKeywords...)
		}

		// Remove empty values and trim each value
		finalKeywords = removeEmptyAndTrim(finalKeywords)

		return finalKeywords
	}

	return nil
}

// Split keywords by space if after that space there is a capital letter
func splitKeywordsBySpace(keyword string) []string {
	var result []string
	words := strings.Fields(keyword)

	for i := 0; i < len(words)-1; i++ {
		currentWord := words[i]
		nextWord := words[i+1]

		// Check if the current word ends with a space and the next word starts with a capital letter
		if strings.HasSuffix(currentWord, " ") && len(nextWord) > 0 && nextWord[0] >= 'A' && nextWord[0] <= 'Z' {
			// Combine the two words into a single keyword
			result = append(result, strings.TrimSpace(currentWord+nextWord))
		}
	}

	// Add the last word
	result = append(result, words[len(words)-1])

	return result
}

// Remove empty values and trim each value
func removeEmptyAndTrim(keywords []string) []string {
	var result []string
	for _, keyword := range keywords {
		trimmedKeyword := strings.TrimSpace(keyword)
		if trimmedKeyword != "" {
			result = append(result, trimmedKeyword)
		}
	}
	return result
}

// GetDOIFromString Get DOI from string
func GetDOIFromString(content string) string {
	// Check if the content is empty
	if content == "" {
		log.Println("DOI string is empty")
		return ""
	}

	// Use regular expression to extract DOI
	DOIRegex := regexp.MustCompile(`\b(10\.[0-9]{4,}(?:\.[0-9]+)*/\S+)\b`)
	matches := DOIRegex.FindStringSubmatch(content)

	// If matches are found, extract DOI
	if len(matches) > 0 {
		return matches[0]
	}

	log.Println("DOI not found")
	return ""
}
