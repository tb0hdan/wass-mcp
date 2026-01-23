package history

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/wass-mcp/pkg/models"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/storage"
)

func setupTestServer(t *testing.T) (*server.Server, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "history-test-*.db")
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

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := server.NewServer(impl, store)

	cleanup := func() {
		srv.Shutdown(context.Background())
		os.Remove(tmpFile.Name())
	}

	return srv, cleanup
}

func TestNew(t *testing.T) {
	logger := zerolog.New(os.Stdout)
	tool := New(logger)

	if tool == nil {
		t.Fatal("expected non-nil tool")
	}
}

func TestHistoryHandler_List_Empty(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = srv.Storage()

	ctx := context.Background()
	input := Input{Action: "list"}

	result, _, err := tool.HistoryHandler(ctx, nil, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Parse response
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected TextContent")
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["total"].(float64) != 0 {
		t.Errorf("expected total 0, got %v", response["total"])
	}
}

func TestHistoryHandler_List_WithData(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	store := srv.Storage()

	// Add test executions
	for i := 0; i < 15; i++ {
		exec := &models.ToolExecution{
			ToolName: "nikto",
			Success:  true,
		}
		if err := store.CreateToolExecution(ctx, exec); err != nil {
			t.Fatalf("failed to create execution: %v", err)
		}
	}

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = store

	input := Input{Action: "list", Limit: 10}

	result, _, err := tool.HistoryHandler(ctx, nil, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent := result.Content[0].(*mcp.TextContent)
	var response map[string]any
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["total"].(float64) != 15 {
		t.Errorf("expected total 15, got %v", response["total"])
	}

	executions := response["executions"].([]any)
	if len(executions) != 10 {
		t.Errorf("expected 10 executions (limit), got %d", len(executions))
	}
}

func TestHistoryHandler_List_Pagination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	store := srv.Storage()

	// Add test executions
	for i := 0; i < 20; i++ {
		exec := &models.ToolExecution{
			ToolName: "nikto",
			Success:  true,
		}
		store.CreateToolExecution(ctx, exec)
	}

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = store

	input := Input{Action: "list", Limit: 5, Offset: 10}

	result, _, err := tool.HistoryHandler(ctx, nil, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent := result.Content[0].(*mcp.TextContent)
	var response map[string]any
	json.Unmarshal([]byte(textContent.Text), &response)

	if response["offset"].(float64) != 10 {
		t.Errorf("expected offset 10, got %v", response["offset"])
	}
	if response["limit"].(float64) != 5 {
		t.Errorf("expected limit 5, got %v", response["limit"])
	}
}

func TestHistoryHandler_Get(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	store := srv.Storage()

	// Create test execution
	exec := &models.ToolExecution{
		ToolName:  "nikto",
		InputJSON: `{"host": "test.com"}`,
		Success:   true,
	}
	store.CreateToolExecution(ctx, exec)

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = store

	input := Input{Action: "get", ID: exec.ID}

	result, _, err := tool.HistoryHandler(ctx, nil, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent := result.Content[0].(*mcp.TextContent)
	var response models.ToolExecution
	if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.ID != exec.ID {
		t.Errorf("expected ID %d, got %d", exec.ID, response.ID)
	}
	if response.ToolName != "nikto" {
		t.Errorf("expected tool name 'nikto', got '%s'", response.ToolName)
	}
}

func TestHistoryHandler_Get_NotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = srv.Storage()

	ctx := context.Background()
	input := Input{Action: "get", ID: 99999}

	_, _, err := tool.HistoryHandler(ctx, nil, input)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}
}

func TestHistoryHandler_Get_NoID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = srv.Storage()

	ctx := context.Background()
	input := Input{Action: "get", ID: 0}

	_, _, err := tool.HistoryHandler(ctx, nil, input)
	if err == nil {
		t.Fatal("expected error when ID is not provided")
	}
}

func TestHistoryHandler_Delete(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	store := srv.Storage()

	// Create test execution
	exec := &models.ToolExecution{
		ToolName: "nikto",
		Success:  true,
	}
	store.CreateToolExecution(ctx, exec)

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = store

	input := Input{Action: "delete", ID: exec.ID}

	result, _, err := tool.HistoryHandler(ctx, nil, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent := result.Content[0].(*mcp.TextContent)
	if textContent.Text == "" {
		t.Error("expected success message")
	}

	// Verify deletion
	_, err = store.GetToolExecution(ctx, exec.ID)
	if err == nil {
		t.Error("expected error when getting deleted execution")
	}
}

func TestHistoryHandler_Delete_NoID(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = srv.Storage()

	ctx := context.Background()
	input := Input{Action: "delete", ID: 0}

	_, _, err := tool.HistoryHandler(ctx, nil, input)
	if err == nil {
		t.Fatal("expected error when ID is not provided for delete")
	}
}

func TestHistoryHandler_Clear(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	store := srv.Storage()

	// Create test executions
	for i := 0; i < 5; i++ {
		exec := &models.ToolExecution{
			ToolName: "nikto",
			Success:  true,
		}
		store.CreateToolExecution(ctx, exec)
	}

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = store

	input := Input{Action: "clear"}

	result, _, err := tool.HistoryHandler(ctx, nil, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent := result.Content[0].(*mcp.TextContent)
	if textContent.Text != "All execution history cleared" {
		t.Errorf("unexpected message: %s", textContent.Text)
	}

	// Verify all deleted
	_, total, _ := store.GetToolExecutions(ctx, 10, 0)
	if total != 0 {
		t.Errorf("expected 0 executions after clear, got %d", total)
	}
}

func TestHistoryHandler_InvalidAction(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = srv.Storage()

	ctx := context.Background()
	input := Input{Action: "invalid"}

	_, _, err := tool.HistoryHandler(ctx, nil, input)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestHistoryHandler_DefaultLimit(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	store := srv.Storage()

	// Create 15 executions
	for i := 0; i < 15; i++ {
		exec := &models.ToolExecution{
			ToolName: "nikto",
			Success:  true,
		}
		store.CreateToolExecution(ctx, exec)
	}

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)
	tool.store = store

	// Don't specify limit - should default to 10
	input := Input{Action: "list"}

	result, _, _ := tool.HistoryHandler(ctx, nil, input)

	textContent := result.Content[0].(*mcp.TextContent)
	var response map[string]any
	json.Unmarshal([]byte(textContent.Text), &response)

	executions := response["executions"].([]any)
	if len(executions) != 10 {
		t.Errorf("expected default limit of 10, got %d", len(executions))
	}
}

func TestRegister(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)

	err := tool.Register(srv)
	if err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	// Verify the store was set
	if tool.store == nil {
		t.Error("expected store to be set after Register()")
	}
}

func TestRegister_SetsStorage(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	logger := zerolog.New(os.Stdout)
	tool := New(logger).(*Tool)

	// Before register, store should be nil
	if tool.store != nil {
		t.Error("expected store to be nil before Register()")
	}

	err := tool.Register(srv)
	if err != nil {
		t.Fatalf("Register() returned error: %v", err)
	}

	// After register, store should be set
	if tool.store == nil {
		t.Error("expected store to be set after Register()")
	}

	// Verify we can use it
	ctx := context.Background()
	_, total, err := tool.store.GetToolExecutions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to use store after Register(): %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 executions, got %d", total)
	}
}
