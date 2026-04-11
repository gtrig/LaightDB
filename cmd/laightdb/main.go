package main

import (
	"log/slog"
	"os"

	"github.com/gtrig/laightdb/internal/launch"
)

func main() {
	if err := launch.Start(); err != nil {
		slog.Error("laightdb", "err", err)
		os.Exit(1)
	}
}
