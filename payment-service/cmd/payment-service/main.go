package main

import (
	"log"

	"payment-service/internal/app"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("failed to load .env: %v", err)
	}

	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	defer a.Close()

	log.Println("Payment Service running...")
	if err := a.Run(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
