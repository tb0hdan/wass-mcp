package nuclei

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
	binaryName  = "nuclei"
	description = "Nuclei is a fast, customizable vulnerability scanner based on YAML templates."
	headerVerb  = "output"
)

// Tool implements the nuclei scanner.
type Tool struct {
	tools.BaseScanner
}

// Scan performs the nuclei scan and returns the output.
func (t *Tool) Scan(ctx context.Context, params tools.ScanParams) tools.ScanResult {
	host, port := t.ResolveHostPort(params.Host, params.Port)

	targetURL := tools.BuildTargetURL(host, port)
	t.Logger.Info().Msgf("Running nuclei scan on %s", targetURL)

	args := []string{"-u", targetURL, "-jsonl"}
	if params.Vhost != "" {
		args = append(args, "-H", fmt.Sprintf("Host: %s", params.Vhost))
	}

	cmd := exec.CommandContext(ctx, binaryName, args...) //nolint:gosec
	output, err := cmd.CombinedOutput()

	if err != nil {
		return tools.ScanResult{
			Output: string(output),
			Error:  fmt.Errorf("failed to execute nuclei: %w", err),
		}
	}

	return tools.ScanResult{
		Output: string(output),
		Error:  nil,
	}
}

// Register registers the nuclei tool with the MCP server.
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

// New creates a new nuclei scanner tool.
func New(logger zerolog.Logger) tools.Scanner {
	return &Tool{
		BaseScanner: tools.NewBaseScanner(binaryName, description, logger),
	}
}
