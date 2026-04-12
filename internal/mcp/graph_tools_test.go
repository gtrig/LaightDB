package mcp

import (
	"context"
	"encoding/json"
	"testing"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/summarize"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestMCPSession(t *testing.T) (*mcp.ClientSession, *lctx.Store) {
	t.Helper()
	dir := t.TempDir()
	store, err := lctx.OpenStore(context.Background(), dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	srv := NewServer(store)
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := srv.MCPServer.Connect(ctx, t1, nil); err != nil {
		t.Fatal(err)
	}
	cli := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	sess, err := cli.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sess.Close() })
	return sess, store
}

func TestMCPLinkContext(t *testing.T) {
	t.Parallel()
	sess, _ := newTestMCPSession(t)
	ctx := context.Background()

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "link_context",
		Arguments: map[string]any{
			"from_id": "node-a",
			"to_id":   "node-b",
			"label":   "child",
			"weight":  1.0,
		},
	})
	if err != nil {
		t.Fatalf("link_context: %v", err)
	}
	if res.IsError {
		t.Fatalf("link_context returned error: %v", res.Content[0].(*mcp.TextContent).Text)
	}
	var out map[string]string
	_ = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
	if out["edge_id"] == "" {
		t.Error("expected non-empty edge_id")
	}
}

func TestMCPUnlinkContext(t *testing.T) {
	t.Parallel()
	sess, store := newTestMCPSession(t)
	ctx := context.Background()

	edgeID, err := store.PutEdge(ctx, lctx.PutEdgeRequest{FromID: "a", ToID: "b", Label: "child"})
	if err != nil {
		t.Fatal(err)
	}

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      "unlink_context",
		Arguments: map[string]any{"edge_id": edgeID},
	})
	if err != nil {
		t.Fatalf("unlink_context: %v", err)
	}
	if res.IsError {
		t.Fatalf("unlink_context error: %v", res.Content[0].(*mcp.TextContent).Text)
	}
}

func TestMCPGetNeighbors(t *testing.T) {
	t.Parallel()
	sess, store := newTestMCPSession(t)
	ctx := context.Background()

	store.PutEdge(ctx, lctx.PutEdgeRequest{FromID: "root", ToID: "child1", Label: "child"}) //nolint:errcheck
	store.PutEdge(ctx, lctx.PutEdgeRequest{FromID: "root", ToID: "child2", Label: "child"}) //nolint:errcheck

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_neighbors",
		Arguments: map[string]any{"id": "root", "max_depth": 1},
	})
	if err != nil {
		t.Fatalf("get_neighbors: %v", err)
	}
	var out map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
	neighbors, _ := out["neighbors"].([]any)
	if len(neighbors) != 2 {
		t.Errorf("expected 2 neighbors, got %d", len(neighbors))
	}
}

func TestMCPGetSubtree(t *testing.T) {
	t.Parallel()
	sess, store := newTestMCPSession(t)
	ctx := context.Background()

	store.PutEdge(ctx, lctx.PutEdgeRequest{FromID: "root", ToID: "A", Label: "child"}) //nolint:errcheck
	store.PutEdge(ctx, lctx.PutEdgeRequest{FromID: "A", ToID: "A1", Label: "child"})   //nolint:errcheck

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_subtree",
		Arguments: map[string]any{"id": "root", "max_depth": 0},
	})
	if err != nil {
		t.Fatalf("get_subtree: %v", err)
	}
	var out map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
	edges, _ := out["edges"].([]any)
	if len(edges) != 2 {
		t.Errorf("expected 2 subtree edges, got %d", len(edges))
	}
	if out["root"] != "root" {
		t.Errorf("expected root='root', got %v", out["root"])
	}
}

func TestMCPGraphSearch(t *testing.T) {
	t.Parallel()
	sess, store := newTestMCPSession(t)
	ctx := context.Background()

	idA, _ := store.Put(ctx, lctx.PutRequest{Collection: "test", Content: "alpha content"})
	idB, _ := store.Put(ctx, lctx.PutRequest{Collection: "test", Content: "beta content"})
	store.PutEdge(ctx, lctx.PutEdgeRequest{FromID: idA, ToID: idB, Label: "child"}) //nolint:errcheck

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "graph_search",
		Arguments: map[string]any{
			"query":         "content",
			"focus_node_id": idA,
			"max_depth":     1,
			"top_k":         5,
		},
	})
	if err != nil {
		t.Fatalf("graph_search: %v", err)
	}
	if res.IsError {
		t.Fatalf("graph_search error: %v", res.Content[0].(*mcp.TextContent).Text)
	}
	var out map[string]any
	_ = json.Unmarshal([]byte(res.Content[0].(*mcp.TextContent).Text), &out)
	if out["hits"] == nil {
		t.Error("expected hits in graph_search result")
	}
}

func TestMCPSuggestLinks_NoEmbedder(t *testing.T) {
	t.Parallel()
	sess, store := newTestMCPSession(t)
	ctx := context.Background()

	id, _ := store.Put(ctx, lctx.PutRequest{Collection: "test", Content: "some node"})

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      "suggest_links",
		Arguments: map[string]any{"id": id, "threshold": 0.7, "top_k": 5},
	})
	if err != nil {
		t.Fatalf("suggest_links: %v", err)
	}
	// Without embedder, should succeed with empty suggestions.
	if res.IsError {
		t.Fatalf("suggest_links error: %v", res.Content[0].(*mcp.TextContent).Text)
	}
}

func TestMCPSuggestLinks_NodeNotFound(t *testing.T) {
	t.Parallel()
	sess, _ := newTestMCPSession(t)
	ctx := context.Background()

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      "suggest_links",
		Arguments: map[string]any{"id": "does-not-exist"},
	})
	if err != nil {
		t.Fatalf("suggest_links: %v", err)
	}
	if !res.IsError {
		t.Error("expected error result for non-existent node")
	}
}
