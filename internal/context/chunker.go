package context

import (
	"strings"

	"github.com/gtrig/laightdb/internal/storage"
)

const (
	defaultTarget = 512
	overlapTok    = 50
)

// ChunkContent splits text into semantic-ish chunks (~target tokens, overlap).
func ChunkContent(parentID, content string, targetTokens int) []storage.Chunk {
	if targetTokens <= 0 {
		targetTokens = defaultTarget
	}
	approxChars := targetTokens * 4
	overlapChars := overlapTok * 4
	if len(content) == 0 {
		return nil
	}
	var out []storage.Chunk
	paras := strings.Split(content, "\n\n")
	var buf strings.Builder
	idx := 0
	flush := func() {
		s := buf.String()
		if strings.TrimSpace(s) == "" {
			return
		}
		out = append(out, storage.Chunk{
			Index:    idx,
			ParentID: parentID,
			Text:     strings.TrimSpace(s),
		})
		idx++
		buf.Reset()
	}
	for _, p := range paras {
		if buf.Len() > 0 && buf.Len()+len(p) > approxChars {
			flush()
		}
		if buf.Len() == 0 {
			buf.WriteString(p)
			continue
		}
		buf.WriteString("\n\n")
		buf.WriteString(p)
		if buf.Len() >= approxChars {
			flush()
			if overlapChars > 0 && len(p) > overlapChars {
				buf.WriteString(p[len(p)-overlapChars:])
			}
		}
	}
	flush()
	if len(out) == 0 {
		out = append(out, storage.Chunk{Index: 0, ParentID: parentID, Text: strings.TrimSpace(content)})
	}
	return out
}
