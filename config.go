package main

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	// API configuration
	DefaultAPIKeys string
	APIRoutePrefix string
	Title          string
	Summary        string
	Version        string
	Description    string

	// Debug and AWS configuration
	Debug                      bool
	AWSRegion                  string
	DefaultModel               string
	DefaultEmbeddingModel      string
	EnableCrossRegionInference bool
}

// NewConfig creates a new configuration with values from environment variables
func NewConfig() *Config {
	return &Config{
		DefaultAPIKeys: getEnv("DEFAULT_API_KEYS", "bedrock"),
		APIRoutePrefix: getEnv("API_ROUTE_PREFIX", "/api/v1"),

		Title:       "Amazon Bedrock Proxy APIs",
		Summary:     "OpenAI-Compatible RESTful APIs for Amazon Bedrock",
		Version:     "0.1.0",
		Description: "Use OpenAI-Compatible RESTful APIs for Amazon Bedrock models.",

		Debug:                      getEnv("DEBUG", false),
		AWSRegion:                  getEnv("AWS_REGION", "us-east-1"),
		DefaultModel:               getEnv("DEFAULT_MODEL", "anthropic.claude-3-sonnet-20240229-v1:0"),
		DefaultEmbeddingModel:      getEnv("DEFAULT_EMBEDDING_MODEL", "cohere.embed-multilingual-v3"),
		EnableCrossRegionInference: getEnv("ENABLE_CROSS_REGION_INFERENCE", false),
	}
}

// getEnv is a generic function that gets an environment variable with a default value
// It supports string, bool, int, float64 types
func getEnv[T string | bool | int | float64](key string, defaultValue T) T {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	var result T
	switch any(defaultValue).(type) {
	case string:
		// For string type, just return the value
		return any(value).(T)
	case bool:
		// For bool type, parse the value
		lowerValue := strings.ToLower(value)
		result = any(!(lowerValue == "false" || lowerValue == "0" || lowerValue == "no" || lowerValue == "n" || lowerValue == "off")).(T)
	case int:
		// For int type, parse the value
		if intValue, err := strconv.Atoi(value); err == nil {
			result = any(intValue).(T)
		} else {
			result = defaultValue
		}
	case float64:
		// For float64 type, parse the value
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			result = any(floatValue).(T)
		} else {
			result = defaultValue
		}
	default:
		// For unsupported types, return the default value
		result = defaultValue
	}

	return result
}
