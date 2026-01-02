package nikto

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
	"github.com/tb0hdan/wass-mcp/pkg/types"
)

const (
	binaryName  = "nikto"
	defaultHost = "localhost"
	defaultPort = 80
)

// Input defines the MCP tool input parameters.
type Input struct {
	Vhost    string `json:"vhost,omitempty"`
	Host     string `json:"host,omitempty" validate:"omitempty,hostname|ip"`
	Port     int    `json:"port,omitempty" validate:"min=0,max=65535"`
	MaxLines int    `json:"max_lines,omitempty" validate:"min=0,max=100000"`
	Offset   int    `json:"offset,omitempty" validate:"min=0"`
}

// Tool implements the nikto scanner.
type Tool struct {
	logger    zerolog.Logger
	validator *validator.Validate
}

// Name returns the scanner name.
func (t *Tool) Name() string {
	return binaryName
}

// IsAvailable checks if the nikto binary is available.
func (t *Tool) IsAvailable() bool {
	_, err := exec.LookPath(binaryName)
	return err == nil
}

// Scan performs the nikto scan and returns the output.
func (t *Tool) Scan(ctx context.Context, params tools.ScanParams) tools.ScanResult {
	host := params.Host
	if host == "" {
		host = defaultHost
	}

	port := params.Port
	if port == 0 {
		port = defaultPort
	}

	targetURL := "http://" + net.JoinHostPort(host, strconv.Itoa(port))
	t.logger.Info().Msgf("Running nikto scan on %s", targetURL)

	args := []string{"-host", host, "-port", fmt.Sprint(port)}
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
	if !t.IsAvailable() {
		return fmt.Errorf("%s binary not found", binaryName)
	}

	t.logger.Debug().Msgf("%s binary found", binaryName)

	tool := &mcp.Tool{
		Name:        binaryName,
		Description: "Nikto is an open source web server scanner.",
	}

	wrappedHandler := tools.WrapToolHandler(
		srv.Storage(),
		binaryName,
		t.NiktoHandler,
	)

	mcp.AddTool(&srv.Server, tool, wrappedHandler)
	t.logger.Debug().Msg("nikto tool registered")

	return nil
}

// NiktoHandler handles MCP tool requests.
func (t *Tool) NiktoHandler(ctx context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, any, error) {
	if err := t.validator.Struct(input); err != nil {
		return nil, nil, fmt.Errorf("validation error: %w", err)
	}

	host := defaultHost
	if input.Host != "" {
		host = input.Host
	}

	port := defaultPort
	if input.Port != 0 {
		port = input.Port
	}

	// Perform the scan using the reusable Scan method.
	params := tools.ScanParams{
		Host:  host,
		Port:  port,
		Vhost: input.Vhost,
	}

	scanResult := t.Scan(ctx, params)
	if scanResult.Error != nil {
		return nil, nil, fmt.Errorf("%w\nOutput: %s", scanResult.Error, scanResult.Output)
	}

	// Apply pagination.
	targetURL := "http://" + net.JoinHostPort(host, strconv.Itoa(port))
	resultText := t.formatOutput(targetURL, scanResult.Output, input.MaxLines, input.Offset)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}

// formatOutput applies pagination and formats the output.
func (t *Tool) formatOutput(targetURL, output string, maxLines, offset int) string {
	if maxLines == 0 {
		maxLines = types.MaxDefaultLines
	}

	lines := strings.Split(output, "\n")
	totalLines := len(lines)

	truncated := false
	if offset > 0 && offset < totalLines {
		end := totalLines
		if offset+maxLines < totalLines {
			end = offset + maxLines
			truncated = true
		}
		lines = lines[offset:end]
	} else if totalLines > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	paginatedOutput := strings.Join(lines, "\n")

	resultText := fmt.Sprintf("nikto output for %s:\n", targetURL)
	if truncated || offset > 0 {
		resultText += fmt.Sprintf("[Showing lines %d-%d of approximately %d lines. Use offset parameter to view more.]\n", offset+1, offset+len(lines), totalLines)
	}
	resultText += "\n" + strings.TrimSpace(paginatedOutput)

	return resultText
}

// New creates a new nikto scanner tool.
func New(logger zerolog.Logger) tools.Scanner {
	return &Tool{
		logger:    logger.With().Str("tool", binaryName).Logger(),
		validator: validator.New(),
	}
}
