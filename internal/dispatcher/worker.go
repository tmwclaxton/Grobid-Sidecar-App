package dispatcher

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"simple-go-app/internal/helpers"
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
)

var (
	lastRequestTime   time.Time
	lastRequestTimeMu sync.Mutex
)

func Worker(id int, messageQueue <-chan *sqs.Message, svc *sqs.SQS, sqsURL, s3Bucket string, s *store.Store) {
	awsRegion := helpers.GetEnvVariable("AWS_REGION")
	minGapBetweenRequests := helpers.GetEnvVariable("MINIMUM_GAP_BETWEEN_REQUESTS_SECONDS")
	minGap, err := time.ParseDuration(minGapBetweenRequests + "s")
	if err != nil {
		log.Fatalf("Error parsing MINIMUM_GAP_BETWEEN_REQUESTS_SECONDS: %v", err)
	}

	log.Printf("Starting worker %d...\n", id)

	for {
		message := <-messageQueue
		processMessage(id, message, svc, sqsURL, s3Bucket, awsRegion, minGap, s)
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

func processMessage(id int, message *sqs.Message, svc *sqs.SQS, sqsURL, s3Bucket, awsRegion string, minGap time.Duration, s *store.Store) {
	var msgData map[string]interface{}
	if err := json.Unmarshal([]byte(*message.Body), &msgData); err != nil {
		log.Println("Error decoding JSON message:", err)
		return
	}

	path := msgData["s3Location"].(string)
	userID := msgData["user_id"].(string)
	screenIDTemp := msgData["screen_id"].(string)
	screenID, err := strconv.ParseInt(screenIDTemp, 10, 64)

	fmt.Printf("Worker %d received message. Path: %s. User ID: %s. Screen ID: %s\n", id, path, userID, screenID)

	lastRequestTimeMu.Lock()
	timeSinceLastRequest := time.Since(lastRequestTime)
	lastRequestTimeMu.Unlock()

	if timeSinceLastRequest < minGap {
		sleepTime := minGap - timeSinceLastRequest
		log.Printf("Worker %d sleeping for %v to meet the minimum gap between requests\n", id, sleepTime)
		time.Sleep(sleepTime)
	}

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
		return
	}

	// clean up grobid response
	tidyGrobidResponse, err := parsing.TidyUpGrobidResponse(CrudeGrobidResponse)
	if err != nil {
		log.Println("Error tidying up Grobid response:", err)
		return
	}

	// cross reference data using the DOI
	crossRefResponse, err := parsing.CrossReferenceData(tidyGrobidResponse.Doi)
	if err != nil {
		log.Println("Error cross referencing data:", err)
		return
	}

	// create a PDFDTO
	pdfDTO := parsing.CreatePDFDTO(tidyGrobidResponse, crossRefResponse)

	//log.Printf("Title: %s\n", pdfDTO.Title)
	//log.Printf("DOI: %s\n", pdfDTO.DOI)
	//log.Printf("Date: %s\n", pdfDTO.Date)
	//log.Printf("Year: %s\n", pdfDTO.Year)
	//log.Printf("Abstract: %s\n", pdfDTO.Abstract)
	//log.Printf("Keywords: %v\n", pdfDTO.Keywords)
	//log.Printf("Sections: %v\n", pdfDTO.Sections)
	//log.Printf("Authors: %v\n", pdfDTO.Authors)
	//log.Printf("Journal: %s\n", pdfDTO.Journal)
	//log.Printf("Notes: %s\n", pdfDTO.Notes)

	if pdfDTO.DOI == "" {
		s.FindDOIFromPaperRepository(pdfDTO, screenID)
	}

	log.Printf("DOI: %s\n", pdfDTO.DOI)

	paper := &store.Paper{}
	paperAlreadyExists := false
	if pdfDTO.DOI != "" {
		paper, _ = s.FindPaperByDOI(screenID, pdfDTO.DOI)
	} else if pdfDTO.Title != "" && pdfDTO.Abstract != "" {
		paper = s.FindPaperByTitleAndAbstract(screenID, pdfDTO.Title, pdfDTO.Abstract)
	} else if pdfDTO.Title != "" {
		paper = s.FindPaperByTitle(screenID, pdfDTO.Title)
	} else {
		paper = nil
	}

	//log.Printf("Paper: %v\n", paper.ID)
	//log.Printf("Paper exists: %v\n", paper == nil)

	if paper.ID == 0 {
		log.Println("Creating paper pt 1...")
		paper, err = s.CreatePaper(pdfDTO, userID, screenID)
		if err != nil {
			log.Println("Error creating paper:", err)
		}
	} else {
		paperAlreadyExists = true
	}

	// ---- Sections ----
	// get sections and headings from $dto

	// initialise sections off by setting the first section to the abstract
	sections := []store.Section{
		{
			Header: "Abstract",
			Text:   pdfDTO.Abstract,
		},
	}
	// iterate through sections and add them to the sections array
	for _, section := range pdfDTO.Sections {
		if len(section.P) == 0 {
			continue
		}
		section.Head = strings.ToLower(section.Head)

		for _, p := range section.P {
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
		order = int(s.GetNextSectionOrder(paper.ID))
	}

	for _, section := range sections {
		_, err := s.CreateSection(paper.ID, section.Header, section.Text, order)
		if err != nil {
			return
		}
		order++
	}

	lastRequestTimeMu.Lock()
	lastRequestTime = time.Now()
	lastRequestTimeMu.Unlock()

	if helpers.GetEnvVariable("ENVIRONMENT") != "production" {
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
	}
}
