package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tb0hdan/wass-mcp/pkg/storage"
)

type Server struct {
	mcp.Server
	storage storage.Storage
}

func NewServer(impl *mcp.Implementation, store storage.Storage) *Server {
	return &Server{
		Server:  *mcp.NewServer(impl, nil),
		storage: store,
	}
}

func (s *Server) Storage() storage.Storage {
	return s.storage
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
}
