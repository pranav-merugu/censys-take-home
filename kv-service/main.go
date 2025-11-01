package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "github.com/pranavmerugu/censys-take-home/proto"
	"google.golang.org/grpc"
)

type kvServer struct {
	pb.UnimplementedKVStoreServer
	mu    sync.RWMutex
	store map[string]string
}

// newKVServer creates a new KV store server instance with an empty map
func newKVServer() *kvServer {
	return &kvServer{
		store: make(map[string]string),
	}
}

// Set stores a key-value pair in the map using a write lock
func (s *kvServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.store[req.Key] = req.Value
	log.Printf("Set key=%s, value=%s", req.Key, req.Value)

	return &pb.SetResponse{
		Success: true,
		Message: fmt.Sprintf("Key '%s' set successfully", req.Key),
	}, nil
}

// Get retrieves a value by key from the map using a read lock
func (s *kvServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, found := s.store[req.Key]
	log.Printf("Get key=%s, found=%v", req.Key, found)

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

// Delete removes a key-value pair from the map using a write lock
func (s *kvServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, found := s.store[req.Key]
	if !found {
		log.Printf("Delete key=%s, found=false", req.Key)
		return &pb.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("Key '%s' not found", req.Key),
		}, nil
	}

	delete(s.store, req.Key)
	log.Printf("Delete key=%s, success=true", req.Key)

	return &pb.DeleteResponse{
		Success: true,
		Message: fmt.Sprintf("Key '%s' deleted successfully", req.Key),
	}, nil
}

func main() {
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterKVStoreServer(grpcServer, newKVServer())

	log.Printf("KV Store gRPC server listening on %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
