package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port 			string
	MongoURI 		string
	JWTSecret      	string
	JWTExpiry      	string
	AllowedOrigins 	string
	UpdateInterval 	string
}

func Load() *Config {
	// load .env variable
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err) // Use Fatal to catch this early
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000" // Default for development
	}

	return &Config{
		Port:            os.Getenv("PORT"),
		MongoURI:        os.Getenv("MONGO_URI"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		JWTExpiry:       os.Getenv("JWT_EXPIRY"),
		AllowedOrigins:  os.Getenv("ALLOWED_ORIGINS"),
		UpdateInterval:  os.Getenv("UPDATE_INTERVAL"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
