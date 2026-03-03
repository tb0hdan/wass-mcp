package tools

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
	"github.com/tb0hdan/wass-mcp/pkg/types"
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
	Error  error
	Output string
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

// ScannerInput defines common MCP tool input parameters for all scanners.
// This eliminates duplicate Input struct definitions across scanner packages.
type ScannerInput struct {
	Host     string `json:"host,omitempty" validate:"omitempty,hostname|ip"`
	MaxLines int    `json:"max_lines,omitempty" validate:"min=0,max=100000"`
	Offset   int    `json:"offset,omitempty" validate:"min=0"`
	Port     int    `json:"port,omitempty" validate:"min=0,max=65535"`
	Vhost    string `json:"vhost,omitempty"`
}

// PaginationResult contains the result of pagination applied to output.
type PaginationResult struct {
	EndLine    int
	Lines      []string
	StartLine  int
	TotalLines int
	Truncated  bool
}

// ApplyPagination applies pagination to the given output string.
// It returns the paginated lines and metadata about the pagination.
func ApplyPagination(output string, maxLines, offset int) PaginationResult {
	if maxLines == 0 {
		maxLines = types.MaxDefaultLines
	}

	lines := strings.Split(output, "\n")
	totalLines := len(lines)

	truncated := false
	startLine := offset

	if offset > 0 && offset < totalLines {
		end := totalLines
		if offset+maxLines < totalLines {
			end = offset + maxLines
			truncated = true
		}
		lines = lines[offset:end]
	} else if totalLines > maxLines {
		lines = lines[:maxLines]
		startLine = 0
		truncated = true
	} else {
		startLine = 0
	}

	return PaginationResult{
		EndLine:    startLine + len(lines),
		Lines:      lines,
		StartLine:  startLine,
		TotalLines: totalLines,
		Truncated:  truncated,
	}
}

// FormatScannerOutput formats scanner output with pagination information.
// toolName is used in the header (e.g., "nikto output for", "wapiti report for").
// headerVerb allows customization (e.g., "output" vs "report").
func FormatScannerOutput(toolName, headerVerb, targetURL, output string, maxLines, offset int) string {
	pagination := ApplyPagination(output, maxLines, offset)
	paginatedOutput := strings.Join(pagination.Lines, "\n")

	resultText := fmt.Sprintf("%s %s for %s:\n", toolName, headerVerb, targetURL)
	if pagination.Truncated || offset > 0 {
		resultText += fmt.Sprintf("[Showing lines %d-%d of %d lines. Use offset parameter to view more.]\n",
			pagination.StartLine+1, pagination.EndLine, pagination.TotalLines)
	}
	resultText += "\n" + strings.TrimSpace(paginatedOutput)

	return resultText
}

// BuildTargetURL constructs a URL from host and port, using HTTPS for port 443.
func BuildTargetURL(host string, port int) string {
	scheme := "http"
	if port == types.HTTPSPort {
		scheme = "https"
	}

	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port))
}

// BaseScanner provides common functionality for scanner tools.
// Embed this struct in concrete scanner implementations to reduce code duplication.
type BaseScanner struct {
	BinaryName  string
	Description string
	Logger      zerolog.Logger
	Validator   *validator.Validate
}

// NewBaseScanner creates a new BaseScanner with the given configuration.
func NewBaseScanner(binaryName, description string, logger zerolog.Logger) BaseScanner {
	return BaseScanner{
		BinaryName:  binaryName,
		Description: description,
		Logger:      logger.With().Str("tool", binaryName).Logger(),
		Validator:   validator.New(),
	}
}

// Name returns the scanner name (binary name).
func (b *BaseScanner) Name() string {
	return b.BinaryName
}

// IsAvailable checks if the scanner binary is available in PATH.
func (b *BaseScanner) IsAvailable() bool {
	_, err := exec.LookPath(b.BinaryName)
	return err == nil
}

// ValidateInput validates the scanner input using the validator.
func (b *BaseScanner) ValidateInput(input any) error {
	if err := b.Validator.Struct(input); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	return nil
}

// ResolveHostPort returns host and port with defaults applied if needed.
func (b *BaseScanner) ResolveHostPort(host string, port int) (string, int) {
	if host == "" {
		host = types.DefaultHost
	}
	if port == 0 {
		port = types.DefaultPort
	}
	return host, port
}

// RegisterTool is a helper to register a scanner tool with the MCP server.
// It handles availability check, tool creation, and handler wrapping.
func (b *BaseScanner) RegisterTool(
	srv *server.Server,
	handler func(context.Context, *mcp.CallToolRequest, ScannerInput) (*mcp.CallToolResult, any, error),
) error {
	if !b.IsAvailable() {
		return fmt.Errorf("%s binary not found", b.BinaryName)
	}

	b.Logger.Debug().Msgf("%s binary found", b.BinaryName)

	tool := &mcp.Tool{
		Name:        b.BinaryName,
		Description: b.Description,
	}

	wrappedHandler := WrapToolHandler(
		srv.Storage(),
		b.BinaryName,
		handler,
	)

	mcp.AddTool(&srv.Server, tool, wrappedHandler)
	b.Logger.Debug().Msgf("%s tool registered", b.BinaryName)

	return nil
}
