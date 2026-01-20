package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/tb0hdan/wass-mcp/pkg/server"
	"github.com/tb0hdan/wass-mcp/pkg/storage"
	"github.com/tb0hdan/wass-mcp/pkg/tools"
	"github.com/tb0hdan/wass-mcp/pkg/tools/fullscan"
	"github.com/tb0hdan/wass-mcp/pkg/tools/history"
	"github.com/tb0hdan/wass-mcp/pkg/tools/nikto"
	"github.com/tb0hdan/wass-mcp/pkg/tools/nuclei"
	"github.com/tb0hdan/wass-mcp/pkg/tools/wapiti"
)

const (
	ServerName      = "wass-mcp"
	ServiceName     = "Web Application Security Scanner MCP Server"
	ShutdownTimeout = 10 * time.Second
)

//go:embed VERSION
var Version string

func main() {
	var (
		debug        bool
		bindAddr     string
		dbPath       string
		printVersion bool
	)
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.StringVar(&bindAddr, "bind", "localhost:8989", "bind address (host:port)")
	flag.StringVar(&dbPath, "db", "build/wass-mcp.db", "SQLite database file path")
	flag.BoolVar(&printVersion, "version", false, "print version and exit")
	flag.Parse()
	// Sanitize version
	version := strings.TrimSpace(Version)
	// Check if the version flag is set
	if printVersion {
		fmt.Printf("%s Version: %s\n", ServiceName, version)
		os.Exit(0)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logger.Debug().Msg("debug mode enabled")
	}

	impl := &mcp.Implementation{
		Name:    ServerName,
		Version: version,
	}

	// Initialize storage
	storeCfg := storage.Config{
		DatabasePath: dbPath,
		Debug:        debug,
	}
	store, err := storage.NewSQLiteStorage(storeCfg)
	if err != nil {
		logger.Fatal().Msgf("Failed to initialize storage: %v", err)
	}
	logger.Info().Msgf("Database initialized at %s", dbPath)

	srv := server.NewServer(impl, store)

	// Create scanner instances.
	scanners := []tools.Scanner{
		nikto.New(logger),
		wapiti.New(logger),
		nuclei.New(logger),
	}

	// Create tool instances.
	toolList := []tools.Tool{
		fullscan.New(logger, scanners...),
		history.New(logger),
	}

	// Add individual scanners as tools
	for _, scanner := range scanners {
		toolList = append(toolList, scanner)
	}

	// Register all tools
	for _, tool := range toolList {
		if err := tool.Register(srv); err != nil {
			logger.Error().Msgf("Failed to register tool: %v", err)
		}
	}
	// Create HTTP handler for MCP server
	// Stateless mode avoids "session not found" errors after server restart
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return &srv.Server
	}, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})

	http.Handle("/mcp", handler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"service": ServiceName,
			"version": version,
			"endpoints": map[string]string{
				"mcp": "/mcp",
			},
		})
	})

	logger.Info().Msgf("%s starting on address %s", ServiceName, bindAddr)
	logger.Info().Msgf("MCP endpoint available at: http://%s/mcp", bindAddr)

	go func() {
		//nolint:gosec
		if err := http.ListenAndServe(bindAddr, nil); !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Msgf("%s failed to start: %v", ServerName, err)
		}
	}()
	<-signalCtx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	// Shutdown MCP server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Msgf("%s shutdown error: %v", ServiceName, err)
	} else {
		logger.Info().Msgf("%s shutdown complete", ServiceName)
	}
}
