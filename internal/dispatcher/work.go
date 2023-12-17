package dispatcher

type Work struct {
	S3Location  string `json:"s3_location"`
	Operation   string `json:"operation"`
	Destination string `json:"destination"`
}

// NewWork creates a new Work object
func newWork(s3Location, operation, destination string) Work {
	return Work{
		S3Location:  s3Location,
		Operation:   operation,
		Destination: destination,
	}
}
