# WASS-MCP Project Notes

## Overview

**WASS-MCP** (Web Application Security Scanner MCP) is a Model Context Protocol server that provides web application security scanning tools. It exposes security scanning capabilities through the MCP protocol, allowing AI assistants and other MCP clients to perform authorized security assessments.

## Architecture

### Technology Stack

- **Language:** Go 1.25.5
- **MCP SDK:** github.com/modelcontextprotocol/go-sdk v1.2.0
- **Database:** SQLite via GORM
- **Validation:** go-playground/validator
- **Logging:** zerolog

### Project Structure

```
wass-mcp/
├── cmd/wass-mcp/
│   ├── main.go          # Application entry point
│   └── VERSION          # Version file (embedded)
├── pkg/
│   ├── server/
│   │   ├── server.go    # MCP server wrapper with storage
│   │   └── server_test.go
│   ├── storage/
│   │   ├── storage.go   # Storage interface
│   │   ├── sqlite.go    # SQLite/GORM implementation
│   │   └── sqlite_test.go
│   ├── models/
│   │   ├── tool_execution.go  # Execution history model
│   │   └── tool_execution_test.go
│   ├── tools/
│   │   ├── tools.go     # Tool interface
│   │   ├── wrapper.go   # Execution logging wrapper
│   │   ├── wrapper_test.go
│   │   ├── nikto/
│   │   │   └── nikto.go # Nikto scanner tool
│   │   ├── wapiti/
│   │   │   └── wapiti.go # Wapiti scanner tool
│   │   ├── nuclei/
│   │   │   └── nuclei.go # Nuclei scanner tool
│   │   ├── shcheck/
│   │   │   └── shcheck.go # Security headers checker tool
│   │   ├── fullscan/
│   │   │   └── fullscan.go # Parallel full scan tool
│   │   └── history/
│   │       ├── history.go # History management tool
│   │       └── history_test.go
│   └── types/
│       ├── constants.go # Shared constants
│       └── constants_test.go
├── docs/
│   └── PROJECT_NOTES.md # This file
├── build/               # Build artifacts
├── Makefile             # Build, test, lint commands
├── go.mod               # Go module definition
└── go.sum               # Dependency checksums
```

## Configuration

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--bind` | `localhost:8989` | HTTP bind address |
| `--db` | `./wass-mcp.db` | SQLite database path |
| `--debug` | `false` | Enable debug logging |
| `--version` | - | Print version and exit |

### Environment

The server exposes:
- `/mcp` - MCP protocol endpoint (Streamable HTTP)
- `/` - Service info JSON endpoint
- `/debug/pprof/*` - Profiling endpoints (when pprof enabled)

## Tools

### nikto

Web server vulnerability scanner using Nikto.

**Input:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `host` | string | Target hostname or IP |
| `port` | int | Target port (default: 80) |
| `vhost` | string | Virtual host header (optional) |
| `max_lines` | int | Max output lines (pagination) |
| `offset` | int | Line offset (pagination) |

**Example:**
```json
{"host": "192.168.1.1", "port": 80}
```

### wapiti

Web application vulnerability scanner using Wapiti. Performs comprehensive security testing including SQL injection, XSS, file inclusion, command execution, and more.

**Input:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `host` | string | Target hostname or IP |
| `port` | int | Target port (default: 80) |
| `vhost` | string | Virtual host header (optional) |
| `max_lines` | int | Max output lines (pagination) |
| `offset` | int | Line offset (pagination) |

**Example:**
```json
{"host": "192.168.1.1", "port": 8080}
```

**Output:** Returns formatted vulnerability report including:
- Summary of vulnerabilities by category
- Detailed findings with proof-of-concept requests
- cURL commands for manual verification

### nuclei

Template-based vulnerability scanner using Nuclei. Performs fast scanning using YAML-based templates for CVE detection, misconfigurations, and more.

**Input:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `host` | string | Target hostname or IP |
| `port` | int | Target port (default: 80) |
| `vhost` | string | Virtual host header (optional) |
| `max_lines` | int | Max output lines (pagination) |
| `offset` | int | Line offset (pagination) |

**Example:**
```json
{"host": "192.168.1.1", "port": 443}
```

**Output:** Returns JSON lines output including:
- Template matches with severity levels
- CVE identifiers when applicable
- Detailed finding information
- Affected URLs and parameters

### shcheck

Security headers checker using shcheck.py. Analyzes HTTP response headers for security best practices including Content-Security-Policy, Strict-Transport-Security, X-Frame-Options, X-Content-Type-Options, and more.

**Input:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `host` | string | Target hostname or IP |
| `port` | int | Target port (default: 80) |
| `vhost` | string | Virtual host header (optional) |
| `max_lines` | int | Max output lines (pagination) |
| `offset` | int | Line offset (pagination) |

**Example:**
```json
{"host": "example.com", "port": 443}
```

**Output:** Returns JSON output including:
- Present security headers with their values
- Missing security headers that should be configured
- Deprecated headers detected

### full_scan

Comprehensive security scan using all available scanners in parallel. Merges results into a unified report.

**Input:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `host` | string | Target hostname or IP |
| `port` | int | Target port (default: 80) |
| `vhost` | string | Virtual host header (optional) |
| `max_lines` | int | Max output lines (pagination) |
| `offset` | int | Line offset (pagination) |

**Example:**
```json
{"host": "192.168.1.1", "port": 8080}
```

**Output:** Unified report containing:
- Scan summary with timing for each scanner
- Success/failure status per scanner
- Merged results from all scanners (nikto, wapiti, nuclei, shcheck)

**Features:**
- Runs all available scanners in parallel
- Gracefully handles missing scanner binaries
- Continues if at least one scanner is available

### history

Browse and manage tool execution history.

**Input:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `action` | string | `list`, `get`, `delete`, or `clear` |
| `id` | uint | Execution ID (for get/delete) |
| `limit` | int | Results per page (default: 10, max: 100) |
| `offset` | int | Pagination offset |

**Actions:**
- `list` - Paginated execution history
- `get` - Full execution details by ID
- `delete` - Delete execution by ID
- `clear` - Delete all history

## Database Schema

### tool_executions

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key (auto-increment) |
| `created_at` | timestamp | Execution timestamp |
| `deleted_at` | timestamp | Soft delete timestamp |
| `session_id` | varchar(64) | MCP session identifier |
| `tool_name` | varchar(255) | Tool that was executed |
| `input_json` | text | JSON-serialized input parameters |
| `output_json` | text | JSON-serialized output/results |
| `error_message` | text | Error message if failed |
| `duration_ms` | int64 | Execution time in milliseconds |
| `success` | bool | Whether execution succeeded |

## Key Implementation Details

### Stateless MCP Sessions

The server uses stateless mode (`Stateless: true` in StreamableHTTPOptions) to avoid "session not found" errors after server restarts. Each request is independent.

### Execution Logging

All tool executions are automatically logged via the `WrapToolHandler` generic wrapper:
- Captures input/output as JSON
- Records timing information
- Logs asynchronously to avoid blocking
- Stores session ID for tracking

### Tool Registration Pattern

Tools implement the `tools.Tool` interface:
```go
type Tool interface {
    Register(srv *server.Server) error
}
```

Scanner tools extend the `tools.BaseScanner` struct for common functionality:
```go
type BaseScanner struct {
    BinaryName  string
    Description string
    Logger      zerolog.Logger
    Validator   *validator.Validate
}
```

The `BaseScanner` provides:
- `Name()` - Returns the scanner binary name
- `IsAvailable()` - Checks if binary exists in PATH
- `PrepareInput()` - Parses URL-style hosts and extracts scheme/hostname/port before validation
- `ValidateInput()` - Validates input using go-playground/validator
- `ResolveInput()` - Resolves input to `ScanParams` with scheme, defaults, and port inference
- `RegisterTool()` - Handles common registration logic

### Shared Types

All scanner tools use shared types from `pkg/tools`:
```go
// ScannerInput - Common MCP tool input parameters
type ScannerInput struct {
    Host     string `json:"host,omitempty" validate:"omitempty,hostname|ip"`
    MaxLines int    `json:"max_lines,omitempty" validate:"min=0,max=100000"`
    Offset   int    `json:"offset,omitempty" validate:"min=0"`
    Port     int    `json:"port,omitempty" validate:"min=0,max=65535"`
    Vhost    string `json:"vhost,omitempty"`
}

// ScanParams - Parameters passed to Scan method
type ScanParams struct {
    Host   string
    Port   int
    Scheme string
    Vhost  string
}

// ScanResult - Result returned from Scan method
type ScanResult struct {
    Error  error
    Output string
}
```

### Handler Signature (MCP SDK v1.x)

```go
func Handler(ctx context.Context, req *mcp.CallToolRequest, input tools.ScannerInput) (
    *mcp.CallToolResult,
    any,
    error,
)
```

### Shared Utility Functions

The `pkg/tools` package provides shared utility functions:
- `ApplyPagination()` - Applies pagination to output strings
- `FormatScannerOutput()` - Formats scanner output with pagination info
- `ParseHostInput()` - Extracts scheme, hostname, and port from URL-style inputs
- `BuildTargetURL()` - Constructs URL from `ScanParams`, omitting default ports
- `ResolveParams()` - Resolves `ScannerInput` into `ScanParams` with scheme inference

## Development Commands

```bash
# Build
make build

# Run linters
make lint

# Run tests with coverage
make test
```

## Testing

The project includes comprehensive test coverage for all packages:

### Test Packages

| Package | Coverage | Description |
|---------|----------|-------------|
| `pkg/storage` | Storage layer | SQLite CRUD operations, pagination |
| `pkg/server` | Server wrapper | Server creation, shutdown, storage access |
| `pkg/models` | Data models | JSON serialization, field validation |
| `pkg/tools` | Tool wrapper | Execution logging, timing, error handling |
| `pkg/tools/history` | History tool | All history actions (list, get, delete, clear) |
| `pkg/tools/shcheck` | shcheck tool | Security headers checker tests |
| `pkg/types` | Constants | Value validation |

### Running Tests

```bash
# Run all tests with coverage
make test

# Run tests for specific package
go test -v ./pkg/storage/...

# Run with race detection
go test -race ./...
```

### Test Coverage Report

After running `make test`, coverage reports are generated:
- `build/coverage.out` - Raw coverage data
- `build/coverage.html` - HTML coverage report

## Dependencies

### Direct Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| modelcontextprotocol/go-sdk | v1.2.0 | MCP protocol implementation |
| go-playground/validator | v10.x | Input validation |
| rs/zerolog | v1.34.0 | Structured logging |
| gorm.io/gorm | v1.25.x | ORM |
| gorm.io/driver/sqlite | v1.5.x | SQLite driver |

## Security Considerations

1. **Authorization Context:** This tool is designed for authorized security testing only
2. **Command Injection:** Nikto arguments are constructed from validated input
3. **Network Access:** Scanner requires network access to targets
4. **Local Storage:** Execution history stored locally in SQLite

## Future Enhancements

Potential additions:
- Additional scanning tools (nmap, sqlmap, etc.)
- Scheduled scans
- PDF/HTML report generation
- Authentication/authorization for MCP clients
- Scan result comparison/diffing
- Webhook notifications
- Scan templates/profiles

## License

BSD 3-Clause License - Copyright (c) 2026, Bohdan Turkynevych.

## Version History

- **Initial:** Basic MCP server with nikto tool
- **v1.0:** Added session persistence, execution history, history management tool
- **v1.1:** Added wapiti web application scanner, comprehensive test suite
- **v1.2:** Added nuclei template-based vulnerability scanner
- **v1.3:** Refactored scanner tools to eliminate code duplication:
  - Added `BaseScanner` struct with common functionality
  - Added shared `ScannerInput` type for all scanner tools
  - Added shared `ApplyPagination()` and `FormatScannerOutput()` functions
  - Added shared constants `DefaultHost` and `DefaultPort` in `pkg/types`
  - Reduced code duplication across nikto, wapiti, nuclei, and fullscan tools
- **v1.4:** Added shcheck security headers checker tool:
  - Analyzes HTTP security headers using shcheck.py with JSON output
  - Supports vhost via custom Host header
  - Included in full_scan parallel scanning
- **v1.5:** Redesigned URL handling for scheme-aware scanning:
  - Added `SchemeHTTP` and `SchemeHTTPS` constants
  - Added `Scheme` field to `ScanParams` for carrying URL scheme through scanning pipeline
  - Added `ParseHostInput()` to extract scheme, hostname, and port from URL-style host inputs (e.g. `https://example.com`)
  - Added `PrepareInput()` on `BaseScanner` to strip URLs before validation
  - Added `ResolveInput()` / `ResolveParams()` for scheme-aware input resolution
  - Redesigned `BuildTargetURL()` to omit default ports (80 for HTTP, 443 for HTTPS)
  - Nikto scanner now passes `-ssl` flag for HTTPS targets
  - Fullscan uses shared `ResolveParams()` for consistent URL handling
