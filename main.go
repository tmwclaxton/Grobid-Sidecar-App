package main

import (
	"fmt"
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

	"simple-go-app/internal/dispatcher"
	"simple-go-app/internal/envHelper"
	"simple-go-app/internal/grobid"
)

var (
	healthStatus bool
	healthMutex  sync.Mutex
)

func main() {
	// Load environment variables
	envHelper.LoadEnv()

	// Get environment variables
	sqsPrefix := envHelper.GetEnvVariable("SQS_PREFIX")
	requestsQueueName := envHelper.GetEnvVariable("REQUESTS_QUEUE")
	sqsURL := fmt.Sprintf("%s/%s", sqsPrefix, requestsQueueName)
	awsSecretKey := envHelper.GetEnvVariable("AWS_SECRET_ACCESS_KEY")
	awsAccessKey := envHelper.GetEnvVariable("AWS_ACCESS_KEY_ID")
	awsRegion := envHelper.GetEnvVariable("AWS_REGION")
	awsBucket := envHelper.GetEnvVariable("AWS_BUCKET")

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
	go dispatcher.Dispatcher(sqsSvc, sqsURL, messageQueue)

	workFunc := func() {
		// Start three workers
		for i := 1; i <= 5; i++ {
			go dispatcher.Worker(i, messageQueue, sqsSvc, sqsURL, awsBucket)
		}
	}

	// Start a timer for periodic health checks
	go func() {
		grobid.CheckGrobidHealth(&healthStatus, &healthMutex, workFunc)
		for {
			time.Sleep(5 * time.Minute) // Adjust the interval as needed
			grobid.CheckGrobidHealth(&healthStatus, &healthMutex)
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
