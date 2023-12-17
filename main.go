package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	healthStatus bool
	healthMutex  sync.Mutex
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		// Not fatal, just log the error and continue
		log.Println("Couldn't load .env file:", err)
	}
}

func getEnvVariable(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s not set", key)
	}
	return value
}

func main() {
	// Load environment variables
	loadEnv()

	// Get environment variables
	sqsPrefix := getEnvVariable("SQS_PREFIX")
	requestsQueueName := getEnvVariable("REQUESTS_QUEUE")
	sqsURL := fmt.Sprintf("%s/%s", sqsPrefix, requestsQueueName)
	awsSecretKey := getEnvVariable("AWS_SECRET_ACCESS_KEY")
	awsAccessKey := getEnvVariable("AWS_ACCESS_KEY_ID")
	awsRegion := getEnvVariable("AWS_REGION")
	awsBucket := getEnvVariable("AWS_BUCKET")

	log.Println("Environment variables loaded successfully.")

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
	for i := 1; i <= 5; i++ {
		go worker(i, messageQueue, sqsSvc, sqsURL, awsBucket)
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
			MaxNumberOfMessages: aws.Int64(10),
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

func worker(id int, messageQueue <-chan *sqs.Message, svc *sqs.SQS, sqsURL string, s3Bucket string) {
	log.Printf("Starting worker %d...\n", id)
	for {
		// Wait for a message
		message := <-messageQueue

		// parse message
		// Parse the JSON message
		var msgData map[string]interface{}
		if err := json.Unmarshal([]byte(*message.Body), &msgData); err != nil {
			log.Println("Error decoding JSON message:", err)
			continue
		}

		path := msgData["s3Location"].(string)
		operation := msgData["operation"].(string)

		//Print the path or use it as needed
		fmt.Printf("Worker %d received message. Path: %s. Operation: %s\n", id, path, operation)

		// Simulate processing time
		time.Sleep(time.Minute) // remove this later <--------------------------

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
