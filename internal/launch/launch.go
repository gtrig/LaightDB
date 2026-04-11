package launch

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ms := mcp.NewServer(store)

	if cfg.MCPTransport == "stdio" {
		slog.Info("mcp stdio (no http)")
		return ms.RunStdio(ctx)
	}

	hs := server.NewHTTPServer(store)
	hs.Mux.Handle("/mcp", ms.StreamableHTTPHandler())
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: hs.Handler()}
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
