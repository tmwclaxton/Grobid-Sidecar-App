package dispatcher

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func Worker(id int, messageQueue <-chan *sqs.Message, svc *sqs.SQS, sqsURL string, s3Bucket string) {
	log.Printf("Starting worker %d...\n", id)
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

		time.Sleep(time.Minute)

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
