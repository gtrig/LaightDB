package mcp

import (
	"context"
	"encoding/json"
	"testing"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/summarize"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPStoreSearchGetGate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store, err := lctx.OpenStore(context.Background(), dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	srv := NewServer(store, nil)
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
	defer func() { _ = sess.Close() }()

	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "store_context",
		Arguments: map[string]any{
			"collection":   "c1",
			"content":      "the quick brown fox jumps",
			"content_type": "doc",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	txt := res.Content[0].(*mcp.TextContent).Text
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(txt), &out); err != nil || out.ID == "" {
		t.Fatalf("store: %q %v", txt, err)
	}
	id := out.ID

	res, err = sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "search_context",
		Arguments: map[string]any{
			"query":  "fox",
			"top_k":  5,
			"detail": "summary",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	txt = res.Content[0].(*mcp.TextContent).Text
	if !json.Valid([]byte(txt)) {
		t.Fatal(txt)
	}

	res, err = sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_context",
		Arguments: map[string]any{
			"id":     id,
			"detail": "summary",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	txt = res.Content[0].(*mcp.TextContent).Text
	var ent map[string]any
	if err := json.Unmarshal([]byte(txt), &ent); err != nil {
		t.Fatal(err)
	}
	if ent["id"] != id {
		t.Fatalf("get id mismatch %+v", ent)
	}
}

func TestMCPDeployCursorIntegration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store, err := lctx.OpenStore(context.Background(), dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	srv := NewServer(store, nil)
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
	defer func() { _ = sess.Close() }()

	proj := t.TempDir()
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "deploy_cursor_integration",
		Arguments: map[string]any{
			"project_root": proj,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("deploy_cursor_integration: %s", res.Content[0].(*mcp.TextContent).Text)
	}
	txt := res.Content[0].(*mcp.TextContent).Text
	var out map[string]any
	if err := json.Unmarshal([]byte(txt), &out); err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("expected ok: %s", txt)
	}
	if out["example_user_prompt"] == nil || out["example_user_prompt"] == "" {
		t.Fatal("missing example_user_prompt")
	}
}
