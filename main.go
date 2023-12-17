package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	healthStatus bool
	healthMutex  sync.Mutex
)

func main() {
	err := godotenv.Load()
	if err != nil {
		// not fatal just continue
		log.Println("Couldn't loading .env file:", err)
	}

	// using os.Getenv() to get the environment variable
	sqsPrefix := os.Getenv("SQS_PREFIX")
	requestsQueueName := os.Getenv("REQUESTS_QUEUE")
	sqsURL := fmt.Sprintf("%s/%s", sqsPrefix, requestsQueueName)
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsRegion := os.Getenv("AWS_REGION")
	if sqsURL == "" {
		log.Fatal("SQS URL not set")
	}
	if awsAccessKey == "" {
		log.Fatal("AWS access key not set")
	}
	if awsSecretKey == "" {
		log.Fatal("AWS secret key not set")
	}
	if awsRegion == "" {
		log.Fatal("AWS region not set")
	}

	// Set up AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
	})
	if err != nil {
		log.Fatal("Error creating AWS session:", err)
	}

	// Set up the queue service
	sqsSvc := sqs.New(sess)

	// Create a channel for communication between dispatcher and workers
	messageQueue := make(chan *sqs.Message, 10) // Adjust the buffer size as needed

	// Start dispatcher
	go dispatcher(sqsSvc, sqsURL, messageQueue)

	// Start three workers
	for i := 1; i <= 3; i++ {
		go worker(i, messageQueue, sqsSvc, sqsURL)
	}

	// Start a timer for periodic health checks
	go func() {
		checkGrobidHealth()
		for {
			time.Sleep(5 * time.Minute) // Adjust the interval as needed
			checkGrobidHealth()
		}
	}()

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

func dispatcher(svc *sqs.SQS, sqsURL string, messageQueue chan<- *sqs.Message) {
	log.Println("Starting dispatcher...")
	for {
		// Receive message from SQS
		result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(sqsURL),
			MaxNumberOfMessages: aws.Int64(20),
			VisibilityTimeout:   aws.Int64(30), // Adjust the visibility timeout as needed
			WaitTimeSeconds:     aws.Int64(20),
		})
		if err != nil {
			log.Println("Error receiving message:", err)
			continue
		}

		// Enqueue message for workers
		for _, message := range result.Messages {
			messageQueue <- message
		}
	}
}

func worker(id int, messageQueue <-chan *sqs.Message, svc *sqs.SQS, sqsURL string) {
	for {
		// Wait for a message
		message := <-messageQueue

		// Process the message (print its contents)
		fmt.Printf("Worker %d received message: %s\n", id, *message.Body)

		// Simulate processing time
		time.Sleep(time.Minute)

		// Put the message back to the queue
		_, err := svc.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
			QueueUrl:          aws.String(sqsURL),
			ReceiptHandle:     message.ReceiptHandle,
			VisibilityTimeout: aws.Int64(30),
		})
		if err != nil {
			log.Println("Error putting message back to the queue:", err)
		}
	}
}
