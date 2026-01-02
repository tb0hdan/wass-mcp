package tools

import (
	"github.com/tb0hdan/wass-mcp/pkg/server"
)

type Tool interface {
	Register(srv *server.Server) error
}
