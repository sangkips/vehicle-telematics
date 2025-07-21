package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port 			string
	MongoURI 		string
	JWTSecret      	string
	JWTExpiry      	string
	AllowedOrigins 	[]string
	UpdateInterval 	string
}

func Load() *Config {
	// load .env variable
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
    if allowedOrigins == "" {
        allowedOrigins = "http://localhost:5173, https://telematics-pearl.vercel.app"
    }

    return &Config{
        Port:           port,
        MongoURI:       mongoURI,
        JWTSecret:      os.Getenv("JWT_SECRET"),
        JWTExpiry:      os.Getenv("JWT_EXPIRY"),
        AllowedOrigins: strings.Split(allowedOrigins, ","),
        UpdateInterval: os.Getenv("UPDATE_INTERVAL"),
    }
}
