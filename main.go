package main

import (
	"log"
	"shorturl/db"
	"shorturl/handlers"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	// Initialize database
	db.InitializeDB()
	defer db.CloseDB()

	// Group routes under /shorturl
	shorturl := app.Group("/shorturl")
	shorturl.Post("/generate", handlers.GenerateShortURL)
	shorturl.Get("/*", handlers.RedirectShortURL)
	shorturl.Post("/report", handlers.GetClickReport)

	// Start server
	log.Println("Server running")
	if err := app.Listen(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
