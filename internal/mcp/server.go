package mcp

import (
	"context"
	"net/http"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server bundles MCP tools and resources for LaightDB.
type Server struct {
	Store     *lctx.Store
	MCPServer *mcp.Server
}

// NewServer creates an MCP server with tools and resources registered.
func NewServer(store *lctx.Store) *Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: "laightdb", Version: "0.1.0"},
		nil,
	)
	registerTools(s, store)
	registerResources(s, store)
	return &Server{Store: store, MCPServer: s}
}

// RunStdio runs the MCP server over stdio until ctx ends.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.MCPServer.Run(ctx, &mcp.StdioTransport{})
}

// StreamableHTTPHandler returns the streamable HTTP handler for MCP.
func (s *Server) StreamableHTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s.MCPServer },
		&mcp.StreamableHTTPOptions{},
	)
}
