package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port 			string
	MongoURI 		string
	JWTSecret      	string
	JWTExpiry      	string
	AllowedOrigins 	[]string
	Redis          	RedisConfig
	RateLimit      	RateLimitConfig
}

type RedisConfig struct {
	Host               string
	Port               string
	Password           string
	DB                 int
	PoolSize           int
	MinIdleConns       int
	MaxRetries         int
	RetryDelay         time.Duration
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	PoolTimeout        time.Duration
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
	URL      			string
}

type RateLimitConfig struct {
	Enabled         bool          `json:"enabled"`
	RedisKeyPrefix  string        `json:"redisKeyPrefix"`
	CleanupInterval time.Duration `json:"cleanupInterval"`
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
        allowedOrigins = "http://localhost:5173, https://telematics-pearl.vercel.app, https://vehicle-telematics-sangkips9264-cnqbn1gd.leapcell.dev"
    }

    return &Config{
        Port:           port,
        MongoURI:       mongoURI,
        JWTSecret:      os.Getenv("JWT_SECRET"),
        JWTExpiry:      os.Getenv("JWT_EXPIRY"),
        AllowedOrigins: strings.Split(allowedOrigins, ","),
        Redis:          loadRedisConfig(),
        RateLimit:      loadRateLimitConfig(),
    }
}
func loadRedisConfig() RedisConfig {
	// Helper function to parse duration with default
	parseDuration := func(envVar string, defaultValue time.Duration) time.Duration {
		if val := os.Getenv(envVar); val != "" {
			if duration, err := time.ParseDuration(val); err == nil {
				return duration
			}
		}
		return defaultValue
	}

	// Helper function to parse int with default
	parseInt := func(envVar string, defaultValue int) int {
		if val := os.Getenv(envVar); val != "" {
			if intVal, err := strconv.Atoi(val); err == nil {
				return intVal
			}
		}
		return defaultValue
	}

	redisHost := os.Getenv("REDIS_URL")
	if redisHost == "" {
		redisHost = "localhost"
	}
	

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	return RedisConfig{
		Host:               redisHost,
		Port:               redisPort,
		Password:           os.Getenv("REDIS_PASSWORD"),
		DB:                 parseInt("REDIS_DB", 0),
		PoolSize:           parseInt("REDIS_POOL_SIZE", 10),
		MinIdleConns:       parseInt("REDIS_MIN_IDLE_CONNS", 5),
		MaxRetries:         parseInt("REDIS_MAX_RETRIES", 3),
		RetryDelay:         parseDuration("REDIS_RETRY_DELAY", 1*time.Second),
		DialTimeout:        parseDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:        parseDuration("REDIS_READ_TIMEOUT", 3*time.Second),
		WriteTimeout:       parseDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		PoolTimeout:        parseDuration("REDIS_POOL_TIMEOUT", 4*time.Second),
		IdleTimeout:        parseDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
		IdleCheckFrequency: parseDuration("REDIS_IDLE_CHECK_FREQUENCY", 1*time.Minute),
	}
}

func loadRateLimitConfig() RateLimitConfig {
	// Helper function to parse duration with default
	parseDuration := func(envVar string, defaultValue time.Duration) time.Duration {
		if val := os.Getenv(envVar); val != "" {
			if duration, err := time.ParseDuration(val); err == nil {
				return duration
			}
		}
		return defaultValue
	}

	// Helper function to parse bool with default
	parseBool := func(envVar string, defaultValue bool) bool {
		if val := os.Getenv(envVar); val != "" {
			if boolVal, err := strconv.ParseBool(val); err == nil {
				return boolVal
			}
		}
		return defaultValue
	}

	keyPrefix := os.Getenv("RATE_LIMIT_KEY_PREFIX")
	if keyPrefix == "" {
		keyPrefix = "ratelimit:"
	}

	return RateLimitConfig{
		Enabled:         parseBool("RATE_LIMIT_ENABLED", true),
		RedisKeyPrefix:  keyPrefix,
		CleanupInterval: parseDuration("RATE_LIMIT_CLEANUP_INTERVAL", 5*time.Minute),
	}
}