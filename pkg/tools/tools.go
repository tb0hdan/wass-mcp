package tools

import (
	"context"

	"github.com/tb0hdan/wass-mcp/pkg/server"
)

// Tool is the interface that all MCP tools must implement.
type Tool interface {
	Register(srv *server.Server) error
}

// ScanParams contains common parameters for scanner tools.
type ScanParams struct {
	Host  string
	Port  int
	Vhost string
}

// ScanResult contains the result of a scan operation.
type ScanResult struct {
	Output string
	Error  error
}

// Scanner is the interface that scanner tools implement for reuse.
type Scanner interface {
	Tool
	// Name returns the scanner name.
	Name() string
	// IsAvailable checks if the scanner binary is available.
	IsAvailable() bool
	// Scan performs the actual scan and returns the output.
	Scan(ctx context.Context, params ScanParams) ScanResult
}
