package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/pranavmerugu/censys-take-home/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type APIServer struct {
	kvClient pb.KVStoreClient
}

type SetRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

type GetRequest struct {
	Key string `json:"key" binding:"required"`
}

type DeleteRequest struct {
	Key string `json:"key" binding:"required"`
}

type SetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GetResponse struct {
	Found   bool   `json:"found"`
	Message string `json:"message"`
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
}

type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// NewAPIServer creates a new API server instance with a gRPC client
func NewAPIServer(kvClient pb.KVStoreClient) *APIServer {
	return &APIServer{
		kvClient: kvClient,
	}
}

// SetHandler handles POST requests to store a key-value pair
func (s *APIServer) SetHandler(c *gin.Context) {
	var req SetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request: " + err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.kvClient.Set(ctx, &pb.SetRequest{
		Key:   req.Key,
		Value: req.Value,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to set key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SetResponse{
		Success: resp.Success,
		Message: resp.Message,
	})
}

// GetHandler handles GET requests to retrieve a value by key
func (s *APIServer) GetHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Key parameter is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.kvClient.Get(ctx, &pb.GetRequest{
		Key: key,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get key: " + err.Error(),
		})
		return
	}

	if !resp.Found {
		c.JSON(http.StatusNotFound, GetResponse{
			Found:   false,
			Message: resp.Message,
		})
		return
	}

	c.JSON(http.StatusOK, GetResponse{
		Found:   true,
		Message: resp.Message,
		Key:     key,
		Value:   resp.Value,
	})
}

// DeleteHandler handles DELETE requests to remove a key-value pair
func (s *APIServer) DeleteHandler(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Key parameter is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.kvClient.Delete(ctx, &pb.DeleteRequest{
		Key: key,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to delete key: " + err.Error(),
		})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusNotFound, DeleteResponse{
			Success: false,
			Message: resp.Message,
		})
		return
	}

	c.JSON(http.StatusOK, DeleteResponse{
		Success: resp.Success,
		Message: resp.Message,
	})
}

func main() {
	// Get KV service address from environment variable or use default
	kvServiceAddr := os.Getenv("KV_SERVICE_ADDR")
	if kvServiceAddr == "" {
		kvServiceAddr = "localhost:50051"
	}

	// Connect to KV store gRPC service
	conn, err := grpc.NewClient(kvServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to KV service: %v", err)
	}
	defer conn.Close()

	kvClient := pb.NewKVStoreClient(conn)
	apiServer := NewAPIServer(kvClient)

	// Set up Gin router
	router := gin.Default()

	// Define REST API endpoints
	router.POST("/kv", apiServer.SetHandler)
	router.GET("/kv/:key", apiServer.GetHandler)
	router.DELETE("/kv/:key", apiServer.DeleteHandler)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("REST API server listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
