package context

import (
	"strings"
	"testing"
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
	_ = ch[0]
}
