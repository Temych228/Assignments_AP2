package main

import (
	"log"

	"notification-service/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to init notification service: %v", err)
	}
	defer a.Close()

	if err := a.Run(); err != nil {
		log.Fatalf("notification service stopped: %v", err)
	}
}
