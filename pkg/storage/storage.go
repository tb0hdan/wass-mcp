package storage

import (
	"context"

	"github.com/tb0hdan/wass-mcp/pkg/models"
)

type Storage interface {
	// Tool execution operations
	CreateToolExecution(ctx context.Context, exec *models.ToolExecution) error
	GetToolExecution(ctx context.Context, id uint) (*models.ToolExecution, error)
	GetToolExecutions(ctx context.Context, limit, offset int) ([]models.ToolExecution, int64, error)
	GetToolExecutionsBySession(ctx context.Context, sessionID string) ([]models.ToolExecution, error)
	GetToolExecutionsByTool(ctx context.Context, toolName string, limit int) ([]models.ToolExecution, error)
	DeleteToolExecution(ctx context.Context, id uint) error
	DeleteAllToolExecutions(ctx context.Context) error

	// Lifecycle
	Close() error
}
