package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gtrig/laightdb/internal/config"
	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/embedding"
	"github.com/gtrig/laightdb/internal/mcpdev"
	"github.com/gtrig/laightdb/internal/summarize"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if err := run(); err != nil {
		slog.Error("laightdb-dev-mcp", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Parse()
	var store *lctx.Store
	if !cfg.DevMCPSkipStore {
		sum := summarize.Noop()
		var emb *embedding.Engine
		if e, err := embedding.NewEngine(); err != nil {
			slog.Warn("embedding disabled", "err", err)
		} else {
			emb = e
		}
		var err error
		store, err = lctx.OpenStore(context.Background(), cfg.DataDir, cfg.MemtableBytes, emb, sum)
		if err != nil {
			return err
		}
		defer func() { _ = store.Close() }()
	} else {
		slog.Info("dev mcp: skipping OpenStore (LAIGHTDB_DEV_MCP_SKIP_STORE); safe alongside running laightdb")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dev := mcpdev.NewServer(store, cfg.DataDir)
	if cfg.DevMCPHTTPAddr != "" {
		return dev.ListenAndServeHTTP(ctx, cfg.DevMCPHTTPAddr)
	}
	slog.Info("dev mcp stdio (do not expose to production)", "data_dir", cfg.DataDir)
	return dev.RunStdio(ctx)
}
