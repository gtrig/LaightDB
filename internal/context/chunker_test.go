package context

import (
	"strings"
	"testing"

	"github.com/gtrig/laightdb/internal/storage"
)

func TestChunkContent(t *testing.T) {
	t.Parallel()
	text := strings.Repeat("word ", 600)
	ch := ChunkContent("p1", text, 100)
	if len(ch) < 1 {
		t.Fatal("expected chunks")
	}
	if ch[0].ParentID != "p1" {
		t.Fatal(ch[0])
	}
	var _ storage.Chunk = ch[0]
}
