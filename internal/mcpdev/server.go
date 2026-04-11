package mcpdev

import (
	"context"
	"net/http"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server is a development-only MCP server (stdio) for inspecting the local data directory and store.
type Server struct {
	Store     *lctx.Store
	DataDir   string
	MCPServer *mcp.Server
}

// NewServer registers dev debug tools. dataDir must be the same directory passed to OpenStore.
func NewServer(store *lctx.Store, dataDir string) *Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: "laightdb-dev", Version: "0.1.0"},
		nil,
	)
	registerDevTools(s, store, dataDir)
	return &Server{Store: store, DataDir: dataDir, MCPServer: s}
}

// RunStdio runs until ctx is cancelled.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.MCPServer.Run(ctx, &mcp.StdioTransport{})
}

// StreamableHTTPHandler returns the streamable HTTP handler for MCP (same path pattern as production: mount at /mcp).
func (s *Server) StreamableHTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s.MCPServer },
		&mcp.StreamableHTTPOptions{},
	)
}
