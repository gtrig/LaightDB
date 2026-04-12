package context

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gtrig/laightdb/internal/embedding"
	"github.com/gtrig/laightdb/internal/index"
	"github.com/gtrig/laightdb/internal/storage"
	"github.com/gtrig/laightdb/internal/summarize"
)

const (
	docKeyPrefix     = "d:"
	ftSnapshotFile   = "fulltext.snap"
	metaSnapshotFile = "metadata.snap"
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
	if !s.loadSnapshots() {
		if err := s.rebuildIndexes(); err != nil {
			_ = eng.Close()
			return nil, err
		}
	}
	return s, nil
}

// loadSnapshots attempts to restore BM25 and metadata indexes from snapshot files.
// Returns true if both loaded successfully and are consistent with the engine.
func (s *Store) loadSnapshots() bool {
	ftData, err := os.ReadFile(filepath.Join(s.dir, ftSnapshotFile))
	if err != nil {
		return false
	}
	ft, err := index.DecodeSnapshot(ftData)
	if err != nil {
		return false
	}
	metaData, err := os.ReadFile(filepath.Join(s.dir, metaSnapshotFile))
	if err != nil {
		return false
	}
	meta, err := index.DecodeMetadataSnapshot(metaData)
	if err != nil {
		return false
	}
	// Validate: count base doc keys in the engine and compare to BM25's base doc count.
	// ft.N counts all indexed IDs (base docs + chunk IDs). We count only base docs in engine
	// to check for consistency, but since chunk counts vary, we use a simpler heuristic:
	// if ft.N > 0 and there are any base docs in engine, trust the snapshot. A full mismatch
	// (e.g. after a crash mid-write) will be caught at query time as missing engine entries.
	keys := s.eng.PrefixKeys(docKeyPrefix)
	engineDocCount := 0
	for _, k := range keys {
		if _, ok := s.eng.Get(k); ok {
			engineDocCount++
		}
	}
	// Empty engine → no snapshots needed; fresh rebuild is instant.
	if engineDocCount == 0 && ft.N == 0 {
		s.ft = ft
		s.meta = meta
		return true
	}
	// Non-empty engine but empty snapshot → snapshot is stale, rebuild.
	if engineDocCount > 0 && ft.N == 0 {
		return false
	}
	s.ft = ft
	s.meta = meta
	return true
}

// saveSnapshots writes BM25 and metadata index snapshots to disk.
func (s *Store) saveSnapshots() {
	_ = os.WriteFile(filepath.Join(s.dir, ftSnapshotFile), s.ft.EncodeSnapshot(), 0o644)
	_ = os.WriteFile(filepath.Join(s.dir, metaSnapshotFile), s.meta.EncodeSnapshot(), 0o644)
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
		for _, ch := range ent.Chunks {
			s.ft.AddDocument(fmt.Sprintf("%s#%d", ent.ID, ch.Index), ch.Text)
		}
		s.meta.Set(ent.ID, ent.Metadata)
	}
	return nil
}

// Close saves index snapshots and releases resources.
func (s *Store) Close() error {
	s.saveSnapshots()
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
// If identical content was already stored, returns the existing entry's ID (idempotent).
func (s *Store) Put(ctx context.Context, req PutRequest) (string, error) {
	if req.Content == "" {
		return "", fmt.Errorf("store put: empty content")
	}
	// Exact-content deduplication: return existing ID without re-indexing.
	if dup, found := FindHashDuplicate(s.eng.Get, req.Content); found {
		return dup.ExistingID, nil
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
	// Record content hash for future dedup lookups.
	if err := RecordHash(s.eng.Put, req.Content, id); err != nil {
		return "", fmt.Errorf("store put hash: %w", err)
	}
	// Index the full document text and each chunk for BM25.
	s.ft.AddDocument(id, req.Content)
	for _, ch := range chunks {
		s.ft.AddDocument(fmt.Sprintf("%s#%d", id, ch.Index), ch.Text)
	}
	s.meta.Set(id, req.Metadata)
	// Batch all vector upserts to a single disk write.
	var vecItems []index.VectorItem
	if s.embed != nil && len(emb) > 0 {
		vecItems = append(vecItems, index.VectorItem{ID: id, Vec: emb})
	}
	for _, ch := range chunks {
		if len(ch.Embedding) > 0 {
			vecItems = append(vecItems, index.VectorItem{
				ID:  fmt.Sprintf("%s#%d", id, ch.Index),
				Vec: ch.Embedding,
			})
		}
	}
	if len(vecItems) > 0 {
		if err := s.vec.UpsertBatch(vecItems); err != nil {
			return "", err
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
	// Remove chunk BM25 docs and chunk vectors.
	for i := 0; i < 1024; i++ {
		cid := fmt.Sprintf("%s#%d", id, i)
		s.ft.RemoveDocument(cid)
		s.vec.Delete(cid)
	}
	return nil
}

// DeleteCollection removes all entries belonging to a collection and returns the count deleted.
func (s *Store) DeleteCollection(ctx context.Context, collection string) (int, error) {
	keys := s.eng.PrefixKeys(docKeyPrefix)
	var deleted int
	for _, k := range keys {
		raw, ok := s.eng.Get(k)
		if !ok {
			continue
		}
		ent, err := storage.Decode(raw)
		if err != nil {
			continue
		}
		if ent.Collection != collection {
			continue
		}
		if err := s.Delete(ctx, ent.ID); err != nil {
			return deleted, fmt.Errorf("delete collection entry %s: %w", ent.ID, err)
		}
		deleted++
	}
	return deleted, nil
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
	ID                string         `json:"id"`
	Score             float64        `json:"score"`
	TokenCount        int            `json:"token_count"`
	CompactTokenCount int            `json:"compact_token_count"`
	TokensSaved       int            `json:"tokens_saved"`
	BestChunkIndex    int            `json:"best_chunk_index"`
	Entry             map[string]any `json:"entry,omitempty"`
}

// Search runs BM25 + vector RRF with optional metadata pre-filter and detail projection.
func (s *Store) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	if req.TopK <= 0 {
		req.TopK = 10
	}

	// P3: compute metadata allow-set before ranking to avoid wasting RRF slots.
	var allow map[string]struct{}
	if len(req.Filters) > 0 {
		allow = s.meta.Match(req.Filters)
		if len(allow) == 0 {
			return nil, nil
		}
	}

	var ftHits []index.RankedID
	if req.Query != "" {
		ftHits = s.ft.Search(req.Query, req.TopK*2)
		if allow != nil {
			ftHits = filterHitsByDocID(ftHits, allow)
		}
	}
	var vecHits []index.RankedID
	if s.embed != nil && req.Query != "" {
		qv, err := s.embed.Embed(ctx, req.Query)
		if err == nil {
			vecHits = s.vec.Search(qv, req.TopK*2)
			if allow != nil {
				vecHits = filterHitsByDocID(vecHits, allow)
			}
		}
	}
	merged := index.RRF([][]index.RankedID{ftHits, vecHits}, req.TopK*2)

	var out []SearchResult
	seen := make(map[string]struct{})
	for _, h := range merged {
		// P1: parse chunk suffix to identify which chunk was the best match.
		docID := h.ID
		chunkIdx := -1
		if i := strings.Index(h.ID, "#"); i >= 0 {
			docID = h.ID[:i]
			if n, err := strconv.Atoi(h.ID[i+1:]); err == nil {
				chunkIdx = n
			}
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
		if _, dup := seen[ent.ID]; dup {
			continue
		}
		seen[ent.ID] = struct{}{}
		sr := SearchResult{
			ID:                ent.ID,
			Score:             h.Score,
			TokenCount:        ent.TokenCount,
			CompactTokenCount: ent.CompactTokenCount,
			TokensSaved:       ent.TokenCount - ent.CompactTokenCount,
			BestChunkIndex:    chunkIdx,
		}
		// P0: attach projected entry payload when the caller requests it.
		if req.Detail != "" {
			sr.Entry = Project(ent, req.Detail)
		}
		out = append(out, sr)
		if len(out) >= req.TopK {
			break
		}
	}
	return out, nil
}

// filterHitsByDocID keeps only hits whose base document ID is in allow.
// Chunk IDs of the form "docID#N" are mapped to their base doc ID for matching.
func filterHitsByDocID(hits []index.RankedID, allow map[string]struct{}) []index.RankedID {
	out := make([]index.RankedID, 0, len(hits))
	for _, h := range hits {
		docID := h.ID
		if i := strings.Index(h.ID, "#"); i >= 0 {
			docID = h.ID[:i]
		}
		if _, ok := allow[docID]; ok {
			out = append(out, h)
		}
	}
	return out
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
	var count int
	for _, k := range keys {
		if _, ok := s.eng.Get(k); ok {
			count++
		}
	}
	cols, _ := s.ListCollections(ctx)
	return map[string]any{
		"entries":      count,
		"collections":  len(cols),
		"vector_nodes": s.vec.Len(),
	}, nil
}

// Engine exposes the underlying storage engine for admin (compaction).
func (s *Store) Engine() *storage.Engine { return s.eng }
