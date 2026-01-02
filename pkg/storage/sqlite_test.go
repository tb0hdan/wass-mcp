package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tb0hdan/wass-mcp/pkg/models"
)

func setupTestDB(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()

	// Create temp file for test database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	cfg := Config{
		DatabasePath: tmpFile.Name(),
		Debug:        false,
	}

	store, err := NewSQLiteStorage(cfg)
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

func TestNewSQLiteStorage(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	if store == nil {
		t.Fatal("expected non-nil storage")
	}
	if store.db == nil {
		t.Fatal("expected non-nil database connection")
	}
}

func TestNewSQLiteStorage_InvalidPath(t *testing.T) {
	cfg := Config{
		DatabasePath: "/nonexistent/path/test.db",
		Debug:        false,
	}

	_, err := NewSQLiteStorage(cfg)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}

func TestCreateToolExecution(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	exec := &models.ToolExecution{
		SessionID:  "test-session-123",
		ToolName:   "nikto",
		InputJSON:  `{"host": "localhost", "port": 80}`,
		OutputJSON: `{"result": "scan complete"}`,
		DurationMs: 1500,
		Success:    true,
	}

	err := store.CreateToolExecution(ctx, exec)
	if err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	if exec.ID == 0 {
		t.Error("expected non-zero ID after creation")
	}
	if exec.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestGetToolExecution(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create test execution
	exec := &models.ToolExecution{
		SessionID:  "test-session",
		ToolName:   "nikto",
		InputJSON:  `{"host": "example.com"}`,
		DurationMs: 1000,
		Success:    true,
	}
	if err := store.CreateToolExecution(ctx, exec); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	// Retrieve it
	retrieved, err := store.GetToolExecution(ctx, exec.ID)
	if err != nil {
		t.Fatalf("failed to get execution: %v", err)
	}

	if retrieved.ID != exec.ID {
		t.Errorf("expected ID %d, got %d", exec.ID, retrieved.ID)
	}
	if retrieved.ToolName != "nikto" {
		t.Errorf("expected tool name 'nikto', got '%s'", retrieved.ToolName)
	}
	if retrieved.SessionID != "test-session" {
		t.Errorf("expected session ID 'test-session', got '%s'", retrieved.SessionID)
	}
}

func TestGetToolExecution_NotFound(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	_, err := store.GetToolExecution(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent execution")
	}
}

func TestGetToolExecutions(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple executions
	for i := 0; i < 15; i++ {
		exec := &models.ToolExecution{
			ToolName:   "nikto",
			InputJSON:  `{}`,
			DurationMs: int64(i * 100),
			Success:    true,
		}
		if err := store.CreateToolExecution(ctx, exec); err != nil {
			t.Fatalf("failed to create execution %d: %v", i, err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Test pagination
	executions, total, err := store.GetToolExecutions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}

	if total != 15 {
		t.Errorf("expected total 15, got %d", total)
	}
	if len(executions) != 10 {
		t.Errorf("expected 10 executions, got %d", len(executions))
	}

	// Test offset
	executions, _, err = store.GetToolExecutions(ctx, 10, 10)
	if err != nil {
		t.Fatalf("failed to get executions with offset: %v", err)
	}
	if len(executions) != 5 {
		t.Errorf("expected 5 executions with offset, got %d", len(executions))
	}
}

func TestGetToolExecutions_Empty(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	executions, total, err := store.GetToolExecutions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get executions: %v", err)
	}

	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(executions) != 0 {
		t.Errorf("expected 0 executions, got %d", len(executions))
	}
}

func TestGetToolExecutionsBySession(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create executions with different sessions
	sessions := []string{"session-a", "session-b", "session-a", "session-c"}
	for _, sid := range sessions {
		exec := &models.ToolExecution{
			SessionID: sid,
			ToolName:  "nikto",
			Success:   true,
		}
		if err := store.CreateToolExecution(ctx, exec); err != nil {
			t.Fatalf("failed to create execution: %v", err)
		}
	}

	// Get executions for session-a
	executions, err := store.GetToolExecutionsBySession(ctx, "session-a")
	if err != nil {
		t.Fatalf("failed to get executions by session: %v", err)
	}

	if len(executions) != 2 {
		t.Errorf("expected 2 executions for session-a, got %d", len(executions))
	}

	for _, exec := range executions {
		if exec.SessionID != "session-a" {
			t.Errorf("expected session-a, got %s", exec.SessionID)
		}
	}
}

func TestGetToolExecutionsByTool(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create executions with different tools
	tools := []string{"nikto", "wapiti", "nikto", "nikto", "wapiti"}
	for _, toolName := range tools {
		exec := &models.ToolExecution{
			ToolName: toolName,
			Success:  true,
		}
		if err := store.CreateToolExecution(ctx, exec); err != nil {
			t.Fatalf("failed to create execution: %v", err)
		}
	}

	// Get executions for nikto
	executions, err := store.GetToolExecutionsByTool(ctx, "nikto", 0)
	if err != nil {
		t.Fatalf("failed to get executions by tool: %v", err)
	}

	if len(executions) != 3 {
		t.Errorf("expected 3 nikto executions, got %d", len(executions))
	}

	// Test with limit
	executions, err = store.GetToolExecutionsByTool(ctx, "nikto", 2)
	if err != nil {
		t.Fatalf("failed to get executions by tool with limit: %v", err)
	}

	if len(executions) != 2 {
		t.Errorf("expected 2 nikto executions with limit, got %d", len(executions))
	}
}

func TestDeleteToolExecution(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create execution
	exec := &models.ToolExecution{
		ToolName: "nikto",
		Success:  true,
	}
	if err := store.CreateToolExecution(ctx, exec); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	// Delete it
	if err := store.DeleteToolExecution(ctx, exec.ID); err != nil {
		t.Fatalf("failed to delete execution: %v", err)
	}

	// Verify it's deleted (soft delete, so GetToolExecution should fail)
	_, err := store.GetToolExecution(ctx, exec.ID)
	if err == nil {
		t.Error("expected error when getting deleted execution")
	}
}

func TestDeleteAllToolExecutions(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple executions
	for i := 0; i < 5; i++ {
		exec := &models.ToolExecution{
			ToolName: "nikto",
			Success:  true,
		}
		if err := store.CreateToolExecution(ctx, exec); err != nil {
			t.Fatalf("failed to create execution: %v", err)
		}
	}

	// Verify they exist
	_, total, _ := store.GetToolExecutions(ctx, 10, 0)
	if total != 5 {
		t.Fatalf("expected 5 executions before delete, got %d", total)
	}

	// Delete all
	if err := store.DeleteAllToolExecutions(ctx); err != nil {
		t.Fatalf("failed to delete all executions: %v", err)
	}

	// Verify all deleted
	_, total, _ = store.GetToolExecutions(ctx, 10, 0)
	if total != 0 {
		t.Errorf("expected 0 executions after delete all, got %d", total)
	}
}

func TestClose(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	err := store.Close()
	if err != nil {
		t.Fatalf("failed to close storage: %v", err)
	}
}

func TestToolExecution_WithError(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	exec := &models.ToolExecution{
		ToolName:     "nikto",
		InputJSON:    `{"host": "invalid"}`,
		ErrorMessage: "connection refused",
		DurationMs:   50,
		Success:      false,
	}

	err := store.CreateToolExecution(ctx, exec)
	if err != nil {
		t.Fatalf("failed to create failed execution: %v", err)
	}

	retrieved, err := store.GetToolExecution(ctx, exec.ID)
	if err != nil {
		t.Fatalf("failed to get execution: %v", err)
	}

	if retrieved.Success {
		t.Error("expected Success to be false")
	}
	if retrieved.ErrorMessage != "connection refused" {
		t.Errorf("expected error message 'connection refused', got '%s'", retrieved.ErrorMessage)
	}
}
