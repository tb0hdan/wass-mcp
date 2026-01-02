package wapiti

import (
	"context"
	"fmt"
	"net"
	"os"
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

type Input struct {
	Vhost    string `json:"vhost,omitempty"` // Virtual host to target
	Host     string `json:"host,omitempty" validate:"omitempty,hostname|ip"`
	Port     int    `json:"port,omitempty" validate:"min=0,max=65535"`
	MaxLines int    `json:"max_lines,omitempty" validate:"min=0,max=100000"`
	Offset   int    `json:"offset,omitempty" validate:"min=0"`
}

type Tool struct {
	logger    zerolog.Logger
	validator *validator.Validate
}

func (p *Tool) Register(srv *server.Server) error {
	// Check if wapiti binary exists
	wapitiPath, err := exec.LookPath("wapiti")
	if err != nil {
		return fmt.Errorf("wapiti binary not found: %w", err)
	}
	p.logger.Debug().Msgf("wapiti binary found at %s", wapitiPath)

	tool := &mcp.Tool{
		Name:        "wapiti",
		Description: "Wapiti is a web application vulnerability scanner.",
	}

	// Wrap handler with execution logging
	wrappedHandler := tools.WrapToolHandler(
		srv.Storage(),
		"wapiti",
		p.WapitiHandler,
	)

	mcp.AddTool(&srv.Server, tool, wrappedHandler)
	p.logger.Debug().Msg("wapiti tool registered")

	return nil
}

func (p *Tool) WapitiHandler(ctx context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, any, error) {
	// Validate input using validator.
	if err := p.validator.Struct(input); err != nil {
		return nil, nil, fmt.Errorf("validation error: %w", err)
	}

	host := "localhost"
	if input.Host != "" {
		host = input.Host
	}

	port := 80
	if input.Port != 0 {
		port = input.Port
	}

	targetURL := "http://" + net.JoinHostPort(host, strconv.Itoa(port))

	// Determine max lines for pagination.
	maxLines := types.MaxDefaultLines
	if input.MaxLines > 0 {
		maxLines = input.MaxLines
	}

	// Create temp file for report output.
	tempFile, err := os.CreateTemp("", "wapiti-report-*.txt")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	reportPath := tempFile.Name()
	_ = tempFile.Close() // Close so wapiti can write to it.
	defer func() {
		if err := os.Remove(reportPath); err != nil {
			p.logger.Warn().Err(err).Msg("Failed to clean up temp file")
		}
	}()

	// Execute wapiti scan.
	p.logger.Info().Msgf("Running wapiti scan on %s", targetURL)
	args := []string{"-u", targetURL, "-f", "txt", "-o", reportPath, "--flush-session"}
	if input.Vhost != "" {
		args = append(args, "-H", fmt.Sprintf("Host: %s", input.Vhost))
	}
	cmd := exec.CommandContext(ctx, "wapiti", args...) //nolint:gosec

	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute wapiti: %w\nOutput: %s", err, string(cmdOutput))
	}

	// Read the generated report file.
	reportData, err := os.ReadFile(reportPath) //nolint:gosec // Path is from controlled temp file
	var reportContent string
	if err != nil {
		// Fall back to command output if report not found.
		p.logger.Warn().Err(err).Msg("Failed to read report file, using command output")
		reportContent = string(cmdOutput)
	} else {
		reportContent = string(reportData)
	}

	// Apply pagination.
	lines := strings.Split(reportContent, "\n")
	totalLines := len(lines)

	offset := 0
	if input.Offset > 0 {
		offset = input.Offset
	}

	// Apply offset if needed.
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

	resultText := fmt.Sprintf("wapiti report for %s:\n", targetURL)
	if truncated || offset > 0 {
		resultText += fmt.Sprintf("[Showing lines %d-%d of %d lines. Use offset parameter to view more.]\n", offset+1, offset+len(lines), totalLines)
	}
	resultText += "\n" + strings.TrimSpace(paginatedOutput)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: resultText,
			},
		},
	}

	return result, nil, nil
}

func New(logger zerolog.Logger) tools.Tool {
	validate := validator.New()

	return &Tool{
		logger:    logger.With().Str("tool", "wapiti").Logger(),
		validator: validate,
	}
}
