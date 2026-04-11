package context

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gtrig/laightdb/internal/embedding"
	"github.com/gtrig/laightdb/internal/index"
	"github.com/gtrig/laightdb/internal/storage"
	"github.com/gtrig/laightdb/internal/summarize"
)

const (
	docKeyPrefix = "d:"
)

// Store is the application facade over storage + indexes.
type Store struct {
	dir   string
	eng   *storage.Engine
	ft    *index.FullText
	meta  *index.MetadataIndex
	vec   *index.VectorIndex
	embed *embedding.Engine
	sum   summarize.Summarizer
}

// OpenStore opens the database and indexes under dataDir.
func OpenStore(ctx context.Context, dataDir string, memBytes int, emb *embedding.Engine, sum summarize.Summarizer) (*Store, error) {
	_ = ctx
	eng, err := storage.OpenEngine(dataDir, memBytes)
	if err != nil {
		return nil, fmt.Errorf("store engine: %w", err)
	}
	vec, err := index.OpenVectorIndex(filepath.Join(dataDir, "vector.hnsw"))
	if err != nil {
		_ = eng.Close()
		return nil, fmt.Errorf("store vector: %w", err)
	}
	s := &Store{
		dir:   dataDir,
		eng:   eng,
		ft:    index.NewFullText(),
		meta:  index.NewMetadataIndex(),
		vec:   vec,
		embed: emb,
		sum:   sum,
	}
	if err := s.rebuildIndexes(); err != nil {
		_ = eng.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) rebuildIndexes() error {
	keys := s.eng.PrefixKeys(docKeyPrefix)
	for _, k := range keys {
		raw, ok := s.eng.Get(k)
		if !ok {
			continue
		}
		ent, err := storage.Decode(raw)
		if err != nil {
			continue
		}
		s.ft.AddDocument(ent.ID, ent.Content)
		s.meta.Set(ent.ID, ent.Metadata)
	}
	return nil
}

// Close releases resources.
func (s *Store) Close() error {
	return s.eng.Close()
}

// PutRequest is ingest input.
type PutRequest struct {
	Collection  string
	Content     string
	ContentType string
	Metadata    map[string]string
}

// Put stores a new context entry; returns assigned ID.
func (s *Store) Put(ctx context.Context, req PutRequest) (string, error) {
	if req.Content == "" {
		return "", fmt.Errorf("store put: empty content")
	}
	id := uuid.New().String()
	if req.Metadata == nil {
		req.Metadata = map[string]string{}
	}
	now := time.Now()
	chunks := ChunkContent(id, req.Content, 512)
	var emb []float32
	if s.embed != nil {
		var err error
		emb, err = s.embed.Embed(ctx, req.Content)
		if err != nil {
			return "", fmt.Errorf("store embed: %w", err)
		}
		for i := range chunks {
			ev, err := s.embed.Embed(ctx, chunks[i].Text)
			if err != nil {
				return "", err
			}
			chunks[i].Embedding = ev
		}
	}
	sum := s.sum
	if sum == nil {
		sum = summarize.Noop()
	}
	summary, _ := sum.Summarize(ctx, req.Content)
	compact := Compact(req.Content)
	ent := storage.ContextEntry{
		ID:                id,
		Collection:        req.Collection,
		Content:           req.Content,
		CompactContent:    compact,
		ContentType:       req.ContentType,
		Summary:           summary,
		Chunks:            chunks,
		Metadata:          req.Metadata,
		Embedding:         emb,
		CreatedAt:         now,
		UpdatedAt:         now,
		TokenCount:        EstimateTokens(req.Content),
		CompactTokenCount: EstimateTokens(compact),
	}
	data := storage.Encode(ent)
	key := docKeyPrefix + id
	if err := s.eng.Put(key, data); err != nil {
		return "", err
	}
	s.ft.AddDocument(id, req.Content)
	s.meta.Set(id, req.Metadata)
	if s.embed != nil && len(emb) > 0 {
		if err := s.vec.Upsert(id, emb); err != nil {
			return "", err
		}
	}
	for _, ch := range chunks {
		if len(ch.Embedding) > 0 {
			cid := fmt.Sprintf("%s#%d", id, ch.Index)
			if err := s.vec.Upsert(cid, ch.Embedding); err != nil {
				return "", err
			}
		}
	}
	return id, nil
}

// Get retrieves by ID at detail level.
func (s *Store) Get(ctx context.Context, id string, d DetailLevel) (storage.ContextEntry, error) {
	_ = ctx
	raw, ok := s.eng.Get(docKeyPrefix + id)
	if !ok {
		return storage.ContextEntry{}, fmt.Errorf("store get: not found")
	}
	ent, err := storage.Decode(raw)
	if err != nil {
		return storage.ContextEntry{}, err
	}
	return ent, nil
}

// Delete removes an entry and updates indexes.
func (s *Store) Delete(ctx context.Context, id string) error {
	_ = ctx
	key := docKeyPrefix + id
	if err := s.eng.Delete(key); err != nil {
		return err
	}
	s.ft.RemoveDocument(id)
	s.meta.Remove(id)
	s.vec.Delete(id)
	// chunk keys
	for i := 0; i < 1024; i++ {
		s.vec.Delete(fmt.Sprintf("%s#%d", id, i))
	}
	return nil
}

// SearchRequest is a hybrid query.
type SearchRequest struct {
	Query      string
	Collection string
	Filters    map[string]string
	TopK       int
	Detail     DetailLevel
}

// SearchResult is one ranked hit.
type SearchResult struct {
	ID                string  `json:"id"`
	Score             float64 `json:"score"`
	TokenCount        int     `json:"token_count"`
	CompactTokenCount int     `json:"compact_token_count"`
	TokensSaved       int     `json:"tokens_saved"`
}

// Search runs BM25 + vector RRF, optional metadata filter.
func (s *Store) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	_ = ctx
	if req.TopK <= 0 {
		req.TopK = 10
	}
	var ftHits []index.RankedID
	if req.Query != "" {
		ftHits = s.ft.Search(req.Query, req.TopK*2)
	}
	var vecHits []index.RankedID
	if s.embed != nil && req.Query != "" {
		qv, err := s.embed.Embed(ctx, req.Query)
		if err == nil {
			vecHits = s.vec.Search(qv, req.TopK*2)
		}
	}
	merged := index.RRF([][]index.RankedID{ftHits, vecHits}, req.TopK*2)
	var allow map[string]struct{}
	if len(req.Filters) > 0 {
		allow = s.meta.Match(req.Filters)
		if len(allow) == 0 {
			return nil, nil
		}
	}
	var out []SearchResult
	seen := make(map[string]struct{})
	for _, h := range merged {
		docID := h.ID
		if i := strings.Index(h.ID, "#"); i >= 0 {
			docID = h.ID[:i]
		}
		raw, ok := s.eng.Get(docKeyPrefix + docID)
		if !ok {
			continue
		}
		ent, err := storage.Decode(raw)
		if err != nil {
			continue
		}
		if req.Collection != "" && ent.Collection != req.Collection {
			continue
		}
		if allow != nil {
			if _, ok := allow[ent.ID]; !ok {
				continue
			}
		}
		if _, dup := seen[ent.ID]; dup {
			continue
		}
		seen[ent.ID] = struct{}{}
		out = append(out, SearchResult{
			ID:                ent.ID,
			Score:             h.Score,
			TokenCount:        ent.TokenCount,
			CompactTokenCount: ent.CompactTokenCount,
			TokensSaved:       ent.TokenCount - ent.CompactTokenCount,
		})
		if len(out) >= req.TopK {
			break
		}
	}
	return out, nil
}

// EntryListItem is a lightweight row for REST listing.
type EntryListItem struct {
	ID                string    `json:"id"`
	Collection        string    `json:"collection"`
	ContentType       string    `json:"content_type"`
	TokenCount        int       `json:"token_count"`
	CompactTokenCount int       `json:"compact_token_count"`
	TokensSaved       int       `json:"tokens_saved"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ListEntries returns entries newest-first, optionally filtered by collection.
func (s *Store) ListEntries(ctx context.Context, collection string, limit int) ([]EntryListItem, error) {
	_ = ctx
	if limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}
	keys := s.eng.PrefixKeys(docKeyPrefix)
	var out []EntryListItem
	for _, k := range keys {
		raw, ok := s.eng.Get(k)
		if !ok {
			continue
		}
		ent, err := storage.Decode(raw)
		if err != nil {
			continue
		}
		if collection != "" && ent.Collection != collection {
			continue
		}
		out = append(out, EntryListItem{
			ID:                ent.ID,
			Collection:        ent.Collection,
			ContentType:       ent.ContentType,
			TokenCount:        ent.TokenCount,
			CompactTokenCount: ent.CompactTokenCount,
			TokensSaved:       ent.TokenCount - ent.CompactTokenCount,
			CreatedAt:         ent.CreatedAt,
			UpdatedAt:         ent.UpdatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ListCollections returns distinct collection names.
func (s *Store) ListCollections(ctx context.Context) ([]string, error) {
	_ = ctx
	keys := s.eng.PrefixKeys(docKeyPrefix)
	seen := make(map[string]struct{})
	for _, k := range keys {
		raw, ok := s.eng.Get(k)
		if !ok {
			continue
		}
		ent, err := storage.Decode(raw)
		if err != nil {
			continue
		}
		if ent.Collection != "" {
			seen[ent.Collection] = struct{}{}
		}
	}
	var out []string
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out, nil
}

// Stats returns coarse database stats.
func (s *Store) Stats(ctx context.Context) (map[string]any, error) {
	_ = ctx
	keys := s.eng.PrefixKeys(docKeyPrefix)
	cols, _ := s.ListCollections(ctx)
	return map[string]any{
		"entries":      len(keys),
		"collections":  len(cols),
		"vector_nodes": s.vec.Len(),
	}, nil
}

// Engine exposes the underlying storage engine for admin (compaction).
func (s *Store) Engine() *storage.Engine { return s.eng }
