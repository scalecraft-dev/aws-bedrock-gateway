package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// EmbeddingsRequest represents a request for embeddings
type EmbeddingsRequest struct {
	Model           string      `json:"model" binding:"required"`
	Input           interface{} `json:"input" binding:"required"`
	EncodingFormat  string      `json:"encoding_format,omitempty"`
	EmbeddingConfig interface{} `json:"embedding_config,omitempty"`
}

// EmbeddingsResponse represents a response from the embeddings service
type EmbeddingsResponse struct {
	Object string          `json:"object"`
	Data   []Embedding     `json:"data"`
	Model  string          `json:"model"`
	Usage  EmbeddingsUsage `json:"usage"`
}

// Embedding represents a single embedding
type Embedding struct {
	Object    string      `json:"object"`
	Embedding interface{} `json:"embedding"`
	Index     int         `json:"index"`
}

// EmbeddingsUsage represents token usage information for embeddings
type EmbeddingsUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// SupportedEmbeddingModels is a map of supported embedding models
var SupportedEmbeddingModels = map[string]string{
	"cohere.embed-multilingual-v3": "Cohere Embed Multilingual",
	"cohere.embed-english-v3":      "Cohere Embed English",
}

// ProcessEmbeddings processes an embeddings request
func (s *BedrockService) ProcessEmbeddings(ctx context.Context, req EmbeddingsRequest) (*EmbeddingsResponse, error) {
	// Check if model is supported
	modelName, ok := SupportedEmbeddingModels[req.Model]
	if !ok {
		return nil, errors.New("unsupported embedding model")
	}

	// Format the request based on the model
	var payload []byte
	var err error

	switch modelName {
	case "Cohere Embed Multilingual", "Cohere Embed English":
		payload, err = formatCohereEmbeddingPayload(req)
	default:
		return nil, errors.New("unsupported embedding model")
	}

	if err != nil {
		return nil, err
	}

	// Call Bedrock InvokeModel API
	resp, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(req.Model),
		ContentType: aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return nil, err
	}

	// Parse the response
	return parseEmbeddingResponse(req.Model, resp.Body, req.EncodingFormat)
}

// formatCohereEmbeddingPayload formats the request for Cohere embedding models
func formatCohereEmbeddingPayload(req EmbeddingsRequest) ([]byte, error) {
	var texts []string

	switch v := req.Input.(type) {
	case string:
		texts = []string{v}
	case []string:
		texts = v
	case []interface{}:
		for _, item := range v {
			if text, ok := item.(string); ok {
				texts = append(texts, text)
			}
		}
	default:
		return nil, errors.New("unsupported input format for embeddings")
	}

	payload := map[string]interface{}{
		"texts":      texts,
		"input_type": "search_document",
		"truncate":   "END",
	}

	return json.Marshal(payload)
}

// parseEmbeddingResponse parses the embedding response
func parseEmbeddingResponse(model string, responseBody []byte, encodingFormat string) (*EmbeddingsResponse, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}

	// Extract embeddings based on model
	var embeddings []interface{}
	var promptTokens int

	if strings.HasPrefix(model, "cohere.embed") {
		if embeds, ok := response["embeddings"].([]interface{}); ok {
			embeddings = embeds
		}
	}

	// Create response
	embeddingResponse := &EmbeddingsResponse{
		Object: "list",
		Model:  model,
		Data:   make([]Embedding, len(embeddings)),
		Usage: EmbeddingsUsage{
			PromptTokens: promptTokens,
			TotalTokens:  promptTokens,
		},
	}

	// Format embeddings based on encoding format
	for i, embed := range embeddings {
		embeddingResponse.Data[i] = Embedding{
			Object: "embedding",
			Index:  i,
		}

		if encodingFormat == "base64" {
			// Convert to base64
			jsonData, _ := json.Marshal(embed)
			embeddingResponse.Data[i].Embedding = base64.StdEncoding.EncodeToString(jsonData)
		} else {
			embeddingResponse.Data[i].Embedding = embed
		}
	}

	return embeddingResponse, nil
}
