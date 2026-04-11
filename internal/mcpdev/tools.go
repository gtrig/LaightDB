package mcpdev

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxReadFileBytes = 512 * 1024
const maxTreeFiles = 500
const maxTreeDepth = 8

type emptyOut struct{}

type treeInput struct {
	MaxDepth int `json:"max_depth,omitempty" jsonschema:"max directory depth (default 6)"`
}

type readFileInput struct {
	Path   string `json:"path" jsonschema:"path relative to LAIGHTDB data dir (e.g. wal.log, auth/sessions.json)"`
	Format string `json:"format,omitempty" jsonschema:"text, json, hex, or base64 (default text)"`
}

func registerDevTools(s *mcp.Server, store *lctx.Store, dataDir string) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "debug_runtime",
		Description: "Go version, OS/arch, process id, and redacted LAIGHTDB_* environment keys (development diagnostics)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyOut, error) {
		_ = ctx
		out := map[string]any{
			"go_version": runtime.Version(),
			"go_os":      runtime.GOOS,
			"go_arch":    runtime.GOARCH,
			"pid":        os.Getpid(),
			"env":        redactedLaightEnv(),
		}
		b, _ := json.Marshal(out)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "debug_store_stats",
		Description: "High-level store stats (entries, collections, vector index size) for the open database",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyOut, error) {
		if store == nil {
			msg := "store not opened (set LAIGHTDB_DEV_MCP_SKIP_STORE=false when no other process uses the data dir, or call GET /v1/stats on the API)"
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: msg}}, IsError: true}, emptyOut{}, nil
		}
		st, err := store.Stats(ctx)
		if err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
		}
		st["data_dir"] = dataDir
		b, _ := json.Marshal(st)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "debug_data_tree",
		Description: "List files and directories under the data dir (non-recursive depth limit; capped file count)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in treeInput) (*mcp.CallToolResult, emptyOut, error) {
		_ = ctx
		depth := in.MaxDepth
		if depth <= 0 {
			depth = 6
		}
		if depth > maxTreeDepth {
			depth = maxTreeDepth
		}
		type node struct {
			Path  string `json:"path"`
			Size  int64  `json:"size,omitempty"`
			IsDir bool   `json:"is_dir"`
		}
		var nodes []node
		var nfiles int
		root := filepath.Clean(dataDir)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			dep := 0
			if rel != "." {
				dep = strings.Count(rel, string(filepath.Separator)) + 1
			}
			if dep > depth {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			var sz int64
			if !d.IsDir() {
				info, err := d.Info()
				if err != nil {
					return err
				}
				sz = info.Size()
			}
			p := rel
			if p == "." {
				p = ""
			}
			nodes = append(nodes, node{Path: p, Size: sz, IsDir: d.IsDir()})
			if !d.IsDir() {
				nfiles++
				if nfiles >= maxTreeFiles {
					return fs.SkipAll
				}
			}
			return nil
		})
		if err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
		}
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].Path < nodes[j].Path })
		b, _ := json.Marshal(map[string]any{"data_dir": dataDir, "max_depth": depth, "entries": nodes})
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "debug_read_file",
		Description: "Read a file under the data directory (max 512 KiB). Raw auth/users.json and auth/tokens.json are blocked; use debug_auth_public instead.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in readFileInput) (*mcp.CallToolResult, emptyOut, error) {
		_ = ctx
		rel := strings.TrimSpace(in.Path)
		if isBlockedAuthPath(rel) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "refused: use debug_auth_public for auth/*.json"}},
				IsError: true,
			}, emptyOut{}, nil
		}
		abs, err := resolveUnderDataDir(dataDir, rel)
		if err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
		}
		fi, err := os.Stat(abs)
		if err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
		}
		if fi.IsDir() {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "path is a directory"}}, IsError: true}, emptyOut{}, nil
		}
		if fi.Size() > maxReadFileBytes {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("file too large (%d bytes, max %d)", fi.Size(), maxReadFileBytes)}},
				IsError: true,
			}, emptyOut{}, nil
		}
		raw, err := os.ReadFile(abs)
		if err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
		}
		format := strings.ToLower(strings.TrimSpace(in.Format))
		if format == "" {
			format = "text"
		}
		var text string
		switch format {
		case "text":
			if !utf8ValidPrintable(raw) {
				text = base64.StdEncoding.EncodeToString(raw) + "\n(base64: binary or non-utf8; retry format=hex)"
			} else {
				text = string(raw)
			}
		case "json":
			var v any
			if err := json.Unmarshal(raw, &v); err != nil {
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
			}
			b, _ := json.MarshalIndent(v, "", "  ")
			text = string(b)
		case "hex":
			text = hex.EncodeToString(raw)
		case "base64":
			text = base64.StdEncoding.EncodeToString(raw)
		default:
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "format must be text, json, hex, or base64"}}, IsError: true}, emptyOut{}, nil
		}
		out := map[string]any{"path": rel, "bytes": len(raw), "content": text}
		b, _ := json.Marshal(out)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "debug_auth_public",
		Description: "Summarize auth/users.json and auth/tokens.json without password or token hashes (dev debugging only)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyOut, error) {
		_ = ctx
		authDir := filepath.Join(dataDir, "auth")
		var users []auth.User
		if err := loadJSONSlice(filepath.Join(authDir, "users.json"), &users); err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
		}
		type userPub struct {
			ID        string    `json:"id"`
			Username  string    `json:"username"`
			Role      auth.Role `json:"role"`
			CreatedAt string    `json:"created_at"`
			UpdatedAt string    `json:"updated_at"`
		}
		up := make([]userPub, 0, len(users))
		for _, u := range users {
			up = append(up, userPub{
				ID: u.ID, Username: u.Username, Role: u.Role,
				CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
				UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			})
		}
		var tokens []auth.APIToken
		if err := loadJSONSlice(filepath.Join(authDir, "tokens.json"), &tokens); err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
		}
		type tokPub struct {
			ID        string     `json:"id"`
			UserID    string     `json:"user_id"`
			Name      string     `json:"name"`
			Prefix    string     `json:"prefix"`
			Role      auth.Role  `json:"role"`
			CreatedAt string     `json:"created_at"`
			RevokedAt *string    `json:"revoked_at,omitempty"`
			Active    bool       `json:"active"`
		}
		tp := make([]tokPub, 0, len(tokens))
		for _, t := range tokens {
			var rev *string
			if t.RevokedAt != nil {
				s := t.RevokedAt.Format("2006-01-02T15:04:05Z")
				rev = &s
			}
			tp = append(tp, tokPub{
				ID: t.ID, UserID: t.UserID, Name: t.Name, Prefix: t.Prefix, Role: t.Role,
				CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
				RevokedAt: rev,
				Active:    t.Active(),
			})
		}
		var sessions []auth.Session
		if err := loadJSONSlice(filepath.Join(authDir, "sessions.json"), &sessions); err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, emptyOut{}, nil
		}
		type sessPub struct {
			ID        string `json:"id"`
			UserID    string `json:"user_id"`
			ExpiresAt string `json:"expires_at"`
		}
		sp := make([]sessPub, 0, len(sessions))
		for _, s := range sessions {
			sp = append(sp, sessPub{ID: s.ID, UserID: s.UserID, ExpiresAt: s.ExpiresAt.Format("2006-01-02T15:04:05Z")})
		}
		out := map[string]any{"users": up, "tokens": tp, "sessions": sp}
		b, _ := json.MarshalIndent(out, "", "  ")
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, emptyOut{}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "debug_process_logs_note",
		Description: "Explains where LaightDB application logs go (stderr); there is no rotating log file unless you redirect stdout/stderr yourself",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, emptyOut, error) {
		_ = ctx
		msg := `LaightDB uses the standard library log/slog with output to the process stderr by default. There is no built-in log file or "logs database". For Docker, use "docker logs <container>". To persist logs, redirect stderr to a file when starting the process. The storage write-ahead log is the file wal.log under the data directory (binary; use debug_read_file with format=hex for a snippet).`
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: msg}}}, emptyOut{}, nil
	})
}

func isBlockedAuthPath(rel string) bool {
	r := filepath.ToSlash(strings.TrimSpace(rel))
	return strings.EqualFold(r, "auth/users.json") ||
		strings.EqualFold(r, "auth/tokens.json") ||
		strings.HasPrefix(strings.ToLower(r), "auth/users.json/") ||
		strings.HasPrefix(strings.ToLower(r), "auth/tokens.json/")
}

func loadJSONSlice[T any](path string, out *[]T) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		*out = nil
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		*out = nil
		return nil
	}
	return json.Unmarshal(data, out)
}

func redactedLaightEnv() map[string]string {
	out := make(map[string]string)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "LAIGHTDB_") {
			continue
		}
		i := strings.IndexByte(e, '=')
		if i <= 0 {
			continue
		}
		k, v := e[:i], e[i+1:]
		lk := strings.ToLower(k)
		if strings.Contains(lk, "KEY") || strings.Contains(lk, "SECRET") || strings.Contains(lk, "PASSWORD") || strings.Contains(lk, "TOKEN") {
			out[k] = "(redacted)"
			continue
		}
		if len(v) > 120 {
			out[k] = v[:120] + "…"
			continue
		}
		out[k] = v
	}
	return out
}

func utf8ValidPrintable(b []byte) bool {
	s := string(b)
	for _, r := range s {
		if r == 0 {
			return false
		}
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}
