package grobid

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"regexp"
	"simple-go-app/internal/envHelper"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DOIRegex is the regular expression for extracting DOIs
var DOIRegex = regexp.MustCompile(`\b(10[.][0-9]{3,}(?:[.][0-9]+)*/(?:(?!["&\'])\S)+)\b`)

// GrobidResponse represents the structure of the Grobid service response.
type GrobidResponse struct {
	Title    string   `xml:"teiHeader>fileDesc>titleStmt>title"`
	Date     string   `xml:"teiHeader>fileDesc>publicationStmt>date"`
	Abstract string   `xml:"teiHeader>profileDesc>abstract>div>p"`
	Keywords string   `xml:"teiHeader>profileDesc>textClass>keywords>term"`
	Doi      string   `xml:"teiHeader>fileDesc>sourceDesc>biblStruct>idno[1]"`
	Sections []string `xml:"text>body>div"`
	Authors  []string `xml:"teiHeader>fileDesc>sourceDesc>biblStruct>analytic>author"`
}

type DOI struct {
}

func CheckGrobidHealth(healthStatus *bool, healthMutex *sync.Mutex, fn ...func()) {
	log.Println("Checking Grobid health...")
	healthMutex.Lock()
	healthMutex.Unlock()
	grobidHostname := "grobid"
	grobidPort := "8070"
	GrobidURL := fmt.Sprintf("http://%s:%s", grobidHostname, grobidPort)
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
		fmt.Printf("Waiting %s seconds before starting workers...\n", envHelper.GetEnvVariable("START_DELAY_SECONDS"))
		// Introduce a 15-second delay before updating healthStatus to true
		startDelay := envHelper.GetEnvVariable("START_DELAY_SECONDS")
		// Convert the startDelay string to an int
		startDelayInt, _ := strconv.Atoi(startDelay)
		time.Sleep(time.Duration(startDelayInt) * time.Second)
		// start up workers
		if len(fn) > 0 {
			fn[0]()
		}
	}
	fmt.Println("Setting Grobid health status to", isHealthy)
	*healthStatus = isHealthy
}

func SendPDF2Grobid(fileContent []byte) (*GrobidResponse, error) {
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
	err = writer.WriteField("consolidateHeader", "true")
	if err != nil {
		return nil, err
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	// Make a POST request to the Grobid service endpoint
	resp, err := http.Post("http://grobid:8070/api/processFulltextDocument", writer.FormDataContentType(), &requestBody)
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

	// Parse XML response
	var parsedGrobidResponse GrobidResponse
	err = xml.Unmarshal(grobidResponse, &parsedGrobidResponse)
	if err != nil {
		return nil, err
	}

	//log the response
	log.Printf("Grobid response: %+v\n", parsedGrobidResponse)
	return &parsedGrobidResponse, nil
}

// GetDOIFromString extracts a DOI from a given string
func GetDOIFromString(response []byte, text string) string {
	matches := DOIRegex.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// ExtractKeywords extracts keywords from the Grobid response
func extractKeywords(response []byte, keywords string) []string {
	if keywords != "" {
		// If keywords is set and is not an array, split it by space if after that space there is a capital letter
		keywordsArr := strings.FieldsFunc(keywords, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})

		// Remove empty values and trim each value
		var cleanedKeywords []string
		for _, value := range keywordsArr {
			if value != "" {
				cleanedKeywords = append(cleanedKeywords, strings.TrimSpace(value))
			}
		}

		if len(cleanedKeywords) > 0 {
			return cleanedKeywords
		}
	}

	return nil
}

// ExtractYear extracts the year from the Grobid response date
func (ph *PDFHelper) extractYear(date string) int {
	if date != "" {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err == nil {
			return parsedDate.Year()
		}
	}

	return 0
}
