package nikto

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
	"github.com/tb0hdan/wass-mcp/pkg/types"
)

const (
	binaryName  = "nikto"
	description = "Nikto is an open source web server scanner."
	headerVerb  = "output"
)

// Tool implements the nikto scanner.
type Tool struct {
	tools.BaseScanner
}

// Scan performs the nikto scan and returns the output.
func (t *Tool) Scan(ctx context.Context, params tools.ScanParams) tools.ScanResult {
	targetURL := tools.BuildTargetURL(params)
	t.Logger.Info().Msgf("Running nikto scan on %s", targetURL)

	args := []string{"-host", params.Host, "-port", fmt.Sprint(params.Port)}
	if params.Scheme == types.SchemeHTTPS {
		args = append(args, "-ssl")
	}
	if params.Vhost != "" {
		args = append(args, "-vhost", params.Vhost)
	}

	cmd := exec.CommandContext(ctx, binaryName, args...) //nolint:gosec
	output, err := cmd.CombinedOutput()

	if err != nil {
		return tools.ScanResult{
			Output: string(output),
			Error:  fmt.Errorf("failed to execute nikto: %w", err),
		}
	}

	return tools.ScanResult{
		Output: string(output),
		Error:  nil,
	}
}

// Register registers the nikto tool with the MCP server.
func (t *Tool) Register(srv *server.Server) error {
	return t.RegisterTool(srv, t.Handler)
}

// Handler handles MCP tool requests.
func (t *Tool) Handler(ctx context.Context, _ *mcp.CallToolRequest, input tools.ScannerInput) (*mcp.CallToolResult, any, error) {
	input = t.PrepareInput(input)

	if err := t.ValidateInput(input); err != nil {
		return nil, nil, err
	}

	params := t.ResolveInput(input)

	scanResult := t.Scan(ctx, params)
	if scanResult.Error != nil {
		return nil, nil, fmt.Errorf("%w\nOutput: %s", scanResult.Error, scanResult.Output)
	}

	targetURL := tools.BuildTargetURL(params)
	resultText := tools.FormatScannerOutput(binaryName, headerVerb, targetURL, scanResult.Output, input.MaxLines, input.Offset)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}

// New creates a new nikto scanner tool.
func New(logger zerolog.Logger) tools.Scanner {
	return &Tool{
		BaseScanner: tools.NewBaseScanner(binaryName, description, logger),
	}
}
