package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/natrontech/alertmanager-uptime-kuma-push/internal/util"
)

// These variables are set in build step
var Version = "v0.0.0-dev.0"
var Commit = "none"
var BuildTime = "unknown"

// pushBudget bounds all attempts of one webhook together. Alertmanager cancels a
// notification after max(group_interval, 10s) plus its cluster peer wait, so this fits any route.
var pushBudget = 9 * time.Second

// alert is the part of an alert in the Alertmanager webhook body that the pusher reads.
type alert struct {
	Status string            `json:"status"`
	Labels map[string]string `json:"labels"`
}

func newApp() *fiber.App {
	app := fiber.New()

	app.Post("/push", func(c *fiber.Ctx) error {

		var body struct {
			Alerts []alert `json:"alerts"`
		}
		if err := json.Unmarshal(c.Body(), &body); err != nil {
			log.Println("Ignoring webhook with invalid body:", err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		// Only a firing Watchdog keeps the monitor alive. Answer OK otherwise, there is nothing for Alertmanager to retry.
		if !slices.ContainsFunc(body.Alerts, func(a alert) bool {
			return a.Status == "firing" && a.Labels["alertname"] == "Watchdog"
		}) {
			log.Println("Webhook has no firing Watchdog alert, not pushing")
			return c.SendString("OK")
		}

		tr := &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   3 * time.Second,
			MaxIdleConns:      1,
		}

		// Each attempt gets an equal share of the budget
		attemptTime := pushBudget / time.Duration(util.Retries)

		// Set the timeout duration
		client := new(http.Client)
		client.Timeout = attemptTime
		client.Transport = tr

		// Loop over the number of retries
		for i := 0; i < util.Retries; i++ {

			// Log trying again if it's not the first try
			if i > 0 {
				log.Println("Try " + strconv.Itoa(i) + " of " + strconv.Itoa(util.Retries) + " failed, trying again")
			}

			start := time.Now()

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

			// Log if it was the last try
			if i == util.Retries-1 {
				log.Println("Try " + strconv.Itoa(i+1) + " of " + strconv.Itoa(util.Retries) + " failed")
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			// Wait out the rest of this attempt's share, so a quick failure does not use up all tries at once
			time.Sleep(attemptTime - time.Since(start))
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
