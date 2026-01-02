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
- Merged results from all scanners (nikto, wapiti)

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

The `Register` method:
1. Checks if required binary exists (e.g., nikto, wapiti)
2. Creates an `mcp.Tool` definition
3. Wraps the handler with `WrapToolHandler` for logging
4. Registers with `mcp.AddTool`
5. Returns error if tool cannot be registered (e.g., missing binary)

### Handler Signature (MCP SDK v1.x)

```go
func Handler(ctx context.Context, req *mcp.CallToolRequest, input InputType) (
    *mcp.CallToolResult,
    OutputType,
    error,
)
```

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
