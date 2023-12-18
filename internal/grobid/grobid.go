package grobid

import (
	"bytes"
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

func SendPDF2Grobid(fileContent []byte) ([]byte, error) {
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
	fmt.Println("Grobid successfully processed the file")
	if err != nil {
		return nil, err
	}

	return grobidResponse, nil
}
