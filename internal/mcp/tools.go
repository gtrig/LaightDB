package mcp

import (
	"context"
	"encoding/json"

	cursorintegration "github.com/gtrig/laightdb/integrations/cursor"
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

// --- Graph input types ---

type linkInput struct {
	FromID   string            `json:"from_id" jsonschema:"source node id"`
	ToID     string            `json:"to_id" jsonschema:"target node id"`
	Label    string            `json:"label,omitempty" jsonschema:"edge label, e.g. child, related_to"`
	Weight   float64           `json:"weight,omitempty" jsonschema:"optional edge weight 0-1"`
	Source   string            `json:"source,omitempty" jsonschema:"user or auto"`
	Metadata map[string]string `json:"metadata,omitempty" jsonschema:"optional edge metadata"`
}

type unlinkInput struct {
	EdgeID string `json:"edge_id" jsonschema:"edge id to remove"`
}

type neighborsInput struct {
	ID       string `json:"id" jsonschema:"node id"`
	MaxDepth int    `json:"max_depth,omitempty" jsonschema:"BFS depth limit (0=unlimited)"`
}

type subtreeInput struct {
	ID       string `json:"id" jsonschema:"root node id"`
	MaxDepth int    `json:"max_depth,omitempty" jsonschema:"BFS depth limit (0=unlimited)"`
}

type graphSearchInput struct {
	Query       string            `json:"query" jsonschema:"search query"`
	FocusNodeID string            `json:"focus_node_id,omitempty" jsonschema:"graph proximity anchor node id"`
	MaxDepth    int               `json:"max_depth,omitempty" jsonschema:"graph BFS depth (0=unlimited)"`
	Collection  string            `json:"collection,omitempty" jsonschema:"filter by collection"`
	Filters     map[string]string `json:"filters,omitempty" jsonschema:"metadata filters"`
	TopK        int               `json:"top_k,omitempty" jsonschema:"max results"`
	Detail      string            `json:"detail,omitempty" jsonschema:"metadata, summary, or full"`
}

type suggestLinksInput struct {
	ID        string  `json:"id" jsonschema:"node id to find suggestions for"`
	Threshold float64 `json:"threshold,omitempty" jsonschema:"minimum cosine similarity (default 0.7)"`
	TopK      int     `json:"top_k,omitempty" jsonschema:"max suggestions (default 10)"`
}

type deployCursorInput struct {
	ProjectRoot    string `json:"project_root" jsonschema:"path to the Cursor workspace root (directory that will contain .cursor)"`
	OverwriteSkill bool   `json:"overwrite_skill,omitempty" jsonschema:"if true, replace existing laightdb-rolling-context SKILL.md"`
	MergeHooks     *bool  `json:"merge_hooks,omitempty" jsonschema:"if false, do not change hooks.json (default true)"`
}

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

	// --- Graph / Mindmap tools ---

	mcp.AddTool(s, &mcp.Tool{
		Name:        "link_context",
		Description: "Create a directed edge between two context entries (mindmap relationship)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in linkInput) (*mcp.CallToolResult, emptyOut, error) {
		id, err := store.PutEdge(ctx, lctx.PutEdgeRequest{
			FromID:   in.FromID,
			ToID:     in.ToID,
			Label:    in.Label,
			Weight:   in.Weight,
			Source:   in.Source,
			Metadata: in.Metadata,
		})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		b, _ := json.Marshal(map[string]string{"edge_id": id})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "unlink_context",
		Description: "Remove a directed edge between two context entries",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in unlinkInput) (*mcp.CallToolResult, emptyOut, error) {
		if err := store.DeleteEdge(ctx, in.EdgeID); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		b, _ := json.Marshal(map[string]bool{"ok": true})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_neighbors",
		Description: "Get nodes connected to a given context entry via graph edges",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in neighborsInput) (*mcp.CallToolResult, emptyOut, error) {
		hits := store.GraphNeighborhood(in.ID, in.MaxDepth)
		b, _ := json.Marshal(map[string]any{"neighbors": hits})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_subtree",
		Description: "Return a mindmap subtree (directed BFS from a root node) as structured JSON",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in subtreeInput) (*mcp.CallToolResult, emptyOut, error) {
		edges := store.SubtreeEdges(in.ID, in.MaxDepth)
		type edgeJSON struct {
			EdgeID   string  `json:"edge_id"`
			TargetID string  `json:"target_id"`
			Label    string  `json:"label"`
			Weight   float64 `json:"weight"`
			Source   string  `json:"source"`
		}
		out := make([]edgeJSON, 0, len(edges))
		for _, e := range edges {
			out = append(out, edgeJSON{
				EdgeID:   e.EdgeID,
				TargetID: e.TargetID,
				Label:    e.Label,
				Weight:   e.Weight,
				Source:   e.Source,
			})
		}
		b, _ := json.Marshal(map[string]any{"root": in.ID, "edges": out})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "graph_search",
		Description: "Hybrid search combining BM25, vector similarity, and graph proximity (3-signal RRF)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in graphSearchInput) (*mcp.CallToolResult, emptyOut, error) {
		hits, err := store.Search(ctx, lctx.SearchRequest{
			Query:       in.Query,
			Collection:  in.Collection,
			Filters:     in.Filters,
			TopK:        in.TopK,
			Detail:      lctx.DetailLevel(in.Detail),
			FocusNodeID: in.FocusNodeID,
			MaxDepth:    in.MaxDepth,
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
		Name:        "suggest_links",
		Description: "Auto-discover missing relationships for a node using vector similarity",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in suggestLinksInput) (*mcp.CallToolResult, emptyOut, error) {
		threshold := in.Threshold
		if threshold <= 0 {
			threshold = 0.7
		}
		suggestions, err := store.SuggestLinks(ctx, in.ID, threshold, in.TopK)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		b, _ := json.Marshal(map[string]any{"suggestions": suggestions})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "deploy_cursor_integration",
		Description: "Install the LaightDB Cursor skill plus hooks (sessionStart memory policy + beforeSubmitPrompt LaightDB search) under project_root/.cursor (optional hooks.json merge)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in deployCursorInput) (*mcp.CallToolResult, emptyOut, error) {
		if in.ProjectRoot == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: `{"error":"project_root is required"}`}},
				IsError: true,
			}, emptyOut{}, nil
		}
		mergeHooks := true
		if in.MergeHooks != nil {
			mergeHooks = *in.MergeHooks
		}
		res, err := cursorintegration.Deploy(in.ProjectRoot, cursorintegration.DeployOptions{
			OverwriteSkill: in.OverwriteSkill,
			MergeHooks:     mergeHooks,
		})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, emptyOut{}, nil
		}
		const examplePrompt = "Deploy the LaightDB rolling-context Cursor integration into my project at <PROJECT_ROOT> (merge hooks unless I already customized hooks.json)."
		b, _ := json.Marshal(map[string]any{
			"ok":                   true,
			"written":              res.Written,
			"skipped":              res.Skipped,
			"hooks_merged":         res.HooksMerged,
			"hooks_merge_note":     res.HooksMergeNote,
			"example_user_prompt":  examplePrompt,
			"manual_assets_folder": "integrations/cursor in the LaightDB repository",
		})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})
}
