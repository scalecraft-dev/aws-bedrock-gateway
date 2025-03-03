package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for the application
func SetupRoutes(r gin.IRouter, bedrockService *BedrockService) {
	// Chat endpoint
	r.POST("/chat/completions", handleChat(bedrockService))

	// Stream chat endpoint
	r.POST("/chat/completions/stream", handleChatStream(bedrockService))

	// List models endpoint
	r.GET("/models", handleListModels(bedrockService))
}

// handleChat handles the chat completion endpoint
func handleChat(bedrockService *BedrockService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var chatReq ChatRequest
		if err := c.ShouldBindJSON(&chatReq); err != nil {
			log.Printf("Error binding JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("Received chat request: %+v", chatReq)
		response, err := bedrockService.ProcessChat(c.Request.Context(), chatReq)
		if err != nil {
			log.Printf("Error processing chat: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ChatResponse{
			ID:      GenerateMessageID(),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   chatReq.Model,
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
				PromptTokens:     1, // TODO: Implement actual token counting
				CompletionTokens: 1,
				TotalTokens:      2,
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

		// Format response in OpenAI-compatible format
		modelList := make([]gin.H, len(models))
		for i, model := range models {
			modelList[i] = gin.H{
				"id":       model,
				"object":   "model",
				"created":  1706745600,                   // You might want to adjust this timestamp
				"owned_by": strings.Split(model, ".")[0], // Extract owner from model ID
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": modelList})
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
