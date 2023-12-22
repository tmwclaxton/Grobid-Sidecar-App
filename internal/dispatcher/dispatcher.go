package dispatcher

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"log"
	"simple-go-app/internal/helpers"
	"strconv"
)

func Dispatcher(svc *sqs.SQS, sqsURL string, messageQueue chan<- *sqs.Message) {
	log.Println("Starting dispatcher...")
	maxNumberOfMessagesInt64, _ := strconv.ParseInt(helpers.GetEnvVariable("DISPATCHER_MAX_MESSAGES"), 10, 64)
	visibilityTimeoutInt64, _ := strconv.ParseInt(helpers.GetEnvVariable("DISPATCHER_VISIBILITY_TIMEOUT"), 10, 64)
	waitTimeSecondsInt64, _ := strconv.ParseInt(helpers.GetEnvVariable("DISPATCHER_WAIT_TIME_SECONDS"), 10, 64)
	for {
		result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(sqsURL),
			MaxNumberOfMessages: aws.Int64(maxNumberOfMessagesInt64),
			VisibilityTimeout:   aws.Int64(visibilityTimeoutInt64),
			WaitTimeSeconds:     aws.Int64(waitTimeSecondsInt64),
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
