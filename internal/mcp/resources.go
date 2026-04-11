package mcp

import (
	"context"
	"encoding/json"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerResources(s *mcp.Server, store *lctx.Store) {
	s.AddResource(&mcp.Resource{URI: "laightdb://collections"}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		cols, err := store.ListCollections(ctx)
		if err != nil {
			return nil, err
		}
		b, _ := json.Marshal(map[string]any{"collections": cols})
		uri := "laightdb://collections"
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "application/json", Text: string(b)}},
		}, nil
	})
}
