package mcp

import (
	"context"
	"encoding/json"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type storeInput struct {
	Collection  string            `json:"collection" jsonschema:"collection namespace"`
	Content     string            `json:"content" jsonschema:"raw text to store"`
	ContentType string            `json:"content_type,omitempty" jsonschema:"code, conversation, doc, or kv"`
	Metadata    map[string]string `json:"metadata,omitempty" jsonschema:"optional key-value metadata"`
}

type searchInput struct {
	Query      string            `json:"query" jsonschema:"search query"`
	Collection string            `json:"collection,omitempty" jsonschema:"filter by collection"`
	Filters    map[string]string `json:"filters,omitempty" jsonschema:"metadata filters"`
	TopK       int               `json:"top_k,omitempty" jsonschema:"max results"`
	Detail     string            `json:"detail,omitempty" jsonschema:"metadata, summary, or full"`
}

type idInput struct {
	ID     string `json:"id" jsonschema:"context entry id"`
	Detail string `json:"detail,omitempty" jsonschema:"metadata, summary, or full"`
}

type emptyOut struct{}

func registerTools(s *mcp.Server, store *lctx.Store) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "store_context",
		Description: "Store text content with optional metadata and collection",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in storeInput) (*mcp.CallToolResult, emptyOut, error) {
		id, err := store.Put(ctx, lctx.PutRequest{
			Collection:  in.Collection,
			Content:     in.Content,
			ContentType: in.ContentType,
			Metadata:    in.Metadata,
		})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		b, _ := json.Marshal(map[string]string{"id": id})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_context",
		Description: "Hybrid full-text and vector search over stored context",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, emptyOut, error) {
		hits, err := store.Search(ctx, lctx.SearchRequest{
			Query:      in.Query,
			Collection: in.Collection,
			Filters:    in.Filters,
			TopK:       in.TopK,
			Detail:     lctx.DetailLevel(in.Detail),
		})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		totalTokens, totalCompact := 0, 0
		for _, h := range hits {
			totalTokens += h.TokenCount
			totalCompact += h.CompactTokenCount
		}
		b, _ := json.Marshal(map[string]any{
			"hits":                hits,
			"total_token_count":   totalTokens,
			"total_compact_count": totalCompact,
			"total_tokens_saved":  totalTokens - totalCompact,
		})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_context",
		Description: "Retrieve a stored context entry by id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, emptyOut, error) {
		d := lctx.DetailSummary
		if in.Detail != "" {
			d = lctx.DetailLevel(in.Detail)
		}
		ent, err := store.Get(ctx, in.ID, d)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "not found"}},
				IsError: true,
			}, emptyOut{}, nil
		}
		b, err := lctx.ProjectJSON(ent, d)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_context",
		Description: "Delete a context entry by id",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, emptyOut, error) {
		if err := store.Delete(ctx, in.ID); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: `{"ok":true}`}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_collections",
		Description: "List collection names",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyOut, error) {
		cols, err := store.ListCollections(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		b, _ := json.Marshal(map[string]any{"collections": cols})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_stats",
		Description: "Database statistics",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyOut, error) {
		st, err := store.Stats(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		b, _ := json.Marshal(st)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})
}
