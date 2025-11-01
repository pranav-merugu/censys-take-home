package main

import (
	"context"
	"testing"

	pb "github.com/pranavmerugu/censys-take-home/proto"
)

func TestSet(t *testing.T) {
	server := newKVServer()
	ctx := context.Background()

	req := &pb.SetRequest{Key: "key1", Value: "value1"}
	resp, err := server.Set(ctx, req)

	if err != nil {
		t.Errorf("Set() error = %v", err)
	}
	if !resp.Success {
		t.Errorf("Set() success = false, want true")
	}

	// Verify the value was stored
	server.mu.RLock()
	storedValue := server.store["key1"]
	server.mu.RUnlock()

	if storedValue != "value1" {
		t.Errorf("Stored value = %v, want %v", storedValue, "value1")
	}
}

func TestGet(t *testing.T) {
	server := newKVServer()
	ctx := context.Background()

	// Pre-populate data
	server.store["existing"] = "value"

	req := &pb.GetRequest{Key: "existing"}
	resp, err := server.Get(ctx, req)

	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if !resp.Found {
		t.Errorf("Get() found = false, want true")
	}
	if resp.Value != "value" {
		t.Errorf("Get() value = %v, want %v", resp.Value, "value")
	}
}

func TestDelete(t *testing.T) {
	server := newKVServer()
	ctx := context.Background()

	// Pre-populate data
	server.store["toDelete"] = "value"

	req := &pb.DeleteRequest{Key: "toDelete"}
	resp, err := server.Delete(ctx, req)

	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
	if !resp.Success {
		t.Errorf("Delete() success = false, want true")
	}

	// Verify the key was deleted
	server.mu.RLock()
	_, exists := server.store["toDelete"]
	server.mu.RUnlock()

	if exists {
		t.Error("Key still exists after deletion")
	}
}

func TestGetNonExistentKey(t *testing.T) {
	server := newKVServer()
	ctx := context.Background()

	req := &pb.GetRequest{Key: "nonexistent"}
	resp, err := server.Get(ctx, req)

	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if resp.Found {
		t.Errorf("Get() found = true, want false for non-existent key")
	}
}

func TestDeleteNonExistentKey(t *testing.T) {
	server := newKVServer()
	ctx := context.Background()

	req := &pb.DeleteRequest{Key: "nonexistent"}
	resp, err := server.Delete(ctx, req)

	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}
	if resp.Success {
		t.Errorf("Delete() success = true, want false for non-existent key")
	}
}

func TestEmptyKey(t *testing.T) {
	server := newKVServer()
	ctx := context.Background()

	// Set with empty key
	setReq := &pb.SetRequest{Key: "", Value: "value"}
	setResp, err := server.Set(ctx, setReq)

	if err != nil {
		t.Errorf("Set() error = %v", err)
	}
	if !setResp.Success {
		t.Errorf("Set() success = false, want true (empty keys are allowed)")
	}

	// Get with empty key
	getReq := &pb.GetRequest{Key: ""}
	getResp, err := server.Get(ctx, getReq)

	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if !getResp.Found || getResp.Value != "value" {
		t.Errorf("Get() failed to retrieve empty key")
	}
}
