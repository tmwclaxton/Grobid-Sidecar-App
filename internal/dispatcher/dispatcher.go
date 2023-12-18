package dispatcher

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
)

func Dispatcher(svc *sqs.SQS, sqsURL string, messageQueue chan<- *sqs.Message) {
	log.Println("Starting dispatcher...")
	for {
		result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(sqsURL),
			MaxNumberOfMessages: aws.Int64(10),
			VisibilityTimeout:   aws.Int64(30),
			WaitTimeSeconds:     aws.Int64(20),
		})
		if err != nil {
			log.Println("Error receiving message:", err)
			continue
		}

		for _, message := range result.Messages {
			messageQueue <- message
		}
	}
}
