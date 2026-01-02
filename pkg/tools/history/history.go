package history

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/storage"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
)

type Input struct {
	Action string `json:"action" validate:"required,oneof=list get delete clear"`
	ID     uint   `json:"id,omitempty"`
	Limit  int    `json:"limit,omitempty" validate:"min=0,max=100"`
	Offset int    `json:"offset,omitempty" validate:"min=0"`
}

type Tool struct {
	logger    zerolog.Logger
	validator *validator.Validate
	store     storage.Storage
}

func (t *Tool) Register(srv *server.Server) error {
	tool := &mcp.Tool{
		Name:        "history",
		Description: "Browse and manage tool execution history. Actions: list (paginated), get (by ID), delete (by ID), clear (all).",
	}

	t.store = srv.Storage()

	mcp.AddTool(&srv.Server, tool, t.HistoryHandler)
	t.logger.Debug().Msg("history tool registered")

	return nil
}

func (t *Tool) HistoryHandler(ctx context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, any, error) {
	if err := t.validator.Struct(input); err != nil {
		return nil, nil, fmt.Errorf("validation error: %w", err)
	}

	var resultText string

	switch input.Action {
	case "list":
		limit := input.Limit
		if limit == 0 {
			limit = 10
		}
		executions, total, err := t.store.GetToolExecutions(ctx, limit, input.Offset)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list executions: %w", err)
		}
		data, _ := json.MarshalIndent(map[string]any{
			"total":      total,
			"limit":      limit,
			"offset":     input.Offset,
			"executions": executions,
		}, "", "  ")
		resultText = string(data)

	case "get":
		if input.ID == 0 {
			return nil, nil, fmt.Errorf("id is required for get action")
		}
		exec, err := t.store.GetToolExecution(ctx, input.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("execution not found: %w", err)
		}
		data, _ := json.MarshalIndent(exec, "", "  ")
		resultText = string(data)

	case "delete":
		if input.ID == 0 {
			return nil, nil, fmt.Errorf("id is required for delete action")
		}
		if err := t.store.DeleteToolExecution(ctx, input.ID); err != nil {
			return nil, nil, fmt.Errorf("failed to delete execution: %w", err)
		}
		resultText = fmt.Sprintf("Execution %d deleted successfully", input.ID)

	case "clear":
		if err := t.store.DeleteAllToolExecutions(ctx); err != nil {
			return nil, nil, fmt.Errorf("failed to clear executions: %w", err)
		}
		resultText = "All execution history cleared"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}

func New(logger zerolog.Logger) tools.Tool {
	return &Tool{
		logger:    logger.With().Str("tool", "history").Logger(),
		validator: validator.New(),
	}
}
