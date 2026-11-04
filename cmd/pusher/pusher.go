package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/natrontech/alertmanager-uptime-kuma-push/internal/util"
)

// These variables are set in build step
var Version = "v0.0.0-dev.0"
var Commit = "none"
var BuildTime = "unknown"

func newApp() *fiber.App {
	app := fiber.New()

	app.All("/push", func(c *fiber.Ctx) error {

		tr := &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   3 * time.Second,
			MaxIdleConns:      1,
		}

		// Set the timeout duration
		client := new(http.Client)
		client.Timeout = 3 * time.Second
		client.Transport = tr

		// Loop over the number of retries
		for i := 0; i < util.Retries; i++ {

			// Log trying again if it's not the first try
			if i > 0 {
				log.Println("Try " + strconv.Itoa(i) + " of " + strconv.Itoa(util.Retries) + " failed, trying again")
			}

			// Only log the host of the URL
			url, _ := url.Parse(util.UptimeKumaURL)
			log.Println("Sending HTTP GET request to", url.Host)

			resp, err := client.Get(util.UptimeKumaURL)
			if err != nil {
				log.Println("Error sending HTTP request to uptime kuma url", err)
			} else {
				_ = resp.Body.Close()

				// Exit the loop if the push was accepted
				if resp.StatusCode/100 == 2 {
					break
				}
				log.Println("Uptime Kuma answered with status", resp.StatusCode)
			}

			// sleep 3 seconds
			time.Sleep(3 * time.Second)

			// Log if it was the last try
			if i == util.Retries-1 {
				log.Println("Try " + strconv.Itoa(i+1) + " of " + strconv.Itoa(util.Retries) + " failed")
				return c.SendStatus(fiber.StatusInternalServerError)
			}

		}
		return c.SendString("OK")
	})

	return app
}

func main() {

	// Log with timestamp
	log.SetFlags(log.LstdFlags)

	log.Println("Starting pusher", Version, "commit", Commit, "build time", BuildTime)

	if err := util.LoadEnv(); err != nil {
		log.Println("Error loading environment variables:", err)
		os.Exit(1)
	}

	err := newApp().Listen(":8081")
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}
