package launch

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gtrig/laightdb/internal/auth"
	"github.com/gtrig/laightdb/internal/config"
	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/embedding"
	"github.com/gtrig/laightdb/internal/mcp"
	"github.com/gtrig/laightdb/internal/server"
	"github.com/gtrig/laightdb/internal/summarize"
)

// Start runs the LaightDB process (HTTP and/or MCP stdio) until interrupted.
func Start() error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg := config.Parse()
	sum := pickSummarizer(cfg.Summarizer)
	var emb *embedding.Engine
	e, err := embedding.NewEngine()
	if err != nil {
		slog.Warn("embedding disabled", "err", err)
	} else {
		emb = e
	}
	store, err := lctx.OpenStore(context.Background(), cfg.DataDir, cfg.MemtableBytes, emb, sum)
	if err != nil {
		return err
	}
	defer store.Close()

	authStore, err := auth.NewFileAuthStore(filepath.Join(cfg.DataDir, "auth"), cfg.SessionTTL)
	if err != nil {
		return fmt.Errorf("init auth store: %w", err)
	}

	if err := bootstrapUser(context.Background(), cfg, authStore); err != nil {
		return fmt.Errorf("bootstrap user: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ms := mcp.NewServer(store)

	if cfg.MCPTransport == "stdio" {
		slog.Info("mcp stdio (no http)")
		return ms.RunStdio(ctx)
	}

	rl := auth.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	hs := server.NewHTTPServer(store, authStore)
	hs.Mux.Handle("/mcp", ms.StreamableHTTPHandler())
	handler := hs.BuildHandler(
		auth.RateLimitMiddleware(rl),
		auth.Middleware(authStore),
	)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: handler}
	go func() {
		slog.Info("http + mcp streamable", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http", "err", err)
		}
	}()
	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shCtx)
}

func bootstrapUser(ctx context.Context, cfg *config.Config, store *auth.FileAuthStore) error {
	if cfg.BootstrapUser == "" || store.UserCount() > 0 {
		return nil
	}
	parts := strings.SplitN(cfg.BootstrapUser, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("LAIGHTDB_BOOTSTRAP_USER must be username:password")
	}
	u, err := store.CreateUser(ctx, parts[0], parts[1], auth.RoleAdmin)
	if err != nil {
		return err
	}
	slog.Info("bootstrap admin created", "username", u.Username)
	return nil
}

func pickSummarizer(name string) summarize.Summarizer {
	switch name {
	case "openai":
		return summarize.NewOpenAI()
	case "anthropic":
		return summarize.NewAnthropic()
	case "ollama":
		return summarize.NewOllama()
	default:
		return summarize.Noop()
	}
}
