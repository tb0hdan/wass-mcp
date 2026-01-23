package wapiti

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
)

type WapitiTestSuite struct {
	suite.Suite
	logger zerolog.Logger
	tool   *Tool
}

func (s *WapitiTestSuite) SetupTest() {
	s.logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	scanner := New(s.logger)
	s.tool = scanner.(*Tool)
}

func (s *WapitiTestSuite) TestNew() {
	scanner := New(s.logger)
	s.NotNil(scanner)
	s.Implements((*interface{ Name() string })(nil), scanner)
}

func (s *WapitiTestSuite) TestName() {
	s.Equal("wapiti", s.tool.Name())
}

func (s *WapitiTestSuite) TestIsAvailable() {
	// This test just ensures IsAvailable doesn't panic
	// It may return true or false depending on if wapiti is installed
	result := s.tool.IsAvailable()
	s.IsType(true, result)
}

func (s *WapitiTestSuite) TestFormatOutput_NoTruncation() {
	output := "line1\nline2\nline3"
	result := s.tool.formatOutput("http://localhost:80", output, 0, 0)

	s.Contains(result, "wapiti report for http://localhost:80:")
	s.Contains(result, "line1")
	s.Contains(result, "line2")
	s.Contains(result, "line3")
	s.NotContains(result, "Showing lines")
}

func (s *WapitiTestSuite) TestFormatOutput_WithTruncation() {
	// Create output with more lines than maxLines
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line"+string(rune('0'+i%10)))
	}
	output := strings.Join(lines, "\n")

	result := s.tool.formatOutput("http://localhost:80", output, 10, 0)

	s.Contains(result, "wapiti report for http://localhost:80:")
	s.Contains(result, "Showing lines 1-10 of 100 lines")
}

func (s *WapitiTestSuite) TestFormatOutput_WithOffset() {
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "line"+string(rune('A'+i%26)))
	}
	output := strings.Join(lines, "\n")

	result := s.tool.formatOutput("http://localhost:80", output, 10, 20)

	s.Contains(result, "Showing lines 21-30 of 50 lines")
}

func (s *WapitiTestSuite) TestFormatOutput_OffsetBeyondEnd() {
	output := "line1\nline2\nline3"
	result := s.tool.formatOutput("http://localhost:80", output, 10, 100)

	// When offset is beyond totalLines, the original truncation logic applies
	s.Contains(result, "wapiti report for http://localhost:80:")
}

func (s *WapitiTestSuite) TestFormatOutput_ZeroMaxLines() {
	// When maxLines is 0, it should use the default
	output := "line1\nline2\nline3"
	result := s.tool.formatOutput("http://localhost:80", output, 0, 0)

	s.Contains(result, "line1")
	s.Contains(result, "line2")
	s.Contains(result, "line3")
}

func (s *WapitiTestSuite) TestInput_Validation() {
	// Test valid input
	input := Input{
		Host: "192.168.1.1",
		Port: 8080,
	}
	err := s.tool.validator.Struct(input)
	s.NoError(err)
}

func (s *WapitiTestSuite) TestInput_ValidationInvalidHost() {
	input := Input{
		Host: "not a valid host!!!",
		Port: 80,
	}
	err := s.tool.validator.Struct(input)
	s.Error(err)
}

func (s *WapitiTestSuite) TestInput_ValidationInvalidPort() {
	input := Input{
		Host: "localhost",
		Port: 70000, // Invalid port
	}
	err := s.tool.validator.Struct(input)
	s.Error(err)
}

func (s *WapitiTestSuite) TestInput_ValidationNegativeOffset() {
	input := Input{
		Host:   "localhost",
		Port:   80,
		Offset: -1,
	}
	err := s.tool.validator.Struct(input)
	s.Error(err)
}

func (s *WapitiTestSuite) TestInput_ValidationValidHostname() {
	input := Input{
		Host: "example.com",
		Port: 443,
	}
	err := s.tool.validator.Struct(input)
	s.NoError(err)
}

func (s *WapitiTestSuite) TestInput_ValidationValidIP() {
	input := Input{
		Host: "10.0.0.1",
		Port: 8080,
	}
	err := s.tool.validator.Struct(input)
	s.NoError(err)
}

func (s *WapitiTestSuite) TestInput_ValidationEmptyHost() {
	// Empty host should be valid (uses default)
	input := Input{
		Port: 80,
	}
	err := s.tool.validator.Struct(input)
	s.NoError(err)
}

func (s *WapitiTestSuite) TestWapitiHandler_ValidationError() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host: "invalid host!!!",
		Port: 80,
	}

	result, output, err := s.tool.WapitiHandler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *WapitiTestSuite) TestWapitiHandler_ValidationErrorInvalidPort() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host: "localhost",
		Port: 70000,
	}

	result, output, err := s.tool.WapitiHandler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *WapitiTestSuite) TestWapitiHandler_ValidationErrorNegativeOffset() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host:   "localhost",
		Port:   80,
		Offset: -1,
	}

	result, output, err := s.tool.WapitiHandler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *WapitiTestSuite) TestWapitiHandler_ValidationErrorMaxLinesExceeded() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host:     "localhost",
		Port:     80,
		MaxLines: 200000,
	}

	result, output, err := s.tool.WapitiHandler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *WapitiTestSuite) TestWapitiHandler_DefaultsApplied() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{}

	// Validation should pass
	err := s.tool.validator.Struct(input)
	s.NoError(err)

	// If wapiti is not available, the handler will fail during Scan
	result, _, err := s.tool.WapitiHandler(ctx, req, input)
	if err != nil {
		s.Contains(err.Error(), "wapiti")
	} else {
		s.NotNil(result)
		s.NotEmpty(result.Content)
	}
}

func (s *WapitiTestSuite) TestWapitiHandler_WithVhost() {
	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host:  "192.168.1.1",
		Port:  8080,
		Vhost: "example.com",
	}

	// Validation should pass
	err := s.tool.validator.Struct(input)
	s.NoError(err)

	// Test handler (will fail if wapiti not installed, but validates input path)
	result, _, err := s.tool.WapitiHandler(ctx, req, input)
	if err != nil {
		s.Contains(err.Error(), "wapiti")
	} else {
		s.NotNil(result)
	}
}

func (s *WapitiTestSuite) TestScan_DefaultHost() {
	ctx := context.Background()

	// Test Scan with empty host - should use default
	result := s.tool.Scan(ctx, tools.ScanParams{Host: "", Port: 0, Vhost: ""})

	// If wapiti is not installed, we expect an error
	if result.Error != nil {
		s.Contains(result.Error.Error(), "wapiti")
	}
}

func (s *WapitiTestSuite) TestScan_WithVhost() {
	ctx := context.Background()

	// Test Scan with vhost parameter
	result := s.tool.Scan(ctx, tools.ScanParams{
		Host:  "localhost",
		Port:  8080,
		Vhost: "test.example.com",
	})

	// If wapiti is not installed, we expect an error
	if result.Error != nil {
		s.Contains(result.Error.Error(), "wapiti")
	}
}

func TestWapitiTestSuite(t *testing.T) {
	suite.Run(t, new(WapitiTestSuite))
}
