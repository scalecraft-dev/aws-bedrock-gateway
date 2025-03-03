package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// ChatRequest represents the incoming chat request
type ChatRequest struct {
	Messages         []Message   `json:"messages" binding:"required"`
	Model            string      `json:"model" binding:"required"`
	Temperature      float32     `json:"temperature,omitempty"`
	TopP             float32     `json:"top_p,omitempty"`
	MaxTokens        int         `json:"max_tokens,omitempty"`
	Stop             []string    `json:"stop,omitempty"`
	Stream           bool        `json:"stream,omitempty"`
	N                int         `json:"n,omitempty"`
	PresencePenalty  float32     `json:"presence_penalty,omitempty"`
	FrequencyPenalty float32     `json:"frequency_penalty,omitempty"`
	User             string      `json:"user,omitempty"`
	Functions        []Function  `json:"functions,omitempty"`
	FunctionCall     interface{} `json:"function_call,omitempty"`
	ResponseFormat   *struct {
		Type string `json:"type,omitempty"`
	} `json:"response_format,omitempty"`
	Seed       int64       `json:"seed,omitempty"`
	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"`
}

// StreamOptions represents options for streaming responses
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Message represents a single message in the conversation
type Message struct {
	Role         string      `json:"role" binding:"required"`
	Content      interface{} `json:"content" binding:"required"`
	Name         string      `json:"name,omitempty"`
	FunctionCall interface{} `json:"function_call,omitempty"`
}

// TextContent represents text content in a message
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ImageContent represents image content in a message
type ImageContent struct {
	Type     string    `json:"type"`
	ImageURL *ImageURL `json:"image_url"`
}

// ImageURL represents an image URL
type ImageURL struct {
	URL string `json:"url"`
}

// Tool represents a tool that can be used by the model
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function that can be called by the model
type Function struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall represents a tool call made by the model
type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// ChatResponse represents the response from the Bedrock service
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int                 `json:"index"`
	Message      ChatResponseMessage `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

// ChatResponseMessage represents a message in the response
type ChatResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// BedrockService handles interactions with AWS Bedrock
type BedrockService struct {
	client *bedrockruntime.Client
}

// NewBedrockService creates a new instance of BedrockService
func NewBedrockService(region string) (*BedrockService, error) {
	// Load AWS configuration with specified region
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	// Create Bedrock client
	client := bedrockruntime.NewFromConfig(cfg)

	return &BedrockService{
		client: client,
	}, nil
}

// ProcessChat sends the chat request to AWS Bedrock and returns the response
func (s *BedrockService) ProcessChat(ctx context.Context, req ChatRequest) (string, error) {
	// Convert the chat request to the appropriate format for the model
	payload, err := formatPayloadForModel(req)
	if err != nil {
		return "", err
	}

	// Call Bedrock InvokeModel API
	resp, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(req.Model),
		ContentType: aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return "", err
	}

	// Parse the response based on the model
	return parseResponseFromModel(resp.Body)
}

// ProcessChatStream sends the chat request to AWS Bedrock and returns a stream of responses
func (s *BedrockService) ProcessChatStream(ctx context.Context, req ChatRequest) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error) {
	// Convert the chat request to the appropriate format for the model
	payload, err := formatPayloadForModel(req)
	if err != nil {
		return nil, err
	}

	// Call Bedrock InvokeModelWithResponseStream API
	resp, err := s.client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(req.Model),
		ContentType: aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// formatPayloadForModel formats the request payload based on the model
func formatPayloadForModel(req ChatRequest) ([]byte, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048 // Default max tokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7 // Default temperature
	}

	payload := map[string]interface{}{
		"messages":    req.Messages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"top_p":       req.TopP,
	}

	// Add anthropic_version for Claude models
	if strings.Contains(req.Model, "anthropic.claude") || strings.Contains(req.Model, ".anthropic.") {
		payload["anthropic_version"] = "bedrock-2023-05-31"
	}

	return json.Marshal(payload)
}

// parseResponseFromModel parses the response based on the model
func parseResponseFromModel(responseBody []byte) (string, error) {
	// Log the raw response for debugging
	log.Printf("Raw response: %s", string(responseBody))

	var response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(response.Content) > 0 {
		return response.Content[0].Text, nil
	}

	return "", errors.New("no content in response")
}

// GenerateMessageID generates a unique message ID
func GenerateMessageID() string {
	return fmt.Sprintf("chatcmpl-%s", time.Now().Format("20060102150405"))
}

// ParseImage tries to get the raw data from an image URL
func ParseImage(imageURL string) ([]byte, string, error) {
	pattern := `^data:(image/[a-z]*);base64,\s*`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(imageURL)

	// If already base64 encoded
	if len(matches) > 1 {
		contentType := matches[1]
		imageData := re.ReplaceAllString(imageURL, "")
		decoded, err := base64.StdEncoding.DecodeString(imageData)
		if err != nil {
			return nil, "", err
		}
		return decoded, contentType, nil
	}

	// Send a request to the image URL
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unable to access the image URL, status: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image") {
		contentType = "image/jpeg"
	}

	imageContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return imageContent, contentType, nil
}

// ConvertFinishReason converts Bedrock finish reasons to OpenAI format
func ConvertFinishReason(finishReason string) string {
	if finishReason == "" {
		return ""
	}

	finishReasonMapping := map[string]string{
		"tool_use":         "tool_calls",
		"finished":         "stop",
		"end_turn":         "stop",
		"max_tokens":       "length",
		"stop_sequence":    "stop",
		"complete":         "stop",
		"content_filtered": "content_filter",
	}

	if mapped, ok := finishReasonMapping[strings.ToLower(finishReason)]; ok {
		return mapped
	}

	return strings.ToLower(finishReason)
}

// ListBedrockModels lists available Bedrock models
func (s *BedrockService) ListBedrockModels(ctx context.Context) ([]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	bedrockClient := bedrock.NewFromConfig(cfg)
	var modelIDs []string

	// Get foundation models
	foundationResp, err := bedrockClient.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{
		ByOutputModality: types.ModelModalityText,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list foundation models: %v", err)
	}

	// Process foundation models
	for _, model := range foundationResp.ModelSummaries {
		if model.ModelLifecycle != nil &&
			model.ModelLifecycle.Status == "ACTIVE" &&
			*model.ResponseStreamingSupported {
			modelIDs = append(modelIDs, *model.ModelId)
		}
	}

	// Get inference profiles
	profileResp, err := bedrockClient.ListInferenceProfiles(ctx, &bedrock.ListInferenceProfilesInput{
		MaxResults: aws.Int32(1000),
		TypeEquals: types.InferenceProfileTypeSystemDefined,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list inference profiles: %v", err)
	}

	// Add inference profile models
	for _, profile := range profileResp.InferenceProfileSummaries {
		if profile.InferenceProfileId != nil {
			modelIDs = append(modelIDs, *profile.InferenceProfileId)
		}
	}

	return modelIDs, nil
}
