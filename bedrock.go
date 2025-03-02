package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// ChatRequest represents the incoming chat request
type ChatRequest struct {
	Messages            []Message     `json:"messages" binding:"required"`
	Model               string        `json:"model" binding:"required"`
	Temperature         float32       `json:"temperature,omitempty"`
	TopP                float32       `json:"top_p,omitempty"`
	MaxTokens           int           `json:"max_tokens,omitempty"`
	Stop                []string      `json:"stop,omitempty"`
	Tools               []Tool        `json:"tools,omitempty"`
	ToolChoice          interface{}   `json:"tool_choice,omitempty"`
	StreamOptions       StreamOptions `json:"stream_options,omitempty"`
	ReasoningEffort     string        `json:"reasoning_effort,omitempty"`
	MaxCompletionTokens int           `json:"max_completion_tokens,omitempty"`
}

// StreamOptions represents options for streaming responses
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Message represents a single message in the conversation
type Message struct {
	Role       string      `json:"role" binding:"required"`
	Content    interface{} `json:"content" binding:"required"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
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
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int                 `json:"index"`
	Message      ChatResponseMessage `json:"message"`
	FinishReason string              `json:"finish_reason"`
	Logprobs     interface{}         `json:"logprobs"`
}

// ChatResponseMessage represents a message in the response
type ChatResponseMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
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
	return parseResponseFromModel(req.Model, resp.Body)
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
	// Implementation depends on the specific models you want to support

	// For Claude models
	if strings.HasPrefix(req.Model, "anthropic.claude") {
		return formatClaudePayload(req)
	}

	// For Llama models
	if strings.HasPrefix(req.Model, "meta.llama") {
		return formatLlamaPayload(req)
	}

	// Add more model formats as needed

	return nil, errors.New("unsupported model")
}

// formatClaudePayload formats the request for Claude models
func formatClaudePayload(req ChatRequest) ([]byte, error) {
	// Build the conversation history
	var prompt strings.Builder

	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			prompt.WriteString("Human: ")
			if content, ok := msg.Content.(string); ok {
				prompt.WriteString(content)
			} else if contentList, ok := msg.Content.([]interface{}); ok {
				// Handle multimodal content
				for _, part := range contentList {
					if partMap, ok := part.(map[string]interface{}); ok {
						if partMap["type"] == "text" {
							prompt.WriteString(partMap["text"].(string))
						}
						// Handle image content if needed
					}
				}
			}
			prompt.WriteString("\n\n")
		case "assistant":
			prompt.WriteString("Assistant: ")
			if content, ok := msg.Content.(string); ok {
				prompt.WriteString(content)
			}
			prompt.WriteString("\n\n")
		case "system":
			// System messages are handled differently in Claude
			// They are typically added at the beginning
			continue
		}
	}

	prompt.WriteString("Assistant: ")

	// Create the Claude payload
	claudePayload := map[string]interface{}{
		"prompt":               prompt.String(),
		"max_tokens_to_sample": req.MaxTokens,
		"temperature":          req.Temperature,
		"top_p":                req.TopP,
	}

	if len(req.Stop) > 0 {
		claudePayload["stop_sequences"] = req.Stop
	}

	// Add system prompt if present
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				claudePayload["system"] = content
				break
			}
		}
	}

	return json.Marshal(claudePayload)
}

// formatLlamaPayload formats the request for Llama models
func formatLlamaPayload(req ChatRequest) ([]byte, error) {
	// Build the conversation history
	var messages []map[string]interface{}

	for _, msg := range req.Messages {
		message := map[string]interface{}{
			"role": msg.Role,
		}

		if content, ok := msg.Content.(string); ok {
			message["content"] = content
		} else if contentList, ok := msg.Content.([]interface{}); ok {
			// Handle multimodal content
			var contentParts []map[string]interface{}
			for _, part := range contentList {
				if partMap, ok := part.(map[string]interface{}); ok {
					contentParts = append(contentParts, partMap)
				}
			}
			message["content"] = contentParts
		}

		messages = append(messages, message)
	}

	// Create the Llama payload
	llamaPayload := map[string]interface{}{
		"messages":    messages,
		"max_gen_len": req.MaxTokens,
		"temperature": req.Temperature,
		"top_p":       req.TopP,
	}

	if len(req.Stop) > 0 {
		llamaPayload["stop"] = req.Stop
	}

	return json.Marshal(llamaPayload)
}

// parseResponseFromModel parses the response based on the model
func parseResponseFromModel(model string, responseBody []byte) (string, error) {
	// Parse based on model
	if strings.HasPrefix(model, "anthropic.claude") {
		return parseClaudeResponse(responseBody)
	} else if strings.HasPrefix(model, "meta.llama") {
		return parseLlamaResponse(responseBody)
	}

	// Default parsing
	return string(responseBody), nil
}

// parseClaudeResponse parses the response from Claude models
func parseClaudeResponse(responseBody []byte) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", err
	}

	if completion, ok := response["completion"].(string); ok {
		return completion, nil
	}

	return string(responseBody), nil
}

// parseLlamaResponse parses the response from Llama models
func parseLlamaResponse(responseBody []byte) (string, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", err
	}

	if generation, ok := response["generation"].(string); ok {
		return generation, nil
	}

	return string(responseBody), nil
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

// ListBedrockModels returns a list of available Bedrock models
func (s *BedrockService) ListBedrockModels(ctx context.Context) ([]string, error) {
	// This is a simplified implementation
	// In a real implementation, you would call the Bedrock API to get the list of models
	return []string{
		"anthropic.claude-v2",
		"anthropic.claude-3-sonnet-20240229-v1:0",
		"meta.llama2-13b-chat-v1",
	}, nil
}
