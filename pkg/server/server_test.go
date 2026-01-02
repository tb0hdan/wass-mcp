package server

import (
	"context"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tb0hdan/wass-mcp/pkg/models"
	"github.com/tb0hdan/wass-mcp/pkg/storage"
)

func setupTestStorage(t *testing.T) (storage.Storage, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "server-test-*.db")
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

func TestNewServer(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := NewServer(impl, store)

	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.storage == nil {
		t.Fatal("expected non-nil storage in server")
	}
}

func TestNewServer_NilStorage(t *testing.T) {
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := NewServer(impl, nil)

	if srv == nil {
		t.Fatal("expected non-nil server even with nil storage")
	}
	if srv.storage != nil {
		t.Error("expected nil storage when nil is passed")
	}
}

func TestServer_Storage(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := NewServer(impl, store)

	retrievedStorage := srv.Storage()
	if retrievedStorage == nil {
		t.Fatal("Storage() returned nil")
	}

	// Verify it's the same storage by using it
	ctx := context.Background()
	exec := &models.ToolExecution{
		ToolName: "test",
		Success:  true,
	}
	if err := retrievedStorage.CreateToolExecution(ctx, exec); err != nil {
		t.Fatalf("failed to use retrieved storage: %v", err)
	}
}

func TestServer_Shutdown(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := NewServer(impl, store)

	ctx := context.Background()
	err := srv.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown() returned error: %v", err)
	}
}

func TestServer_Shutdown_NilStorage(t *testing.T) {
	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := NewServer(impl, nil)

	ctx := context.Background()
	err := srv.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown() with nil storage returned error: %v", err)
	}
}

func TestServer_EmbeddedMCPServer(t *testing.T) {
	store, cleanup := setupTestStorage(t)
	defer cleanup()

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := NewServer(impl, store)

	// Verify the server was created and embedded MCP server is accessible
	// We can't directly check the implementation name, but we can verify
	// the server is not nil and the embedded Server field is initialized
	if srv == nil {
		t.Fatal("expected non-nil server")
	}

	// The embedded mcp.Server should be initialized
	// We verify this indirectly by checking we can access Storage
	if srv.Storage() != store {
		t.Error("expected Storage() to return the same store passed to NewServer")
	}
}
