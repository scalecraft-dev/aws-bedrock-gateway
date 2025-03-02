package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for the application
func SetupRoutes(r gin.IRouter, bedrockService *BedrockService) {
	// Chat endpoint
	r.POST("/chat", handleChat(bedrockService))

	// Stream chat endpoint
	r.POST("/chat/stream", handleChatStream(bedrockService))

	// List models endpoint
	r.GET("/models", handleListModels(bedrockService))

	// Embeddings endpoint
	r.POST("/embeddings", handleEmbeddings(bedrockService))
}

// handleChat handles the chat completion endpoint
func handleChat(bedrockService *BedrockService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var chatReq ChatRequest
		if err := c.ShouldBindJSON(&chatReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		response, err := bedrockService.ProcessChat(c.Request.Context(), chatReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ChatResponse{
			ID:                GenerateMessageID(),
			Object:            "chat.completion",
			Created:           int64(0), // Set to current timestamp in production
			Model:             chatReq.Model,
			SystemFingerprint: "fp",
			Choices: []Choice{
				{
					Index: 0,
					Message: ChatResponseMessage{
						Role:    "assistant",
						Content: response,
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     0, // Set to actual token counts in production
				CompletionTokens: 0,
				TotalTokens:      0,
			},
		})
	}
}

// handleChatStream handles the streaming chat endpoint
func handleChatStream(bedrockService *BedrockService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var chatReq ChatRequest
		if err := c.ShouldBindJSON(&chatReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set headers for SSE
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")

		// Process chat with streaming
		stream, err := bedrockService.ProcessChatStream(c.Request.Context(), chatReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Stream the response
		for event := range stream.GetStream().Events() {
			// Log the event type for debugging
			log.Printf("Event type: %T", event)

			// Try to extract bytes using reflection or type assertion
			// This is a simplified approach - just print the event type and continue
			c.Writer.Write([]byte("data: {\"content\": \"Streaming not fully implemented yet\"}\n\n"))
			c.Writer.Flush()
		}

		// Send the [DONE] message
		c.Writer.Write([]byte("data: [DONE]\n\n"))
		c.Writer.Flush()
	}
}

// handleListModels handles the list models endpoint
func handleListModels(bedrockService *BedrockService) gin.HandlerFunc {
	return func(c *gin.Context) {
		models, err := bedrockService.ListBedrockModels(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"models": models})
	}
}

// handleEmbeddings handles the embeddings endpoint
func handleEmbeddings(bedrockService *BedrockService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var embeddingsReq EmbeddingsRequest
		if err := c.ShouldBindJSON(&embeddingsReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		response, err := bedrockService.ProcessEmbeddings(c.Request.Context(), embeddingsReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

// extractBytesFromEvent attempts to extract bytes from an event
func extractBytesFromEvent(event interface{}) ([]byte, error) {
	// Use reflection or type assertions to try to extract bytes
	// This is a simplified example
	return nil, errors.New("unable to extract bytes from event")
}
