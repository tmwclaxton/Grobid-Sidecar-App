package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"simple-go-app/internal/helpers"
	"simple-go-app/internal/logging"
	"simple-go-app/internal/parsing"
	"simple-go-app/internal/store"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sqs"
	"golang.org/x/sync/semaphore"
)

var (
	lastRequestTime   time.Time
	lastRequestTimeMu sync.Mutex
	totalRequests     = 0
	grobidSemaphore   = semaphore.NewWeighted(1)
)

func Worker(id int, messageQueue <-chan *sqs.Message, svc *sqs.SQS, sqsURL, s3Bucket string, s *store.Store) {
	awsRegion := helpers.GetEnvVariable("AWS_REGION")
	minGapBetweenRequests, err := time.ParseDuration(helpers.GetEnvVariable("MINIMUM_GAP_BETWEEN_REQUESTS_SECONDS") + "s")
	if err != nil {
		log.Fatalf("Error parsing MINIMUM_GAP_BETWEEN_REQUESTS_SECONDS: %v", err)
	}
	gracePeriodRequests, _ := strconv.Atoi(helpers.GetEnvVariable("GRACE_PERIOD_REQUESTS"))
	allowedWorkers, _ := strconv.Atoi(helpers.GetEnvVariable("GRACE_PERIOD_WORKERS"))

	log.Printf("Starting worker %d...\n", id)

	for {
		pass := true

		if totalRequests < gracePeriodRequests {
			// if worker id is greater than the allowed workers then return
			if id > allowedWorkers {
				pass = false
			}
			if pass {
				// Acquire a semaphore before accessing
				if err := grobidSemaphore.Acquire(context.Background(), 1); err != nil {
					log.Printf("Worker %d could not acquire semaphore: %v\n", id, err)
					pass = false
				}
				lastRequestTimeMu.Lock()
				timeSinceLastRequest := time.Since(lastRequestTime)
				lastRequestTimeMu.Unlock()
				//log.Printf("Worker %d acquired semaphore\n", id)

				// If the time since the last request is less than the minimum gap between requests, sleep for the difference
				if timeSinceLastRequest < minGapBetweenRequests {
					sleepTime := minGapBetweenRequests - timeSinceLastRequest
					//log.Printf("Worker %d sleeping for %v to meet the minimum gap between requests\n", id, sleepTime)
					time.Sleep(sleepTime)
				}
				lastRequestTimeMu.Lock()
				lastRequestTime = time.Now()
				lastRequestTimeMu.Unlock()
				grobidSemaphore.Release(1) // Release the semaphore when the function exits

				//log.Printf("Worker %d releasing semaphore\n", id)
			}
		}

		if pass {
			message := <-messageQueue
			processMessage(id, message, svc, sqsURL, s3Bucket, awsRegion, s)
		}
		time.Sleep(1 * time.Second)
	}
}

func createAWSSession(region string) *session.Session {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	return sess
}

func downloadFileFromS3(s3Svc *s3.S3, bucket, path string) ([]byte, error) {
	output, err := s3Svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing S3 response body:", err)
		}
	}(output.Body)

	fileContent, err := ioutil.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}
	return fileContent, nil
}

func processMessage(id int, message *sqs.Message, svc *sqs.SQS, sqsURL, s3Bucket, awsRegion string, s *store.Store) {
	defer func() {
		totalRequests++
		log.Printf("Total requests: %d\n", totalRequests)
	}()
	var msgData map[string]interface{}
	if err := json.Unmarshal([]byte(*message.Body), &msgData); err != nil {
		log.Println("Error decoding JSON message:", err)
		return
	}

	// check if message has all the required fields if not return error
	if _, ok := msgData["s3Location"]; !ok {
		log.Println("Message missing s3Location field")
		return
	}

	if _, ok := msgData["user_id"]; !ok {
		log.Println("Message missing user_id field")
		return
	}

	if _, ok := msgData["screen_id"]; !ok {
		log.Println("Message missing screen_id field")
		return
	}

	path := msgData["s3Location"].(string)
	userIDTemp := msgData["user_id"].(string)
	userID, err := strconv.ParseInt(userIDTemp, 10, 64)
	screenIDTemp := msgData["screen_id"].(string)
	screenID, err := strconv.ParseInt(screenIDTemp, 10, 64)

	fmt.Printf("Worker %d received message. Path: %s. User ID: %s. Screen ID: %s\n", id, path, userID, screenIDTemp)

	sess := createAWSSession(awsRegion)
	s3Svc := s3.New(sess)

	fileContent, err := downloadFileFromS3(s3Svc, s3Bucket, path)
	if err != nil {
		log.Println("Error downloading file from S3:", err)
		log.Printf("Bucket: %s, Key: %s\n", s3Bucket, path)
		return
	}

	CrudeGrobidResponse, err := parsing.SendPDF2Grobid(fileContent)
	if err != nil {
		log.Println("Error sending file to Grobid service:", err)

		// if err contains connect: connection refused kill entire go app
		if strings.Contains(err.Error(), "connect: connection refused") || strings.Contains(err.Error(), "server misbehaving") || strings.Contains(err.Error(), "host not found") {
			log.Println("Grobid service is down, killing app...")
			os.Exit(1)
		}

		return
	}

	// clean up grobid response
	tidyGrobidResponse, err := parsing.TidyUpGrobidResponse(CrudeGrobidResponse)
	if err != nil {
		log.Println("Error tidying up Grobid response:", err)
		return
	}

	crossRefResponse := &parsing.TidyCrossRefResponse{}

	// Cross reference data using the DOI
	if tidyGrobidResponse.Doi != "" {
		crossRefResponse, err = parsing.CrossRefDataDOI(tidyGrobidResponse.Doi)
		if err != nil {
			log.Println("Error cross referencing data using DOI:", err)
		}
	}

	// If DOI is not available or failed, try cross-referencing using Title
	if crossRefResponse.DOI == "" && tidyGrobidResponse.Title != "" {
		crossRefResponse, err = parsing.CrossRefDataTitle(tidyGrobidResponse.Title)
		if err != nil {
			log.Println("Error cross referencing data using Title:", err)
		}
	}

	// create a PDFDTO
	pdfDTO := parsing.CreatePDFDTO(tidyGrobidResponse, crossRefResponse)

	if pdfDTO.DOI == "" {
		s.FindDOIFromPaperRepository(pdfDTO, screenID)
	}

	// ---- Paper ----
	var paper store.Paper

	// check if paper already exists
	paperAlreadyExists := false
	if pdfDTO.DOI != "" {
		log.Println("Finding paper by DOI...")
		paper, err = s.FindPaperByDOI(screenID, pdfDTO.DOI)
	} else if pdfDTO.Title != "" && pdfDTO.Abstract != "" {
		log.Println("Finding paper by title and abstract...")
		paper, err = s.FindPaperByTitleAndAbstract(screenID, pdfDTO.Title, pdfDTO.Abstract)
	} else if pdfDTO.Title != "" {
		log.Println("Finding paper by title...")
		paper, err = s.FindPaperByTitle(screenID, pdfDTO.Title)
	}

	if err != nil {
		log.Println("NON-FATAL: Couldn't find paper:", err)
	}

	// if paper does not exist, create it
	if paper.ID == 0 {
		paper, err = s.CreatePaper(pdfDTO, userID, screenID)
		if err != nil {
			log.Println("Error creating paper:", err)
			return
		} else {
			log.Printf("Created paper: %v\n", paper.ID)
		}
	} else {
		log.Printf("Found paper: %v\n", paper.ID)
		paperAlreadyExists = true
	}

	// ---- Sections ----
	// get sections and headings from $dto

	// initialise sections off by setting the first section to the abstract
	sections := []store.Section{
		{
			Header: "abstract",
			Text:   pdfDTO.Abstract,
		},
	}
	//// iterate through sections and add them to the sections array
	for _, section := range pdfDTO.Sections {
		if len(section.P) == 0 {
			continue
		}
		section.Head = strings.ToLower(section.Head)

		for _, p := range section.P {
			//log.Printf("Section header: %s\n Section text length: %d\n", section.Head, len(p))
			sections = append(sections, store.Section{
				Header: section.Head,
				Text:   p,
			})
		}
	}

	// if new section (by p), save it, else skip, give ascending order
	// the embeddings will be created later elsewhere when the user wants to screen the full text
	order := 0
	if paperAlreadyExists {
		orderTemp, _ := s.GetNextSectionOrder(paper.ID)
		order = int(orderTemp)
	}

	for _, section := range sections {
		//log.Printf("Section: %s\n", section.Header)
		//log.Printf("Text: %s\n", section.Text)
		_, err := s.CreateSection(paper.ID, section.Header, section.Text, order)
		if err != nil {
			logging.ErrorLogger.Println(err)
			// skip this section
			continue
		}
		order++
	}
	log.Printf("Sections iterated: %d\n", len(sections))

	if helpers.GetEnvVariable("REQUEUE_REQUESTS") == "true" {
		_, err = svc.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
			QueueUrl:          aws.String(sqsURL),
			ReceiptHandle:     message.ReceiptHandle,
			VisibilityTimeout: aws.Int64(30),
		})
		if err != nil {
			log.Println("Error putting message back to the queue:", err)
		}
	} else {
		_, err = svc.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(sqsURL),
			ReceiptHandle: message.ReceiptHandle,
		})
		if err != nil {
			log.Println("Error deleting message:", err)
		}

		// delete file from S3
		_, err = s3Svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(s3Bucket),
			Key:    aws.String(path),
		})
		if err != nil {
			log.Println("Error deleting file from S3:", err)
		}
	}

	log.Printf("Worker %d finished processing message\n", id)
}
