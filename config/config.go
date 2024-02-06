package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// SetupConfig initializes the application configuration.
//
// It loads environment variables from a .env file if the debug
// parameter is set to true.
// No return value.
func SetupConfig(debug bool) {
	if debug {
		err := godotenv.Load(".env")
		if err != nil {
			log.Printf("Error loading .env file")
		}
	}

}

// GetConfig retrieves the value of the environment variable
// associated with the given key.
//
// It accepts a string `key` which is the name of the environment
// variable.
// It returns a string containing the value of the environment
// variable.
func GetConfig(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Environment variable %s not set", key)
	}
	return val
}
