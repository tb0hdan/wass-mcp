package shcheck

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
)

const (
	binaryName  = "shcheck.py"
	description = "shcheck is a security headers checker that analyzes HTTP response headers for security best practices."
	headerVerb  = "output"
)

// Tool implements the shcheck security headers scanner.
type Tool struct {
	tools.BaseScanner
}

// Scan performs the shcheck scan and returns the output.
func (t *Tool) Scan(ctx context.Context, params tools.ScanParams) tools.ScanResult {
	host, port := t.ResolveHostPort(params.Host, params.Port)

	targetURL := tools.BuildTargetURL(host, port)
	t.Logger.Info().Msgf("Running shcheck scan on %s", targetURL)

	args := []string{"-j", "-d", targetURL}
	if params.Vhost != "" {
		args = append(args, "-a", fmt.Sprintf("Host: %s", params.Vhost))
	}

	cmd := exec.CommandContext(ctx, binaryName, args...) //nolint:gosec
	output, err := cmd.CombinedOutput()

	if err != nil {
		return tools.ScanResult{
			Output: string(output),
			Error:  fmt.Errorf("failed to execute shcheck: %w", err),
		}
	}

	return tools.ScanResult{
		Output: string(output),
		Error:  nil,
	}
}

// Register registers the shcheck tool with the MCP server.
func (t *Tool) Register(srv *server.Server) error {
	return t.RegisterTool(srv, t.Handler)
}

// Handler handles MCP tool requests.
func (t *Tool) Handler(ctx context.Context, _ *mcp.CallToolRequest, input tools.ScannerInput) (*mcp.CallToolResult, any, error) {
	if err := t.ValidateInput(input); err != nil {
		return nil, nil, err
	}

	host, port := t.ResolveHostPort(input.Host, input.Port)

	params := tools.ScanParams{
		Host:  host,
		Port:  port,
		Vhost: input.Vhost,
	}

	scanResult := t.Scan(ctx, params)
	if scanResult.Error != nil {
		return nil, nil, fmt.Errorf("%w\nOutput: %s", scanResult.Error, scanResult.Output)
	}

	targetURL := tools.BuildTargetURL(host, port)
	resultText := tools.FormatScannerOutput(binaryName, headerVerb, targetURL, scanResult.Output, input.MaxLines, input.Offset)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}

// New creates a new shcheck scanner tool.
func New(logger zerolog.Logger) tools.Scanner {
	return &Tool{
		BaseScanner: tools.NewBaseScanner(binaryName, description, logger),
	}
}
