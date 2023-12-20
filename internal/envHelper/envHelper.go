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
		// This is because we want to also be able to pass environment variables via docker-compose or ecs task definitions
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
