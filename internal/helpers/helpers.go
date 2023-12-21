package helpers

import (
	"crypto/rand"
	"encoding/base64"
	"log"
)

func GenerateRandomString(length int) string {
	// Calculate the number of bytes needed to represent the string
	numBytes := (length * 6) / 8
	if (length*6)%8 != 0 {
		numBytes++
	}

	// Generate random bytes
	randomBytes := make([]byte, numBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Println("Error generating random string:", err)
		return ""
	}

	// Encode random bytes to base64
	randomString := base64.URLEncoding.EncodeToString(randomBytes)

	// Trim extra padding characters
	randomString = randomString[:length]

	return randomString
}
