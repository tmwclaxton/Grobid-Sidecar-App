package grobid

import (
	"fmt"
	"io"
	"log"
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
	grobidURL := fmt.Sprintf("http://%s:%s", grobidHostname, grobidPort)
	healthEndpoint := "/api/isalive"
	// Attempt to make a GET request to the Grobid health endpoint
	resp, err := http.Get(grobidURL + healthEndpoint)
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
