package config

import "time"

type AppConfig struct {
	Dispatcher                   DispatcherConfig `json:"dispatcher"`
	Queue                        QueueConfig      `json:"queue"`
	Worker                       WorkerConfig     `json:"worker"`
	MaxRequestsPerGrobidInstance int              `json:"max_requests_per_grobid_instance"`
	MinimumGapBetweenRequests    time.Duration    `json:"minimum_gap_between_requests"`
	RequestsQueue                string           `json:"requests_queue"`
	ResultsQueue                 string           `json:"results_queue"`
	S3Bucket                     string           `json:"s3_bucket"`
}

type DispatcherConfig struct {
	Count         int           `json:"count"`
	Interval      time.Duration `json:"interval"`
	WorkerCount   int           `json:"worker_count"`
	DispatchLimit int           `json:"dispatch_limit"`
}

type QueueConfig struct {
	QueueName            string `json:"queue_name"`
	PollingWaitTime      int64  `json:"polling_wait_time"`
	VisibilityTimeout    int64  `json:"visibility_timeout"`
	AckRetries           int    `json:"ack_retries"`
	MaxMessagesToProcess int    `json:"max_messages_to_process"`
}

type WorkerConfig struct {
	Count    int           `json:"count"`
	Interval time.Duration `json:"interval"`
}
