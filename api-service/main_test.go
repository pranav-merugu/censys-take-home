package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	pb "github.com/pranavmerugu/censys-take-home/proto"
	"google.golang.org/grpc"
)

// Mock KVStoreClient for testing
type mockKVClient struct {
	setFunc    func(ctx context.Context, req *pb.SetRequest, opts ...grpc.CallOption) (*pb.SetResponse, error)
	getFunc    func(ctx context.Context, req *pb.GetRequest, opts ...grpc.CallOption) (*pb.GetResponse, error)
	deleteFunc func(ctx context.Context, req *pb.DeleteRequest, opts ...grpc.CallOption) (*pb.DeleteResponse, error)
}

func (m *mockKVClient) Set(ctx context.Context, req *pb.SetRequest, opts ...grpc.CallOption) (*pb.SetResponse, error) {
	if m.setFunc != nil {
		return m.setFunc(ctx, req, opts...)
	}
	return &pb.SetResponse{Success: true, Message: "Key set successfully"}, nil
}

func (m *mockKVClient) Get(ctx context.Context, req *pb.GetRequest, opts ...grpc.CallOption) (*pb.GetResponse, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, req, opts...)
	}
	return &pb.GetResponse{Found: true, Value: "test-value", Message: "Key retrieved successfully"}, nil
}

func (m *mockKVClient) Delete(ctx context.Context, req *pb.DeleteRequest, opts ...grpc.CallOption) (*pb.DeleteResponse, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, req, opts...)
	}
	return &pb.DeleteResponse{Success: true, Message: "Key deleted successfully"}, nil
}

func setupRouter(apiServer *APIServer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/kv", apiServer.SetHandler)
	router.GET("/kv/:key", apiServer.GetHandler)
	router.DELETE("/kv/:key", apiServer.DeleteHandler)
	return router
}

func TestSetHandler(t *testing.T) {
	mockClient := &mockKVClient{}
	apiServer := NewAPIServer(mockClient)
	router := setupRouter(apiServer)

	reqBody := SetRequest{Key: "test-key", Value: "test-value"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/kv", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp SetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Errorf("Expected success=true, got %v", resp.Success)
	}
}

func TestGetHandler(t *testing.T) {
	mockClient := &mockKVClient{}
	apiServer := NewAPIServer(mockClient)
	router := setupRouter(apiServer)

	req := httptest.NewRequest(http.MethodGet, "/kv/test-key", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp GetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if !resp.Found {
		t.Errorf("Expected found=true, got %v", resp.Found)
	}
}

func TestDeleteHandler(t *testing.T) {
	mockClient := &mockKVClient{}
	apiServer := NewAPIServer(mockClient)
	router := setupRouter(apiServer)

	req := httptest.NewRequest(http.MethodDelete, "/kv/test-key", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp DeleteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if !resp.Success {
		t.Errorf("Expected success=true, got %v", resp.Success)
	}
}

func TestSetHandlerMissingFields(t *testing.T) {
	mockClient := &mockKVClient{}
	apiServer := NewAPIServer(mockClient)
	router := setupRouter(apiServer)

	// Missing value field
	reqBody := map[string]string{"key": "test-key"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/kv", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetHandlerNonExistentKey(t *testing.T) {
	mockClient := &mockKVClient{
		getFunc: func(ctx context.Context, req *pb.GetRequest, opts ...grpc.CallOption) (*pb.GetResponse, error) {
			return &pb.GetResponse{Found: false, Message: "Key not found"}, nil
		},
	}
	apiServer := NewAPIServer(mockClient)
	router := setupRouter(apiServer)

	req := httptest.NewRequest(http.MethodGet, "/kv/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp GetResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if resp.Found {
		t.Errorf("Expected found=false, got %v", resp.Found)
	}
}

func TestDeleteHandlerNonExistentKey(t *testing.T) {
	mockClient := &mockKVClient{
		deleteFunc: func(ctx context.Context, req *pb.DeleteRequest, opts ...grpc.CallOption) (*pb.DeleteResponse, error) {
			return &pb.DeleteResponse{Success: false, Message: "Key not found"}, nil
		},
	}
	apiServer := NewAPIServer(mockClient)
	router := setupRouter(apiServer)

	req := httptest.NewRequest(http.MethodDelete, "/kv/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp DeleteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}
	if resp.Success {
		t.Errorf("Expected success=false, got %v", resp.Success)
	}
}
