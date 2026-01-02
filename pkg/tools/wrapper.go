package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tb0hdan/wass-mcp/pkg/models"
	"github.com/tb0hdan/wass-mcp/pkg/storage"
)

// WrapToolHandler wraps a tool handler to add execution logging.
func WrapToolHandler[In, Out any](
	store storage.Storage,
	toolName string,
	handler func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error),
) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		startTime := time.Now()

		// Get session ID from request
		sessionID := ""
		if req.Session != nil {
			sessionID = req.Session.ID()
		}

		// Marshal input for logging
		inputJSON, _ := json.Marshal(input)

		// Execute the actual handler
		result, output, err := handler(ctx, req, input)

		duration := time.Since(startTime)

		// Create execution record
		exec := &models.ToolExecution{
			SessionID:  sessionID,
			ToolName:   toolName,
			InputJSON:  string(inputJSON),
			DurationMs: duration.Milliseconds(),
			Success:    err == nil,
		}

		if err != nil {
			exec.ErrorMessage = err.Error()
		} else if result != nil {
			outputJSON, _ := json.Marshal(result)
			exec.OutputJSON = string(outputJSON)
		}

		// Log execution asynchronously to avoid blocking.
		// Using background context intentionally - logging should complete even if request is cancelled.
		go func() { //nolint:contextcheck
			_ = store.CreateToolExecution(context.Background(), exec)
		}()

		return result, output, err
	}
}
