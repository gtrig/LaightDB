package mcp

import (
	"context"
	"net/http"

	"github.com/gtrig/laightdb/internal/calllog"
	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server bundles MCP tools and resources for LaightDB.
type Server struct {
	Store     *lctx.Store
	CallLog   *calllog.Store
	MCPServer *mcp.Server
}

// NewServer creates an MCP server with tools and resources registered.
// callLog may be nil (no MCP call recording).
func NewServer(store *lctx.Store, callLog *calllog.Store) *Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: "laightdb", Version: "0.1.0"},
		nil,
	)
	registerTools(s, store, callLog)
	registerResources(s, store)
	return &Server{Store: store, CallLog: callLog, MCPServer: s}
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
