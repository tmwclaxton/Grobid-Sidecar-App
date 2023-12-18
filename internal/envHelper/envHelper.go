package envHelper

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		// Not fatal, just log the error and continue
		log.Println("Couldn't load .env file:", err)
	}
}

func GetEnvVariable(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s not set", key)
	}
	return value
}
