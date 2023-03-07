package main

import (
    "os"
    "net/http"
    "log"

    "github.com/gofiber/fiber/v2"
    "github.com/natrontech/alertmanager-uptime-kuma-push/internal/util"

)

func init() {

    // Log with timestamp
    log.SetFlags(log.LstdFlags)

    if err := util.LoadEnv(); err != nil {
		log.Println("Error loading environment variables:", err)
		os.Exit(1)
	}
  }

func main() {
    app := fiber.New()

    app.All("/push", func(c *fiber.Ctx) error {

        log.Println("Sending HTTP Get request to Uptime Kuma URL")
        _ , err := http.Get(util.UptimeKumaURL)
        if err != nil {
            log.Println("Error sending HTTP request to uptime kuma url", err)
        }
        return c.SendString("OK")
    })

    app.Listen(":8081")
}