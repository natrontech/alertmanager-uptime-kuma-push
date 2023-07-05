package util

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
)

var (
	UptimeKumaURL string
	Retries       int
)

// LoadEnv loads OS environment variables
func LoadEnv() error {
	// Load environment variables
	if UptimeKumaURL = os.Getenv("UPTIME_KUMA_URL"); UptimeKumaURL == "" {
		return errors.New("UPTIME_KUMA_URL not set")
	}

	// Get the value of GET_RETRIES environment variable
	getRetriesStr := os.Getenv("GET_RETRIES")

	// Check if GET_RETRIES is not set, set it to 3
	if getRetriesStr == "" {
		Retries = 3
		log.Println("GET_RETRIES not set, defaulting to 3")
	} else {
		// Convert the GET_RETRIES value to an int
		getRetries, err := strconv.Atoi(getRetriesStr)
		if err != nil {
			return fmt.Errorf("error converting GET_RETRIES to int: %w", err)
		}
		Retries = getRetries
	}

	// Validate uptime kuma url
	uri, err := url.ParseRequestURI(UptimeKumaURL)
	if err != nil {
		return errors.New("UPTIME_KUMA_URL is not a valid URL")
	}

	switch uri.Scheme {
	case "http":
	case "https":
	default:
		return errors.New("UPTIME_KUMA_URL has invalid scheme (http or https only)")
	}

	return nil
}
