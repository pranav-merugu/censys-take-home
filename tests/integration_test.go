package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	pb "github.com/pranavmerugu/censys-take-home/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// KV Server implementation (copied from kv-service for integration testing)
type kvServer struct {
	pb.UnimplementedKVStoreServer
	mu    sync.RWMutex
	store map[string]string
}

func newKVServer() *kvServer {
	return &kvServer{
		store: make(map[string]string),
	}
}

func (s *kvServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[req.Key] = req.Value

	return &pb.SetResponse{
		Success: true,
		Message: fmt.Sprintf("Key '%s' set successfully", req.Key),
	}, nil
}

func (s *kvServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, found := s.store[req.Key]

	if !found {
		return &pb.GetResponse{
			Found:   false,
			Value:   "",
			Message: fmt.Sprintf("Key '%s' not found", req.Key),
		}, nil
	}

	return &pb.GetResponse{
		Found:   true,
		Value:   value,
		Message: "Key retrieved successfully",
	}, nil
}

func (s *kvServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, found := s.store[req.Key]
	if !found {
		return &pb.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("Key '%s' not found", req.Key),
		}, nil
	}

	delete(s.store, req.Key)

	return &pb.DeleteResponse{
		Success: true,
		Message: fmt.Sprintf("Key '%s' deleted successfully", req.Key),
	}, nil
}

// API Server implementation (copied from api-service for integration testing)
type APIServer struct {
	kvClient pb.KVStoreClient
}

type SetRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
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

func NewAPIServer(kvClient pb.KVStoreClient) *APIServer {
	return &APIServer{
		kvClient: kvClient,
	}
}

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

// Test infrastructure
type testEnv struct {
	grpcServer *grpc.Server
	httpServer *http.Server
	httpAddr   string
}

func setupTestEnvironment(t *testing.T) *testEnv {
	// Start gRPC server
	grpcListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create gRPC listener: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterKVStoreServer(grpcServer, newKVServer())

	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	grpcAddr := grpcListener.Addr().String()
	time.Sleep(100 * time.Millisecond)

	// Connect to gRPC server
	conn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		grpcServer.Stop()
		t.Fatalf("Failed to connect to gRPC server: %v", err)
	}

	kvClient := pb.NewKVStoreClient(conn)
	apiServer := NewAPIServer(kvClient)

	// Set up HTTP server
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/kv", apiServer.SetHandler)
	router.GET("/kv/:key", apiServer.GetHandler)
	router.DELETE("/kv/:key", apiServer.DeleteHandler)

	httpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		grpcServer.Stop()
		t.Fatalf("Failed to create HTTP listener: %v", err)
	}

	httpServer := &http.Server{Handler: router}
	go func() {
		if err := httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	httpAddr := httpListener.Addr().String()
	time.Sleep(100 * time.Millisecond)

	return &testEnv{
		grpcServer: grpcServer,
		httpServer: httpServer,
		httpAddr:   httpAddr,
	}
}

func (e *testEnv) cleanup() {
	if e.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		e.httpServer.Shutdown(ctx)
	}
	if e.grpcServer != nil {
		e.grpcServer.Stop()
	}
}

// Integration test - tests the full workflow
func TestIntegrationSetGetDelete(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	baseURL := fmt.Sprintf("http://%s", env.httpAddr)

	// Set a key-value pair
	setReq := SetRequest{Key: "test-key", Value: "test-value"}
	setBody, _ := json.Marshal(setReq)
	resp, err := http.Post(baseURL+"/kv", "application/json", bytes.NewBuffer(setBody))
	if err != nil {
		t.Fatalf("Failed to send Set request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Set request failed with status %d", resp.StatusCode)
	}

	// Get the value back
	getResp, err := http.Get(baseURL + "/kv/test-key")
	if err != nil {
		t.Fatalf("Failed to send Get request: %v", err)
	}
	defer getResp.Body.Close()

	var getResult GetResponse
	json.NewDecoder(getResp.Body).Decode(&getResult)

	if !getResult.Found || getResult.Value != "test-value" {
		t.Errorf("Expected value 'test-value', got '%s'", getResult.Value)
	}

	// Delete the key
	client := &http.Client{}
	delReq, _ := http.NewRequest(http.MethodDelete, baseURL+"/kv/test-key", nil)
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("Failed to send Delete request: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusOK {
		t.Errorf("Delete request failed with status %d", delResp.StatusCode)
	}

	// Verify deletion
	getResp2, _ := http.Get(baseURL + "/kv/test-key")
	defer getResp2.Body.Close()

	if getResp2.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 after deletion, got %d", getResp2.StatusCode)
	}
}

// Integration test - tests getting a non-existent key
func TestIntegrationGetNonExistentKey(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.cleanup()

	baseURL := fmt.Sprintf("http://%s", env.httpAddr)

	// Try to get a key that doesn't exist
	getResp, err := http.Get(baseURL + "/kv/nonexistent-key")
	if err != nil {
		t.Fatalf("Failed to send Get request: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", getResp.StatusCode)
	}

	var result GetResponse
	json.NewDecoder(getResp.Body).Decode(&result)

	if result.Found {
		t.Error("Expected found=false for non-existent key")
	}
}
