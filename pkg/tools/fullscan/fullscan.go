package fullscan

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
)

const (
	reportLineWidth = 78
	toolName        = "full_scan"
)

// scannerResult holds the result from a single scanner with timing.
type scannerResult struct {
	Duration time.Duration
	Error    error
	Name     string
	Output   string
}

// Tool implements the full scan tool.
type Tool struct {
	logger    zerolog.Logger
	scanners  []tools.Scanner
	validator *validator.Validate
}

// Register registers the full_scan tool with the MCP server.
func (t *Tool) Register(srv *server.Server) error {
	// Filter to only available scanners.
	var availableScanners []tools.Scanner
	for _, scanner := range t.scanners {
		if scanner.IsAvailable() {
			t.logger.Debug().Msgf("scanner %s is available", scanner.Name())
			availableScanners = append(availableScanners, scanner)
		} else {
			t.logger.Warn().Msgf("scanner %s not available, will be skipped", scanner.Name())
		}
	}

	if len(availableScanners) == 0 {
		return fmt.Errorf("no scanner binaries available")
	}

	t.scanners = availableScanners

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Performs a comprehensive security scan using all available scanners in parallel and merges results.",
	}

	wrappedHandler := tools.WrapToolHandler(
		srv.Storage(),
		toolName,
		t.FullScanHandler,
	)

	mcp.AddTool(&srv.Server, tool, wrappedHandler)
	t.logger.Debug().Msgf("%s tool registered with %d scanners", toolName, len(t.scanners))

	return nil
}

// FullScanHandler handles MCP tool requests.
func (t *Tool) FullScanHandler(ctx context.Context, _ *mcp.CallToolRequest, input tools.ScannerInput) (*mcp.CallToolResult, any, error) {
	// Parse URL-style hosts before validation.
	parsed := tools.ParseHostInput(input.Host)
	input.Host = parsed.Host

	if input.Port == 0 && parsed.Port != 0 {
		input.Port = parsed.Port
	}

	if err := t.validator.Struct(input); err != nil {
		return nil, nil, fmt.Errorf("validation error: %w", err)
	}

	params := tools.ResolveParams(input)
	targetURL := tools.BuildTargetURL(params)
	t.logger.Info().Msgf("Starting full scan on %s with %d scanners", targetURL, len(t.scanners))

	// Run all scanners in parallel.
	results := t.runScannersParallel(ctx, params)

	// Merge results into report.
	mergedOutput := t.mergeResults(targetURL, results)

	// Apply pagination using the shared function.
	resultText := t.applyPagination(mergedOutput, input.MaxLines, input.Offset)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, nil, nil
}

// runScannersParallel runs all scanners in parallel and collects results.
func (t *Tool) runScannersParallel(ctx context.Context, params tools.ScanParams) []scannerResult {
	var waitGroup sync.WaitGroup
	resultsChan := make(chan scannerResult, len(t.scanners))

	for _, scanner := range t.scanners {
		waitGroup.Add(1)
		go func(currentScanner tools.Scanner) {
			defer waitGroup.Done()

			start := time.Now()
			scanResult := currentScanner.Scan(ctx, params)
			duration := time.Since(start)

			resultsChan <- scannerResult{
				Name:     currentScanner.Name(),
				Output:   scanResult.Output,
				Duration: duration,
				Error:    scanResult.Error,
			}
		}(scanner)
	}

	// Wait for all scanners to complete.
	go func() {
		waitGroup.Wait()
		close(resultsChan)
	}()

	// Collect results.
	var results []scannerResult
	for result := range resultsChan {
		results = append(results, result)
		if result.Error != nil {
			t.logger.Warn().Err(result.Error).Msgf("%s scan failed", result.Name)
		} else {
			t.logger.Info().Dur("duration", result.Duration).Msgf("%s scan completed", result.Name)
		}
	}

	return results
}

// mergeResults merges scanner results into a unified report.
func (t *Tool) mergeResults(targetURL string, results []scannerResult) string {
	var builder strings.Builder

	separator := "=" + strings.Repeat("=", reportLineWidth)
	dashLine := "-" + strings.Repeat("-", reportLineWidth)

	builder.WriteString(separator + "\n")
	builder.WriteString("                    FULL SECURITY SCAN REPORT\n")
	builder.WriteString(separator + "\n")
	builder.WriteString(fmt.Sprintf("Target: %s\n", targetURL))
	builder.WriteString(fmt.Sprintf("Date: %s\n", time.Now().UTC().Format(time.RFC1123)))
	builder.WriteString(separator + "\n\n")

	// Summary section.
	builder.WriteString("SCAN SUMMARY\n")
	builder.WriteString(dashLine + "\n")

	var totalDuration time.Duration
	failCount := 0
	successCount := 0

	for _, result := range results {
		totalDuration += result.Duration
		status := "SUCCESS"
		if result.Error != nil {
			status = "FAILED"
			failCount++
		} else {
			successCount++
		}
		builder.WriteString(fmt.Sprintf("  %-10s: %s (%.2fs)\n", result.Name, status, result.Duration.Seconds()))
	}

	builder.WriteString(fmt.Sprintf("\nTotal scanners: %d | Successful: %d | Failed: %d\n", len(results), successCount, failCount))
	builder.WriteString(fmt.Sprintf("Total scan time: %.2fs\n", totalDuration.Seconds()))
	builder.WriteString("\n")

	// Individual scanner results.
	for _, result := range results {
		builder.WriteString(separator + "\n")
		builder.WriteString(fmt.Sprintf("                    %s RESULTS\n", strings.ToUpper(result.Name)))
		builder.WriteString(separator + "\n\n")

		if result.Error != nil {
			builder.WriteString(fmt.Sprintf("ERROR: %s\n\n", result.Error.Error()))
			if result.Output != "" {
				builder.WriteString("Output:\n")
				builder.WriteString(result.Output)
				builder.WriteString("\n")
			}
		} else {
			builder.WriteString(strings.TrimSpace(result.Output))
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}

	builder.WriteString(separator + "\n")
	builder.WriteString("                    END OF REPORT\n")
	builder.WriteString(separator + "\n")

	return builder.String()
}

// applyPagination applies pagination to the output using the shared pagination logic.
func (t *Tool) applyPagination(output string, maxLines, offset int) string {
	pagination := tools.ApplyPagination(output, maxLines, offset)
	paginatedOutput := strings.Join(pagination.Lines, "\n")

	resultText := ""
	if pagination.Truncated || offset > 0 {
		resultText = fmt.Sprintf("[Showing lines %d-%d of %d lines. Use offset parameter to view more.]\n\n",
			pagination.StartLine+1, pagination.EndLine, pagination.TotalLines)
	}
	resultText += paginatedOutput

	return resultText
}

// New creates a new full scan tool with the given scanners.
func New(logger zerolog.Logger, scanners ...tools.Scanner) tools.Tool {
	return &Tool{
		logger:    logger.With().Str("tool", toolName).Logger(),
		scanners:  scanners,
		validator: validator.New(),
	}
}
