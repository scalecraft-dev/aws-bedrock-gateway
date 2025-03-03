package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Global configuration instance
var AppConfig *Config

func init() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist, we'll use environment variables
		log.Println("No .env file found, using environment variables")
	} else {
		log.Println("Loaded configuration from .env file")
	}

	// Initialize configuration
	AppConfig = NewConfig()
}

func main() {
	// Set Gin mode based on debug setting
	if AppConfig.Debug {
		gin.SetMode(gin.DebugMode)
		log.Println("Running in debug mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create a new Gin router
	r := gin.Default()

	// Create Bedrock service with region from config
	bedrockService, err := NewBedrockService(AppConfig.AWSRegion)
	if err != nil {
		log.Fatalf("Failed to create Bedrock service: %v", err)
	}

	// Setup routes with API prefix from config
	apiGroup := r.Group(AppConfig.APIRoutePrefix)
	SetupRoutes(apiGroup, bedrockService)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	// Log startup information
	log.Printf("Starting %s v%s", AppConfig.Title, AppConfig.Version)
	log.Printf("Listening on port %s", port)
	log.Printf("Using AWS Region: %s", AppConfig.AWSRegion)
	log.Printf("Default model: %s", AppConfig.DefaultModel)

	// Start the server
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
