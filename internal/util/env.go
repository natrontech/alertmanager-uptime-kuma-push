package util

import (
	"errors"
	"os"
	"net/url"
)

var (
	UptimeKumaURL      string
)

// LoadEnv loads OS environment variables
func LoadEnv() error {
	// Load environment variables
	if UptimeKumaURL = os.Getenv("UPTIME_KUMA_URL"); UptimeKumaURL == "" {
		return errors.New("UPTIME_KUMA_URL not set")
	}

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