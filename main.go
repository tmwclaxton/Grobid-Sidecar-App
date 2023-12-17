package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	healthStatus bool
	healthMutex  sync.Mutex
)

func main() {
	// Perform the initial health check on startup

	// Start a timer for periodic health checks
	go func() {
		checkGrobidHealth()
		for {
			time.Sleep(5 * time.Minute) // Adjust the interval as needed
			checkGrobidHealth()
		}
	}()

	//cfg := &config.AppConfig{}

	//awsCfg := &aws.Config{
	//	Region: aws.String("eu-west-2"),
	//}

	// queueReceiver, err := queue.NewReceiver(sess, queue.Config{
	//queueReceiver, err := queue.NewReceiver(sess, queue.Config{
	//	QueueName:            "grobid-queue",
	//	PollingWaitTime:      20,
	//	VisibilityTimeout:    60,
	//	AckRetries:           3,
	//	MaxMessagesToProcess: 10,
	//})

	//if err != nil {
	//	log.Fatal(err)
	//}

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		host, _ := os.Hostname()
		c.JSON(http.StatusOK, gin.H{"hostname": host})
	})

	r.GET("/health", func(c *gin.Context) {
		// Return the global health status
		c.JSON(http.StatusOK, gin.H{"healthy": healthStatus})
	})

	r.Run()
}

func checkGrobidHealth() {
	log.Println("Checking Grobid health...")
	healthMutex.Lock()
	defer healthMutex.Unlock()
	grobidHostname := "grobid"
	grobidPort := "8070"
	grobidURL := fmt.Sprintf("http://%s:%s", grobidHostname, grobidPort)
	healthEndpoint := "/api/isalive"
	// Attempt to make a GET request to the Grobid health endpoint
	resp, err := http.Get(grobidURL + healthEndpoint)
	if err != nil {
		fmt.Println("Error checking Grobid health:", err)
		healthStatus = false
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
		// Introduce a 15-second delay before updating healthStatus to true
		time.Sleep(15 * time.Second)
	}
	fmt.Println("Setting Grobid health status to", isHealthy)
	healthStatus = isHealthy
}
