# WASS-MCP

A Model Context Protocol (MCP) server for web application security scanning.

## Features

- **MCP Protocol Support** - Full compatibility with MCP clients (Claude, etc.)
- **Nikto Integration** - Web server vulnerability scanning
- **Nuclei Integration** - Template-based vulnerability scanning
- **Wapiti Integration** - Web application vulnerability scanning
- **Execution History** - Persistent storage of scan results
- **Stateless Design** - Survives server restarts without session errors
- **RESTful HTTP Transport** - Streamable HTTP-based MCP protocol

## Requirements

- Go 1.25+
- Nikto (`apt install nikto` or equivalent)
- Nuclei (`go install github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest`)
- Wapiti (`apt install wapiti` or equivalent)
- SQLite3

## Installation

```bash
# Clone the repository
git clone https://github.com/tb0hdan/wass-mcp.git
cd wass-mcp

# Build
make build

# Run
./build/wass-mcp
```

## Usage

### Starting the Server

```bash
# Default (localhost:8989)
./build/wass-mcp

# Custom bind address
./build/wass-mcp --bind 0.0.0.0:8080

# Custom database path
./build/wass-mcp --db /var/lib/wass-mcp/data.db

# Debug mode
./build/wass-mcp --debug
```

### Configuration Options

| Flag | Default | Description |
|------|---------|-------------|
| `--bind` | `localhost:8989` | HTTP server bind address |
| `--db` | `./wass-mcp.db` | SQLite database file path |
| `--debug` | `false` | Enable debug logging |
| `--version` | - | Print version and exit |

### MCP Client Configuration

Add to your MCP client configuration:

```json
{
  "mcpServers": {
    "wass-mcp": {
      "url": "http://localhost:8989/mcp"
    }
  }
}
```

## Available Tools

### nikto

Perform web server vulnerability scans using Nikto.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `host` | string | Yes | Target hostname or IP address |
| `port` | integer | No | Target port (default: 80) |
| `vhost` | string | No | Virtual host header |
| `max_lines` | integer | No | Maximum output lines |
| `offset` | integer | No | Output line offset |

**Example:**

```json
{
  "host": "192.168.1.100",
  "port": 443
}
```

### nuclei

Perform template-based vulnerability scanning using Nuclei.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `host` | string | Yes | Target hostname or IP address |
| `port` | integer | No | Target port (default: 80) |
| `vhost` | string | No | Virtual host header |
| `max_lines` | integer | No | Maximum output lines |
| `offset` | integer | No | Output line offset |

**Vulnerabilities Detected:**
- CVE detection via community templates
- Misconfigurations
- Exposed panels/dashboards
- Default credentials
- Technology detection
- Security headers analysis
- And many more via 8000+ community templates

**Example:**

```json
{
  "host": "192.168.1.100",
  "port": 443
}
```

### wapiti

Perform comprehensive web application vulnerability scans using Wapiti.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `host` | string | Yes | Target hostname or IP address |
| `port` | integer | No | Target port (default: 80) |
| `vhost` | string | No | Virtual host header |
| `max_lines` | integer | No | Maximum output lines |
| `offset` | integer | No | Output line offset |

**Vulnerabilities Detected:**
- SQL Injection / Blind SQL Injection
- Cross-Site Scripting (XSS)
- File Inclusion / Path Traversal
- Command Execution
- CRLF Injection
- Server-Side Request Forgery (SSRF)
- Open Redirects
- HTTP Security Headers
- Content Security Policy issues

**Example:**

```json
{
  "host": "192.168.1.100",
  "port": 8080
}
```

### full_scan

Perform a comprehensive security scan using all available scanners in parallel.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `host` | string | Yes | Target hostname or IP address |
| `port` | integer | No | Target port (default: 80) |
| `vhost` | string | No | Virtual host header |
| `max_lines` | integer | No | Maximum output lines |
| `offset` | integer | No | Output line offset |

**Features:**
- Runs nikto, nuclei and wapiti scanners in parallel
- Merges results into a unified report
- Includes timing and status for each scanner
- Gracefully handles missing scanner binaries

**Example:**

```json
{
  "host": "192.168.1.100",
  "port": 8080
}
```

### history

Browse and manage tool execution history.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `action` | string | Yes | One of: `list`, `get`, `delete`, `clear` |
| `id` | integer | For get/delete | Execution ID |
| `limit` | integer | No | Results per page (default: 10) |
| `offset` | integer | No | Pagination offset |

**Actions:**

- `list` - List execution history with pagination
- `get` - Get full details of a specific execution
- `delete` - Delete a specific execution by ID
- `clear` - Delete all execution history

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /mcp` | MCP protocol endpoint |
| `GET /` | Service information (JSON) |
| `GET /debug/pprof/*` | Profiling endpoints |

## Development

### Building

```bash
make build
```

### Linting

```bash
make lint
```

### Testing

```bash
make test
```

### Project Structure

```
wass-mcp/
├── cmd/wass-mcp/        # Application entry point
├── pkg/
│   ├── server/          # MCP server wrapper
│   ├── storage/         # Database layer (SQLite/GORM)
│   ├── models/          # Data models
│   ├── tools/           # MCP tool implementations
│   │   ├── nikto/       # Nikto web server scanner
│   │   ├── wapiti/      # Wapiti web app scanner
│   │   ├── nuclei/      # Nuclei template scanner
│   │   ├── fullscan/    # Parallel full scan
│   │   └── history/     # History management
│   └── types/           # Shared types and constants
├── docs/                # Documentation
└── build/               # Build output and coverage reports
```

## Security Notice

This tool is intended for **authorized security testing only**. Ensure you have proper authorization before scanning any systems. Unauthorized scanning may be illegal in your jurisdiction.

## Project notes

For complete project notes, design decisions, and architecture overview, please refer to the [Project Notes](docs/PROJECT_NOTES.md) document.

## License

BSD 3-Clause License - Copyright (c) 2026, Bohdan Turkynevych. See [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/new-tool`)
3. Commit your changes (`git commit -am 'Add new scanning tool'`)
4. Push to the branch (`git push origin feature/new-tool`)
5. Create a Pull Request

## Acknowledgments

- [Model Context Protocol](https://modelcontextprotocol.io/) - Protocol specification
- [Nikto](https://cirt.net/Nikto2) - Web server scanner
- [Nuclei](https://github.com/projectdiscovery/nuclei) - Template-based vulnerability scanner
- [Wapiti](https://wapiti-scanner.github.io/) - Web application vulnerability scanner
- [GORM](https://gorm.io/) - Go ORM library
