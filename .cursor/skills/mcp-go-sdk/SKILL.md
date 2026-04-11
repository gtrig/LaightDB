---
name: mcp-go-sdk
description: Build MCP servers using the official Go SDK (github.com/modelcontextprotocol/go-sdk/mcp). Use when implementing MCP tools, resources, server setup, transport configuration, or anything in internal/mcp/.
---

# MCP Go SDK Reference

SDK: `github.com/modelcontextprotocol/go-sdk/mcp` v1.5+

## Server Creation

```go
import "github.com/modelcontextprotocol/go-sdk/mcp"

server := mcp.NewServer(
    &mcp.Implementation{Name: "laightdb", Version: "v0.1.0"},
    nil, // *ServerOptions (capabilities, keep-alive, etc.)
)
```

## Adding Tools

Tools use typed Go structs for input. Struct fields use `json` and `jsonschema` tags:

```go
type SearchInput struct {
    Query      string            `json:"query" jsonschema:"search query text"`
    Collection string            `json:"collection,omitempty" jsonschema:"collection to search in"`
    TopK       int               `json:"top_k,omitempty" jsonschema:"number of results (default 10)"`
    Filters    map[string]string `json:"filters,omitempty" jsonschema:"metadata key-value filters"`
    Detail     string            `json:"detail,omitempty" jsonschema:"metadata, summary, or full"`
}

mcp.AddTool(server, &mcp.Tool{
    Name:        "search_context",
    Description: "Search stored context using hybrid full-text and semantic search",
}, handleSearch)
```

## Tool Handler Signature

```go
func handleSearch(
    ctx context.Context,
    req *mcp.CallToolRequest,
    input SearchInput,
) (*mcp.CallToolResult, any, error) {
    // Business logic here...

    // Return text content:
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{Text: resultJSON},
        },
    }, nil, nil

    // Return error to client (not a Go error):
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{Text: "context not found"},
        },
        IsError: true,
    }, nil, nil
}
```

Three return values: `(*CallToolResult, OutputType, error)`
- Return Go `error` only for internal failures (transport/protocol)
- Use `IsError: true` on the result for user-facing tool errors
- Second return value (OutputType) is auto-serialized as structured content

## Transports

### stdio (subprocess mode)

```go
if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
    log.Fatal(err)
}
```

### Streamable HTTP (network mode)

```go
handler := mcp.NewStreamableHTTPHandler(
    func(r *http.Request) *mcp.Server {
        return server
    },
    &mcp.StreamableHTTPOptions{
        // Stateless: true,  // for stateless deployments
    },
)
mux := http.NewServeMux()
mux.Handle("/mcp", handler)
http.ListenAndServe(":8080", mux)
```

## Adding Resources

Resources expose data that clients can browse/read:

```go
mcp.AddResource(server, &mcp.Resource{
    Name:        "collections",
    URI:         "laightdb://collections",
    Description: "List all context collections",
    MimeType:    "application/json",
}, handleListCollectionsResource)

func handleListCollectionsResource(
    ctx context.Context,
    req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
    // Return resource content
    return &mcp.ReadResourceResult{
        Contents: []mcp.ResourceContents{
            &mcp.TextResourceContents{
                URI:      "laightdb://collections",
                MimeType: "application/json",
                Text:     collectionsJSON,
            },
        },
    }, nil
}
```

## Progress Reporting

For long operations (like bulk store), report progress:

```go
func handleBulkStore(ctx context.Context, req *mcp.CallToolRequest, input BulkInput) (*mcp.CallToolResult, any, error) {
    if token := mcp.ProgressToken(ctx); token != nil {
        for i, item := range input.Items {
            mcp.SendProgress(ctx, float64(i), float64(len(input.Items)))
            // process item...
        }
    }
    return &mcp.CallToolResult{...}, nil, nil
}
```

## Additional Reference

- SDK docs: https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp
- Protocol docs: https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/protocol.md
- Examples: https://github.com/modelcontextprotocol/go-sdk/tree/main/examples
