package fullscan

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/storage"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
)

// mockScanner is a mock implementation of tools.Scanner for testing.
type mockScanner struct {
	name       string
	available  bool
	scanOutput string
	scanError  error
	scanDelay  time.Duration
	scanCalled bool
	scanParams tools.ScanParams
}

func (m *mockScanner) Name() string {
	return m.name
}

func (m *mockScanner) IsAvailable() bool {
	return m.available
}

func (m *mockScanner) Scan(ctx context.Context, params tools.ScanParams) tools.ScanResult {
	m.scanCalled = true
	m.scanParams = params

	if m.scanDelay > 0 {
		time.Sleep(m.scanDelay)
	}

	return tools.ScanResult{
		Output: m.scanOutput,
		Error:  m.scanError,
	}
}

func (m *mockScanner) Register(_ *server.Server) error {
	if !m.available {
		return errors.New("scanner not available")
	}
	return nil
}

type FullScanTestSuite struct {
	suite.Suite
	logger zerolog.Logger
}

func (s *FullScanTestSuite) SetupTest() {
	s.logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
}

func (s *FullScanTestSuite) TestNew() {
	scanner1 := &mockScanner{name: "mock1", available: true}
	scanner2 := &mockScanner{name: "mock2", available: true}

	tool := New(s.logger, scanner1, scanner2)
	s.NotNil(tool)
}

func (s *FullScanTestSuite) TestNew_NoScanners() {
	tool := New(s.logger)
	s.NotNil(tool)
}

func (s *FullScanTestSuite) TestRunScannersParallel_SingleScanner() {
	scanner := &mockScanner{
		name:       "mock1",
		available:  true,
		scanOutput: "test output",
	}

	tool := New(s.logger, scanner).(*Tool)
	tool.scanners = []tools.Scanner{scanner}

	ctx := context.Background()
	params := tools.ScanParams{
		Host:  "localhost",
		Port:  80,
		Vhost: "",
	}

	results := tool.runScannersParallel(ctx, params)

	s.Len(results, 1)
	s.Equal("mock1", results[0].Name)
	s.Equal("test output", results[0].Output)
	s.Nil(results[0].Error)
	s.True(scanner.scanCalled)
}

func (s *FullScanTestSuite) TestRunScannersParallel_MultipleScanners() {
	scanner1 := &mockScanner{
		name:       "mock1",
		available:  true,
		scanOutput: "output1",
	}
	scanner2 := &mockScanner{
		name:       "mock2",
		available:  true,
		scanOutput: "output2",
	}

	tool := New(s.logger, scanner1, scanner2).(*Tool)
	tool.scanners = []tools.Scanner{scanner1, scanner2}

	ctx := context.Background()
	params := tools.ScanParams{
		Host:  "example.com",
		Port:  8080,
		Vhost: "test.example.com",
	}

	results := tool.runScannersParallel(ctx, params)

	s.Len(results, 2)
	s.True(scanner1.scanCalled)
	s.True(scanner2.scanCalled)

	// Verify params were passed correctly
	s.Equal("example.com", scanner1.scanParams.Host)
	s.Equal(8080, scanner1.scanParams.Port)
	s.Equal("test.example.com", scanner1.scanParams.Vhost)
}

func (s *FullScanTestSuite) TestRunScannersParallel_WithError() {
	scanner := &mockScanner{
		name:       "mock1",
		available:  true,
		scanOutput: "partial output",
		scanError:  errors.New("scan failed"),
	}

	tool := New(s.logger, scanner).(*Tool)
	tool.scanners = []tools.Scanner{scanner}

	ctx := context.Background()
	params := tools.ScanParams{Host: "localhost", Port: 80}

	results := tool.runScannersParallel(ctx, params)

	s.Len(results, 1)
	s.Equal("mock1", results[0].Name)
	s.Equal("partial output", results[0].Output)
	s.NotNil(results[0].Error)
	s.Contains(results[0].Error.Error(), "scan failed")
}

func (s *FullScanTestSuite) TestRunScannersParallel_Concurrent() {
	// Test that scanners actually run in parallel
	scanner1 := &mockScanner{
		name:       "mock1",
		available:  true,
		scanOutput: "output1",
		scanDelay:  50 * time.Millisecond,
	}
	scanner2 := &mockScanner{
		name:       "mock2",
		available:  true,
		scanOutput: "output2",
		scanDelay:  50 * time.Millisecond,
	}

	tool := New(s.logger, scanner1, scanner2).(*Tool)
	tool.scanners = []tools.Scanner{scanner1, scanner2}

	ctx := context.Background()
	params := tools.ScanParams{Host: "localhost", Port: 80}

	start := time.Now()
	results := tool.runScannersParallel(ctx, params)
	duration := time.Since(start)

	s.Len(results, 2)
	// If running in parallel, total time should be less than 100ms (50ms + 50ms)
	// Allow some buffer for test environment
	s.Less(duration, 150*time.Millisecond)
}

func (s *FullScanTestSuite) TestMergeResults_Success() {
	tool := New(s.logger).(*Tool)

	results := []scannerResult{
		{
			Name:     "scanner1",
			Output:   "findings from scanner1",
			Duration: 1 * time.Second,
			Error:    nil,
		},
		{
			Name:     "scanner2",
			Output:   "findings from scanner2",
			Duration: 2 * time.Second,
			Error:    nil,
		},
	}

	merged := tool.mergeResults("http://localhost:80", results)

	s.Contains(merged, "FULL SECURITY SCAN REPORT")
	s.Contains(merged, "Target: http://localhost:80")
	s.Contains(merged, "scanner1")
	s.Contains(merged, "scanner2")
	s.Contains(merged, "findings from scanner1")
	s.Contains(merged, "findings from scanner2")
	s.Contains(merged, "SUCCESS")
	s.Contains(merged, "Total scanners: 2")
	s.Contains(merged, "Successful: 2")
	s.Contains(merged, "Failed: 0")
	s.Contains(merged, "END OF REPORT")
}

func (s *FullScanTestSuite) TestMergeResults_WithFailure() {
	tool := New(s.logger).(*Tool)

	results := []scannerResult{
		{
			Name:     "scanner1",
			Output:   "findings from scanner1",
			Duration: 1 * time.Second,
			Error:    nil,
		},
		{
			Name:     "scanner2",
			Output:   "partial output",
			Duration: 500 * time.Millisecond,
			Error:    errors.New("connection timeout"),
		},
	}

	merged := tool.mergeResults("http://localhost:80", results)

	s.Contains(merged, "FULL SECURITY SCAN REPORT")
	s.Contains(merged, "scanner1")
	s.Contains(merged, "scanner2")
	s.Contains(merged, "SUCCESS")
	s.Contains(merged, "FAILED")
	s.Contains(merged, "connection timeout")
	s.Contains(merged, "Successful: 1")
	s.Contains(merged, "Failed: 1")
}

func (s *FullScanTestSuite) TestMergeResults_Empty() {
	tool := New(s.logger).(*Tool)

	results := []scannerResult{}

	merged := tool.mergeResults("http://localhost:80", results)

	s.Contains(merged, "FULL SECURITY SCAN REPORT")
	s.Contains(merged, "Total scanners: 0")
}

func (s *FullScanTestSuite) TestApplyPagination_NoTruncation() {
	tool := New(s.logger).(*Tool)

	output := "line1\nline2\nline3"
	result := tool.applyPagination(output, 0, 0)

	s.Contains(result, "line1")
	s.Contains(result, "line2")
	s.Contains(result, "line3")
	s.NotContains(result, "Showing lines")
}

func (s *FullScanTestSuite) TestApplyPagination_WithTruncation() {
	tool := New(s.logger).(*Tool)

	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line"+string(rune('0'+i%10)))
	}
	output := strings.Join(lines, "\n")

	result := tool.applyPagination(output, 10, 0)

	s.Contains(result, "Showing lines 1-10 of 100 lines")
}

func (s *FullScanTestSuite) TestApplyPagination_WithOffset() {
	tool := New(s.logger).(*Tool)

	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "line"+string(rune('A'+i%26)))
	}
	output := strings.Join(lines, "\n")

	result := tool.applyPagination(output, 10, 20)

	s.Contains(result, "Showing lines 21-30 of 50 lines")
}

func (s *FullScanTestSuite) TestApplyPagination_OffsetBeyondEnd() {
	tool := New(s.logger).(*Tool)

	output := "line1\nline2\nline3"
	result := tool.applyPagination(output, 10, 100)

	// When offset is beyond totalLines, output should still be returned
	s.NotEmpty(result)
}

func (s *FullScanTestSuite) TestInput_Validation() {
	tool := New(s.logger).(*Tool)

	// Test valid input
	input := Input{
		Host: "192.168.1.1",
		Port: 8080,
	}
	err := tool.validator.Struct(input)
	s.NoError(err)
}

func (s *FullScanTestSuite) TestInput_ValidationInvalidHost() {
	tool := New(s.logger).(*Tool)

	input := Input{
		Host: "not a valid host!!!",
		Port: 80,
	}
	err := tool.validator.Struct(input)
	s.Error(err)
}

func (s *FullScanTestSuite) TestInput_ValidationInvalidPort() {
	tool := New(s.logger).(*Tool)

	input := Input{
		Host: "localhost",
		Port: 70000, // Invalid port
	}
	err := tool.validator.Struct(input)
	s.Error(err)
}

func (s *FullScanTestSuite) TestInput_ValidationEmptyHost() {
	tool := New(s.logger).(*Tool)

	// Empty host should be valid (uses default)
	input := Input{
		Port: 80,
	}
	err := tool.validator.Struct(input)
	s.NoError(err)
}

func (s *FullScanTestSuite) TestInput_ValidationWithVhost() {
	tool := New(s.logger).(*Tool)

	input := Input{
		Host:  "192.168.1.1",
		Port:  80,
		Vhost: "example.com",
	}
	err := tool.validator.Struct(input)
	s.NoError(err)
}

func (s *FullScanTestSuite) TestInput_ValidationMaxLinesExceeded() {
	tool := New(s.logger).(*Tool)

	input := Input{
		Host:     "localhost",
		Port:     80,
		MaxLines: 200000, // Exceeds max of 100000
	}
	err := tool.validator.Struct(input)
	s.Error(err)
}

func (s *FullScanTestSuite) setupTestServer() (*server.Server, func()) {
	tmpFile, err := os.CreateTemp("", "fullscan-test-*.db")
	s.Require().NoError(err)
	tmpFile.Close()

	cfg := storage.Config{
		DatabasePath: tmpFile.Name(),
		Debug:        false,
	}

	store, err := storage.NewSQLiteStorage(cfg)
	s.Require().NoError(err)

	impl := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}

	srv := server.NewServer(impl, store)

	cleanup := func() {
		srv.Shutdown(context.Background())
		os.Remove(tmpFile.Name())
	}

	return srv, cleanup
}

func (s *FullScanTestSuite) TestRegister_NoScannersAvailable() {
	scanner1 := &mockScanner{name: "mock1", available: false}
	scanner2 := &mockScanner{name: "mock2", available: false}

	tool := New(s.logger, scanner1, scanner2).(*Tool)

	srv, cleanup := s.setupTestServer()
	defer cleanup()

	// Register should fail when no scanners are available
	err := tool.Register(srv)
	s.Error(err)
	s.Contains(err.Error(), "no scanner binaries available")
}

func (s *FullScanTestSuite) TestRegister_SomeScannersAvailable() {
	scanner1 := &mockScanner{name: "mock1", available: true}
	scanner2 := &mockScanner{name: "mock2", available: false}
	scanner3 := &mockScanner{name: "mock3", available: true}

	tool := New(s.logger, scanner1, scanner2, scanner3).(*Tool)

	srv, cleanup := s.setupTestServer()
	defer cleanup()

	// Register should succeed with at least one available scanner
	err := tool.Register(srv)
	s.NoError(err)

	// Verify only available scanners are kept
	s.Len(tool.scanners, 2)
}

func (s *FullScanTestSuite) TestRegister_AllScannersAvailable() {
	scanner1 := &mockScanner{name: "mock1", available: true}
	scanner2 := &mockScanner{name: "mock2", available: true}

	tool := New(s.logger, scanner1, scanner2).(*Tool)

	srv, cleanup := s.setupTestServer()
	defer cleanup()

	// Register should succeed
	err := tool.Register(srv)
	s.NoError(err)

	// All scanners should be kept
	s.Len(tool.scanners, 2)
}

func (s *FullScanTestSuite) TestFullScanHandler_ValidationError() {
	scanner := &mockScanner{name: "mock1", available: true, scanOutput: "test"}
	tool := New(s.logger, scanner).(*Tool)
	tool.scanners = []tools.Scanner{scanner}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host: "invalid host!!!",
		Port: 80,
	}

	result, output, err := tool.FullScanHandler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *FullScanTestSuite) TestFullScanHandler_ValidationErrorInvalidPort() {
	scanner := &mockScanner{name: "mock1", available: true, scanOutput: "test"}
	tool := New(s.logger, scanner).(*Tool)
	tool.scanners = []tools.Scanner{scanner}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host: "localhost",
		Port: 70000,
	}

	result, output, err := tool.FullScanHandler(ctx, req, input)
	s.Nil(result)
	s.Nil(output)
	s.Error(err)
	s.Contains(err.Error(), "validation error")
}

func (s *FullScanTestSuite) TestFullScanHandler_Success() {
	scanner1 := &mockScanner{name: "scanner1", available: true, scanOutput: "findings from scanner1"}
	scanner2 := &mockScanner{name: "scanner2", available: true, scanOutput: "findings from scanner2"}

	tool := New(s.logger, scanner1, scanner2).(*Tool)
	tool.scanners = []tools.Scanner{scanner1, scanner2}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host: "192.168.1.1",
		Port: 8080,
	}

	result, _, err := tool.FullScanHandler(ctx, req, input)
	s.NoError(err)
	s.NotNil(result)
	s.Len(result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	s.Contains(textContent.Text, "FULL SECURITY SCAN REPORT")
	s.Contains(textContent.Text, "scanner1")
	s.Contains(textContent.Text, "scanner2")
	s.Contains(textContent.Text, "findings from scanner1")
	s.Contains(textContent.Text, "findings from scanner2")
}

func (s *FullScanTestSuite) TestFullScanHandler_DefaultsApplied() {
	scanner := &mockScanner{name: "mock1", available: true, scanOutput: "test output"}
	tool := New(s.logger, scanner).(*Tool)
	tool.scanners = []tools.Scanner{scanner}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{} // All defaults

	result, _, err := tool.FullScanHandler(ctx, req, input)
	s.NoError(err)
	s.NotNil(result)

	// Verify the scanner was called with defaults
	s.True(scanner.scanCalled)
	s.Equal("localhost", scanner.scanParams.Host)
	s.Equal(80, scanner.scanParams.Port)
}

func (s *FullScanTestSuite) TestFullScanHandler_WithPagination() {
	// Create scanner that returns many lines
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "line")
	}
	output := strings.Join(lines, "\n")

	scanner := &mockScanner{name: "mock1", available: true, scanOutput: output}
	tool := New(s.logger, scanner).(*Tool)
	tool.scanners = []tools.Scanner{scanner}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host:     "localhost",
		Port:     80,
		MaxLines: 50,
		Offset:   10,
	}

	result, _, err := tool.FullScanHandler(ctx, req, input)
	s.NoError(err)
	s.NotNil(result)

	textContent := result.Content[0].(*mcp.TextContent)
	s.Contains(textContent.Text, "Showing lines")
}

func (s *FullScanTestSuite) TestFullScanHandler_WithVhost() {
	scanner := &mockScanner{name: "mock1", available: true, scanOutput: "test"}
	tool := New(s.logger, scanner).(*Tool)
	tool.scanners = []tools.Scanner{scanner}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{
		Host:  "192.168.1.1",
		Port:  8080,
		Vhost: "example.com",
	}

	result, _, err := tool.FullScanHandler(ctx, req, input)
	s.NoError(err)
	s.NotNil(result)

	// Verify vhost was passed to scanner
	s.Equal("example.com", scanner.scanParams.Vhost)
}

func (s *FullScanTestSuite) TestFullScanHandler_WithScannerError() {
	scanner := &mockScanner{
		name:       "mock1",
		available:  true,
		scanOutput: "partial output",
		scanError:  errors.New("scan failed"),
	}
	tool := New(s.logger, scanner).(*Tool)
	tool.scanners = []tools.Scanner{scanner}

	ctx := context.Background()
	req := &mcp.CallToolRequest{}
	input := Input{Host: "localhost", Port: 80}

	// Handler should still return results even if scanner fails
	result, _, err := tool.FullScanHandler(ctx, req, input)
	s.NoError(err)
	s.NotNil(result)

	textContent := result.Content[0].(*mcp.TextContent)
	s.Contains(textContent.Text, "FAILED")
	s.Contains(textContent.Text, "scan failed")
}

func TestFullScanTestSuite(t *testing.T) {
	suite.Run(t, new(FullScanTestSuite))
}
