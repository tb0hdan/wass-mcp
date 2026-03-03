package shcheck

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
)

// scanTestTimeout is a short timeout for tests that invoke the actual scanner.
// This ensures tests don't hang when the binary is available but scans take too long.
const scanTestTimeout = 1 * time.Second

type ShcheckTestSuite struct {
	suite.Suite
	logger zerolog.Logger
	tool   *Tool
}

func (s *ShcheckTestSuite) SetupTest() {
	s.logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	scanner := New(s.logger)
	s.tool = scanner.(*Tool)
}

func (s *ShcheckTestSuite) TestNew() {
	scanner := New(s.logger)
	s.NotNil(scanner)
	s.Implements((*interface{ Name() string })(nil), scanner)
}

func (s *ShcheckTestSuite) TestName() {
	s.Equal("shcheck.py", s.tool.Name())
}

func (s *ShcheckTestSuite) TestIsAvailable() {
	// This test just ensures IsAvailable doesn't panic.
	// It may return true or false depending on if shcheck is installed.
	result := s.tool.IsAvailable()
	s.IsType(true, result)
}

func (s *ShcheckTestSuite) TestFormatScannerOutput_NoTruncation() {
	output := "line1\nline2\nline3"
	result := tools.FormatScannerOutput("shcheck.py", "output", "http://localhost:80", output, 0, 0)

	s.Contains(result, "shcheck.py output for http://localhost:80:")
	s.Contains(result, "line1")
	s.Contains(result, "line2")
	s.Contains(result, "line3")
	s.NotContains(result, "Showing lines")
}

func (s *ShcheckTestSuite) TestFormatScannerOutput_WithTruncation() {
	// Create output with more lines than maxLines.
	var lines []string
	for i := range 100 {
		lines = append(lines, "line"+string(rune('0'+i%10)))
	}
	output := strings.Join(lines, "\n")

	result := tools.FormatScannerOutput("shcheck.py", "output", "http://localhost:80", output, 10, 0)

	s.Contains(result, "shcheck.py output for http://localhost:80:")
	s.Contains(result, "Showing lines 1-10 of 100 lines")
}

func (s *ShcheckTestSuite) TestFormatScannerOutput_WithOffset() {
	var lines []string
	for i := range 50 {
		lines = append(lines, "line"+string(rune('A'+i%26)))
	}
	output := strings.Join(lines, "\n")

	result := tools.FormatScannerOutput("shcheck.py", "output", "http://localhost:80", output, 10, 20)

	s.Contains(result, "Showing lines 21-30 of 50 lines")
}

func (s *ShcheckTestSuite) TestFormatScannerOutput_OffsetBeyondEnd() {
	output := "line1\nline2\nline3"
	result := tools.FormatScannerOutput("shcheck.py", "output", "http://localhost:80", output, 10, 100)

	// When offset is beyond totalLines, the original truncation logic applies.
	s.Contains(result, "shcheck.py output for http://localhost:80:")
}

func (s *ShcheckTestSuite) TestFormatScannerOutput_ZeroMaxLines() {
	// When maxLines is 0, it should use the default.
	output := "line1\nline2\nline3"
	result := tools.FormatScannerOutput("shcheck.py", "output", "http://localhost:80", output, 0, 0)

	s.Contains(result, "line1")
	s.Contains(result, "line2")
	s.Contains(result, "line3")
}

func (s *ShcheckTestSuite) TestScannerInput_Validation() {
	// Test valid input.
	input := tools.ScannerInput{
		Host: "192.168.1.1",
		Port: 8080,
	}
	err := s.tool.Validator.Struct(input)
	s.NoError(err)
}

func (s *ShcheckTestSuite) TestScannerInput_ValidationInvalidHost() {
	input := tools.ScannerInput{
		Host: "not a valid host!!!",
		Port: 80,
	}
	err := s.tool.Validator.Struct(input)
	s.Error(err)
}

func (s *ShcheckTestSuite) TestScannerInput_ValidationInvalidPort() {
	input := tools.ScannerInput{
		Host: "localhost",
		Port: 70000, // Invalid port.
	}
	err := s.tool.Validator.Struct(input)
	s.Error(err)
}

func (s *ShcheckTestSuite) TestScannerInput_ValidationNegativeOffset() {
	input := tools.ScannerInput{
		Host:   "localhost",
		Port:   80,
		Offset: -1,
	}
	err := s.tool.Validator.Struct(input)
	s.Error(err)
}

func (s *ShcheckTestSuite) TestScannerInput_ValidationValidHostname() {
	input := tools.ScannerInput{
		Host: "example.com",
		Port: 443,
	}
	err := s.tool.Validator.Struct(input)
	s.NoError(err)
}

func (s *ShcheckTestSuite) TestScannerInput_ValidationValidIP() {
	input := tools.ScannerInput{
		Host: "10.0.0.1",
		Port: 8080,
	}
	err := s.tool.Validator.Struct(input)
	s.NoError(err)
}

func (s *ShcheckTestSuite) TestScannerInput_ValidationEmptyHost() {
	// Empty host should be valid (uses default).
	input := tools.ScannerInput{
		Port: 80,
	}
	err := s.tool.Validator.Struct(input)
	s.NoError(err)
}

func (s *ShcheckTestSuite) TestHandler_ValidationError() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := tools.ScannerInput{
		Host: "invalid host!!!",
		Port: 80,
	}

	result, output, err := s.tool.Handler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *ShcheckTestSuite) TestHandler_ValidationErrorInvalidPort() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := tools.ScannerInput{
		Host: "localhost",
		Port: 70000,
	}

	result, output, err := s.tool.Handler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *ShcheckTestSuite) TestHandler_ValidationErrorNegativeOffset() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := tools.ScannerInput{
		Host:   "localhost",
		Port:   80,
		Offset: -1,
	}

	result, output, err := s.tool.Handler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *ShcheckTestSuite) TestHandler_ValidationErrorMaxLinesExceeded() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := tools.ScannerInput{
		Host:     "localhost",
		Port:     80,
		MaxLines: 200000,
	}

	result, output, err := s.tool.Handler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *ShcheckTestSuite) TestHandler_DefaultsApplied() {
	// This test verifies defaults are applied when host/port are empty.
	ctx, cancel := context.WithTimeout(context.Background(), scanTestTimeout)
	defer cancel()

	req := &mcp.CallToolRequest{}
	input := tools.ScannerInput{} // All defaults.

	// Validation should pass.
	err := s.tool.Validator.Struct(input)
	s.NoError(err)

	// If shcheck is not available, the handler will fail during Scan
	// but we at least confirm validation succeeds with defaults.
	result, _, err := s.tool.Handler(ctx, req, input)
	// Either succeeds (shcheck available) or fails (shcheck not found or timeout).
	if err != nil {
		// Expected when shcheck is not installed or scan times out.
		s.True(strings.Contains(err.Error(), "shcheck") || strings.Contains(err.Error(), "context"))
	} else {
		s.NotNil(result)
		s.NotEmpty(result.Content)
	}
}

func (s *ShcheckTestSuite) TestHandler_WithVhost() {
	ctx, cancel := context.WithTimeout(context.Background(), scanTestTimeout)
	defer cancel()

	req := &mcp.CallToolRequest{}
	input := tools.ScannerInput{
		Host:  "192.168.1.1",
		Port:  8080,
		Vhost: "example.com",
	}

	// Validation should pass.
	err := s.tool.Validator.Struct(input)
	s.NoError(err)

	// Test handler (will fail if shcheck not installed or times out).
	result, _, err := s.tool.Handler(ctx, req, input)
	if err != nil {
		s.True(strings.Contains(err.Error(), "shcheck") || strings.Contains(err.Error(), "context"))
	} else {
		s.NotNil(result)
	}
}

func (s *ShcheckTestSuite) TestScan_DefaultHost() {
	ctx, cancel := context.WithTimeout(context.Background(), scanTestTimeout)
	defer cancel()

	params := s.tool.Name() // Just to confirm tool is set up.

	s.Equal("shcheck.py", params)

	// Test Scan with empty host - should use default.
	// This will fail if shcheck is not installed or times out, which is expected.
	result := s.tool.Scan(ctx, tools.ScanParams{Host: "", Port: 0, Vhost: ""})

	// If shcheck is not installed or times out, we expect an error.
	if result.Error != nil {
		s.True(strings.Contains(result.Error.Error(), "shcheck") || strings.Contains(result.Error.Error(), "context"))
	}
}

func (s *ShcheckTestSuite) TestScan_WithVhost() {
	ctx, cancel := context.WithTimeout(context.Background(), scanTestTimeout)
	defer cancel()

	// Test Scan with vhost parameter.
	result := s.tool.Scan(ctx, tools.ScanParams{
		Host:  "localhost",
		Port:  8080,
		Vhost: "test.example.com",
	})

	// If shcheck is not installed or times out, we expect an error.
	if result.Error != nil {
		s.True(strings.Contains(result.Error.Error(), "shcheck") || strings.Contains(result.Error.Error(), "context"))
	}
}

func TestShcheckTestSuite(t *testing.T) {
	suite.Run(t, new(ShcheckTestSuite))
}
