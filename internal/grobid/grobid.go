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
	"simple-go-app/internal/envHelper"
	"strconv"
	"sync"
	"time"
)

// DOIRegex is the regular expression for extracting DOIs
//var DOIRegex = regexp.MustCompile(`\b(10[.][0-9]{3,}(?:[.][0-9]+)*/(?:(?!["&\'])\S)+)\b`)

// CrudeGrobidResponse represents the structure of the Grobid service response.
type CrudeGrobidResponse struct {
	Doi      string   `xml:"teiHeader>fileDesc>sourceDesc>biblStruct>idno[1]"`
	Keywords []string `xml:"teiHeader>profileDesc>textClass>keywords>term"`
	Title    string   `xml:"teiHeader>fileDesc>titleStmt>title"`
	Date     string   `xml:"teiHeader>fileDesc>publicationStmt>date"`
	Abstract string   `xml:"teiHeader>profileDesc>abstract>div>p"`
	//Sections []xml.CharData `xml:"text>body>div"`
	Sections []GrobidSection `xml:"text>body>div"`
	Authors  string          `xml:"teiHeader>fileDesc>sourceDesc>biblStruct>analytic>author"`
}

type GrobidSection struct {
	Head       string `xml:"head"`
	RawContent string `xml:",innerxml"`
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
	println(string(grobidResponse))

	// Parse XML response
	var parsedGrobidResponse CrudeGrobidResponse
	err = xml.Unmarshal(grobidResponse, &parsedGrobidResponse)
	if err != nil {
		return nil, err
	}

	//log the response
	log.Printf("Grobid response: %+v\n", parsedGrobidResponse)

	// Iterate through all the sections and print the length
	for i, section := range parsedGrobidResponse.Sections {
		fmt.Printf("Section %d length: %d\n", i+1, len(section.RawContent))

		// Print the raw XML content as-is
		fmt.Printf("Section %d Raw Content:\n%s\n", i+1, section.RawContent)

		// Access the head of the section if needed
		fmt.Printf("Section %d Head: %s\n", i+1, section.Head)
	}

	fmt.Println("First keyword: ", parsedGrobidResponse.Keywords[0])
	return &parsedGrobidResponse, nil
}
