package main

import (
	"app/rtc/db"
	"app/rtc/routers"
	"app/rtc/services"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, using system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	// Initialize database
	log.Println("Initializing database...")
	db.InitDB()
	log.Println("Database initialized successfully")

	// Setup routes
	log.Println("Setting up routes...")
	router := routers.SetupRoutes()

	// Create rate limiter (60 requests per minute, burst of 10)
	rateLimiter := services.NewRateLimiter(60, 10)

	// Apply services stack
	handler := services.Recovery(
		services.RequestLogger(
			services.SecurityHeaders(
				services.CORS(
					rateLimiter.Limit(
						services.UserContext(
							services.ProjectContext(
								router,
							),
						),
					),
				),
			),
		),
	)

	// Configure CORS with allowed origins
	allowedOrigins := []string{"*"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = []string{origins}
	}

	corsHandler := handlers.CORS(
		handlers.AllowedOrigins(allowedOrigins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "X-User-ID", "X-Project-ID", "X-Session-ID"}),
		handlers.ExposedHeaders([]string{"X-Session-ID"}),
		handlers.AllowCredentials(),
	)(handler)

	// Start server
	log.Printf("===========================================")
	log.Printf("RTC server starting on port %s", port)
	log.Printf("===========================================")
	log.Printf("API base URL: http://localhost:%s", port)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("GitHub OAuth: http://localhost:%s/github/login", port)
	log.Printf("Database: Connected")
	log.Printf("===========================================")

	if err := http.ListenAndServe(":"+port, corsHandler); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
