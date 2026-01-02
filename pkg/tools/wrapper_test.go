package tools

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tb0hdan/wass-mcp/pkg/storage"
)

type testInput struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func setupTestStorage(t *testing.T) (storage.Storage, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "wrapper-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	cfg := storage.Config{
		DatabasePath: tmpFile.Name(),
		Debug:        false,
	}

	store, err := storage.NewSQLiteStorage(cfg)
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to create storage: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.Remove(tmpFile.Name())
	}

	return store, cleanup
}

func TestWrapToolHandler_Success(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "success"},
			},
		}, nil, nil
	}

	wrapped := WrapToolHandler(store, "test-tool", handler)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := testInput{Host: "localhost", Port: 80}

	result, _, err := wrapped(ctx, req, input)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	// Wait for async logging
	time.Sleep(100 * time.Millisecond)

	// Verify execution was logged
	executions, total, err := store.GetToolExecutions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 execution logged, got %d", total)
	}
	if len(executions) > 0 {
		if executions[0].ToolName != "test-tool" {
			t.Errorf("expected tool name 'test-tool', got '%s'", executions[0].ToolName)
		}
		if !executions[0].Success {
			t.Error("expected Success to be true")
		}
	}
}

func TestWrapToolHandler_Error(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	expectedErr := errors.New("test error")
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, any, error) {
		return nil, nil, expectedErr
	}

	wrapped := WrapToolHandler(store, "test-tool", handler)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := testInput{Host: "localhost", Port: 80}

	_, _, err := wrapped(ctx, req, input)

	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", err.Error())
	}

	// Wait for async logging
	time.Sleep(100 * time.Millisecond)

	// Verify failed execution was logged
	executions, _, err := store.GetToolExecutions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if len(executions) > 0 {
		if executions[0].Success {
			t.Error("expected Success to be false for failed execution")
		}
		if executions[0].ErrorMessage != "test error" {
			t.Errorf("expected error message 'test error', got '%s'", executions[0].ErrorMessage)
		}
	}
}

func TestWrapToolHandler_InputSerialization(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{}, nil, nil
	}

	wrapped := WrapToolHandler(store, "test-tool", handler)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := testInput{Host: "example.com", Port: 443}

	_, _, _ = wrapped(ctx, req, input)

	// Wait for async logging
	time.Sleep(100 * time.Millisecond)

	// Verify input was serialized
	executions, _, err := store.GetToolExecutions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if len(executions) > 0 {
		if executions[0].InputJSON == "" {
			t.Error("expected InputJSON to be set")
		}
		if !containsString(executions[0].InputJSON, "example.com") {
			t.Errorf("expected InputJSON to contain 'example.com', got '%s'", executions[0].InputJSON)
		}
		if !containsString(executions[0].InputJSON, "443") {
			t.Errorf("expected InputJSON to contain '443', got '%s'", executions[0].InputJSON)
		}
	}
}

func TestWrapToolHandler_DurationTracking(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, any, error) {
		time.Sleep(50 * time.Millisecond)
		return &mcp.CallToolResult{}, nil, nil
	}

	wrapped := WrapToolHandler(store, "test-tool", handler)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := testInput{}

	_, _, _ = wrapped(ctx, req, input)

	// Wait for async logging
	time.Sleep(100 * time.Millisecond)

	// Verify duration was tracked
	executions, _, err := store.GetToolExecutions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if len(executions) > 0 {
		if executions[0].DurationMs < 50 {
			t.Errorf("expected DurationMs >= 50, got %d", executions[0].DurationMs)
		}
	}
}

func TestWrapToolHandler_MultipleExecutions(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	callCount := 0
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, any, error) {
		callCount++
		return &mcp.CallToolResult{}, nil, nil
	}

	wrapped := WrapToolHandler(store, "test-tool", handler)

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := testInput{}

	// Execute multiple times
	for i := 0; i < 5; i++ {
		_, _, _ = wrapped(ctx, req, input)
	}

	// Wait for async logging
	time.Sleep(200 * time.Millisecond)

	if callCount != 5 {
		t.Errorf("expected handler to be called 5 times, got %d", callCount)
	}

	// Verify all executions were logged
	_, total, err := store.GetToolExecutions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}
	if total != 5 {
		t.Errorf("expected 5 executions logged, got %d", total)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
