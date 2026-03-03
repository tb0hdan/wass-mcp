package wapiti

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
)

const (
	binaryName  = "wapiti"
	description = "Wapiti is a web application vulnerability scanner."
	headerVerb  = "report"
)

// Tool implements the wapiti scanner.
type Tool struct {
	tools.BaseScanner
}

// Scan performs the wapiti scan and returns the output.
func (t *Tool) Scan(ctx context.Context, params tools.ScanParams) tools.ScanResult {
	targetURL := tools.BuildTargetURL(params)
	t.Logger.Info().Msgf("Running wapiti scan on %s", targetURL)

	// Create temp file for report output.
	tempFile, err := os.CreateTemp("", "wapiti-report-*.txt")
	if err != nil {
		return tools.ScanResult{
			Error: fmt.Errorf("failed to create temp file: %w", err),
		}
	}
	reportPath := tempFile.Name()
	_ = tempFile.Close()
	defer func() {
		_ = os.Remove(reportPath)
	}()

	args := []string{"-u", targetURL, "-f", "txt", "-o", reportPath, "--flush-session"}
	if params.Vhost != "" {
		args = append(args, "-H", fmt.Sprintf("Host: %s", params.Vhost))
	}

	cmd := exec.CommandContext(ctx, binaryName, args...) //nolint:gosec
	cmdOutput, err := cmd.CombinedOutput()

	if err != nil {
		return tools.ScanResult{
			Output: string(cmdOutput),
			Error:  fmt.Errorf("failed to execute wapiti: %w", err),
		}
	}

	// Read the generated report file.
	reportData, err := os.ReadFile(reportPath) //nolint:gosec
	if err != nil {
		t.Logger.Warn().Err(err).Msg("Failed to read report file, using command output")
		return tools.ScanResult{
			Output: string(cmdOutput),
			Error:  nil,
		}
	}

	return tools.ScanResult{
		Output: string(reportData),
		Error:  nil,
	}
}

// Register registers the wapiti tool with the MCP server.
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

// New creates a new wapiti scanner tool.
func New(logger zerolog.Logger) tools.Scanner {
	return &Tool{
		BaseScanner: tools.NewBaseScanner(binaryName, description, logger),
	}
}
