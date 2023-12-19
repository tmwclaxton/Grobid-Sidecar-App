package dispatcher

// Work represents a message to be processed by the worker
type Work struct {
	Path      string `json:"s3Location"`
	Operation string `json:"operation"`
}

// IsValid For example, a method to check if the work is valid
func (w *Work) IsValid() bool {
	return w.Path != "" && w.Operation != ""
}

// NewWork creates a new Work instance
func NewWork(path, operation string) *Work {
	return &Work{Path: path, Operation: operation}
}
