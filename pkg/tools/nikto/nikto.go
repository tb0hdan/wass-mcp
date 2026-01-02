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

type Input struct {
	Vhost    string `json:"vhost,omitempty"` // Virtual host to target
	Host     string `json:"host,omitempty" validate:"omitempty,hostname|ip"`
	Port     int    `json:"port,omitempty" validate:"min=0,max=65535"`
	MaxLines int    `json:"max_lines,omitempty" validate:"min=0,max=100000"` // Maximum lines to return (default: 100 for top view)
	Offset   int    `json:"offset,omitempty" validate:"min=0"`               // Line offset for pagination
}

type Tool struct {
	logger    zerolog.Logger
	validator *validator.Validate
}

func (p *Tool) Register(srv *server.Server) error {
	// Check if nikto binary exists
	niktoPath, err := exec.LookPath("nikto")
	if err != nil {
		return fmt.Errorf("nikto binary not found: %w", err)
	}
	p.logger.Debug().Msgf("nikto binary found at %s", niktoPath)

	tool := &mcp.Tool{
		Name:        "nikto",
		Description: "Nikto is an open source web server scanner.",
	}

	// Wrap handler with execution logging
	wrappedHandler := tools.WrapToolHandler(
		srv.Storage(),
		"nikto",
		p.NiktoHandler,
	)

	mcp.AddTool(&srv.Server, tool, wrappedHandler)
	p.logger.Debug().Msg("nikto tool registered")

	return nil
}

func (p *Tool) NiktoHandler(ctx context.Context, _ *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, any, error) {
	// Validate input using validator
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

	profileURL := "http://" + net.JoinHostPort(host, strconv.Itoa(port))

	// Determine max lines for pagination (default: 100 for top view)
	maxLines := types.MaxDefaultLines
	if input.MaxLines > 0 {
		maxLines = input.MaxLines
	}

	// Execute nikto scan
	p.logger.Info().Msgf("Running nikto scan on %s", profileURL)
	args := []string{"-host", host, "-port", fmt.Sprint(port)}
	if input.Vhost != "" {
		args = append(args, "-vhost", input.Vhost)
	}
	cmd := exec.CommandContext(ctx, "nikto", args...) //nolint:gosec

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute nikto: %w\nOutput: %s", err, string(output))
	}

	// Apply pagination
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	totalLines := len(lines)

	offset := 0
	if input.Offset > 0 {
		offset = input.Offset
	}

	// Apply offset if needed
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

	resultText := fmt.Sprintf("nikto output for %s:\n", profileURL)
	if truncated || offset > 0 {
		resultText += fmt.Sprintf("[Showing lines %d-%d of approximately %d lines. Use offset parameter to view more.]\n", offset+1, offset+len(lines), totalLines)
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
		logger:    logger.With().Str("tool", "nikto").Logger(),
		validator: validate,
	}
}
