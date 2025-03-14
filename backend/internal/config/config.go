package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config represents the application configuration
type Config struct {
	S3     S3Config     `json:"s3"`
	Redis  RedisConfig  `json:"redis"`
	Server ServerConfig `json:"server"`
}

// S3Config represents the S3 configuration
type S3Config struct {
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Endpoint        string `json:"endpoint,omitempty"`
}

// RedisConfig represents the Redis configuration
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password,omitempty"`
	DB       int    `json:"db,omitempty"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

// LoadConfig loads the configuration from a file and overrides with environment variables
func LoadConfig(path string) (*Config, error) {
	// Default configuration
	config := Config{
		S3: S3Config{
			Region: "us-east-1",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		Server: ServerConfig{
			Port: 8080,
			Host: "0.0.0.0",
		},
	}

	// Load from file if it exists
	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		err = decoder.Decode(&config)
		if err != nil {
			return nil, fmt.Errorf("failed to decode config file: %w", err)
		}
	} else {
		fmt.Printf("Config file %s not found, using environment variables and defaults\n", path)
	}

	// Override with environment variables
	if region := os.Getenv("S3_REGION"); region != "" {
		config.S3.Region = region
	}

	if bucket := os.Getenv("S3_BUCKET"); bucket != "" {
		config.S3.Bucket = bucket
	}

	if accessKeyID := os.Getenv("S3_ACCESS_KEY_ID"); accessKeyID != "" {
		config.S3.AccessKeyID = accessKeyID
	}

	if secretAccessKey := os.Getenv("S3_SECRET_ACCESS_KEY"); secretAccessKey != "" {
		config.S3.SecretAccessKey = secretAccessKey
	}

	if endpoint := os.Getenv("S3_ENDPOINT"); endpoint != "" {
		config.S3.Endpoint = endpoint
	}

	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.Redis.Host = redisHost
	}

	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		if port, err := strconv.Atoi(redisPort); err == nil {
			config.Redis.Port = port
		}
	}

	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		config.Redis.Password = redisPassword
	}

	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			config.Redis.DB = db
		}
	}

	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		if port, err := strconv.Atoi(serverPort); err == nil {
			config.Server.Port = port
		}
	}

	if serverHost := os.Getenv("SERVER_HOST"); serverHost != "" {
		config.Server.Host = serverHost
	}

	// Validate required configuration
	if config.S3.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket name is required")
	}

	return &config, nil
}

// Address returns the Redis address
func (r *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}
