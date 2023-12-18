package dispatcher

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"simple-go-app/internal/envHelper"
	"simple-go-app/internal/grobid"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	lastRequestTime   time.Time
	lastRequestTimeMu sync.Mutex
)

func Worker(id int, messageQueue <-chan *sqs.Message, svc *sqs.SQS, sqsURL string, s3Bucket string) {

	awsRegion := envHelper.GetEnvVariable("AWS_REGION")

	minGapBetweenRequests := envHelper.GetEnvVariable("MINIMUM_GAP_BETWEEN_REQUESTS_SECONDS")
	minGap, err := time.ParseDuration(minGapBetweenRequests + "s")
	if err != nil {
		log.Fatalf("Error parsing MINIMUM_GAP_BETWEEN_REQUESTS_SECONDS: %v", err)
	}

	log.Printf("Starting worker %d...\n", id)
	// Create an AWS session with the specified region
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	}))
	s3Svc := s3.New(sess)

	for {
		message := <-messageQueue

		var msgData map[string]interface{}
		if err := json.Unmarshal([]byte(*message.Body), &msgData); err != nil {
			log.Println("Error decoding JSON message:", err)
			continue
		}

		path := msgData["s3Location"].(string)
		operation := msgData["operation"].(string)

		fmt.Printf("Worker %d received message. Path: %s. Operation: %s\n", id, path, operation)

		// Ensure a minimum gap between requests
		lastRequestTimeMu.Lock()
		timeSinceLastRequest := time.Since(lastRequestTime)
		lastRequestTimeMu.Unlock()

		if timeSinceLastRequest < minGap {
			sleepTime := minGap - timeSinceLastRequest
			log.Printf("Worker %d sleeping for %v to meet the minimum gap between requests\n", id, sleepTime)
			time.Sleep(sleepTime)
		}

		// Download the file from S3
		output, err := s3Svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(s3Bucket),
			Key:    aws.String(path),
		})
		if err != nil {
			log.Println("Error downloading file from S3:", err)
			log.Printf("Bucket: %s, Key: %s\n", s3Bucket, path)
			continue
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Println("Error closing S3 response body:", err)
			}
		}(output.Body)

		fileContent, err := ioutil.ReadAll(output.Body)
		if err != nil {
			log.Println("Error reading file content:", err)
			continue
		}

		// Send the file content to the Grobid service
		response, err := grobid.SendPDF2Grobid(fileContent)

		// print a snippet of the response
		fmt.Println(string(response[0:100]))

		if err != nil {
			log.Println("Error sending file to Grobid service:", err)
		}

		// Update the last request time
		lastRequestTimeMu.Lock()
		lastRequestTime = time.Now()
		lastRequestTimeMu.Unlock()

		// Change message visibility or delete the message from the SQS queue
		// based on your application logic

		// For example, changing message visibility:
		_, err = svc.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
			QueueUrl:          aws.String(sqsURL),
			ReceiptHandle:     message.ReceiptHandle,
			VisibilityTimeout: aws.Int64(30),
		})
		if err != nil {
			log.Println("Error putting message back to the queue:", err)
		}
	}
}
