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
	docKeyPrefix      = "d:"
	edgeKeyPrefix     = "e:"
	edgeFwdKeyPrefix  = "ef:"
	edgeRevKeyPrefix  = "et:"
	ftSnapshotFile    = "fulltext.snap"
	metaSnapshotFile  = "metadata.snap"
	graphSnapshotFile = "graph.snap"
)

// Store is the application facade over storage + indexes.
type Store struct {
	dir   string
	eng   *storage.Engine
	ft    *index.FullText
	meta  *index.MetadataIndex
	vec   *index.VectorIndex
	graph *index.GraphIndex
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
		graph: index.NewGraphIndex(),
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

// loadSnapshots attempts to restore BM25, metadata, and graph indexes from snapshot files.
// Returns true if all loaded successfully and are consistent with the engine.
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
		// Load graph snapshot if present; ignore errors (graph is rebuilt from LSM if missing).
		if g := s.tryLoadGraphSnapshot(); g != nil {
			s.graph = g
		}
		return true
	}
	// Non-empty engine but empty snapshot → snapshot is stale, rebuild.
	if engineDocCount > 0 && ft.N == 0 {
		return false
	}
	s.ft = ft
	s.meta = meta
	if g := s.tryLoadGraphSnapshot(); g != nil {
		s.graph = g
	}
	return true
}

func (s *Store) tryLoadGraphSnapshot() *index.GraphIndex {
	data, err := os.ReadFile(filepath.Join(s.dir, graphSnapshotFile))
	if err != nil {
		return nil
	}
	g, err := index.DecodeGraphSnapshot(data)
	if err != nil {
		return nil
	}
	return g
}

// saveSnapshots writes BM25, metadata, and graph index snapshots to disk.
func (s *Store) saveSnapshots() {
	_ = os.WriteFile(filepath.Join(s.dir, ftSnapshotFile), s.ft.EncodeSnapshot(), 0o644)
	_ = os.WriteFile(filepath.Join(s.dir, metaSnapshotFile), s.meta.EncodeSnapshot(), 0o644)
	_ = os.WriteFile(filepath.Join(s.dir, graphSnapshotFile), s.graph.EncodeSnapshot(), 0o644)
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
	// Rebuild graph index from edge LSM keys.
	for _, k := range s.eng.PrefixKeys(edgeKeyPrefix) {
		raw, ok := s.eng.Get(k)
		if !ok {
			continue
		}
		e, err := storage.DecodeEdge(raw)
		if err != nil {
			continue
		}
		s.graph.Add(e.ID, e.FromID, e.ToID, e.Label, e.Source, e.Weight)
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
	Query       string
	Collection  string
	Filters     map[string]string
	TopK        int
	Detail      DetailLevel
	FocusNodeID string // optional: boosts graph-proximity of this node via RRF
	MaxDepth    int    // graph BFS depth when FocusNodeID is set (0 = unlimited)
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
	// Third signal: graph proximity from a focus node.
	var graphHits []index.RankedID
	if req.FocusNodeID != "" {
		graphHits = s.graph.Neighborhood(req.FocusNodeID, req.MaxDepth)
		if allow != nil {
			graphHits = filterHitsByDocID(graphHits, allow)
		}
	}
	merged := index.RRF([][]index.RankedID{ftHits, vecHits, graphHits}, req.TopK*2)

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
		"edges":        s.graph.Len(),
	}, nil
}

// Engine exposes the underlying storage engine for admin (compaction).
func (s *Store) Engine() *storage.Engine { return s.eng }

// -------------------------------------------------------------------
// Edge / Graph API
// -------------------------------------------------------------------

// PutEdgeRequest is the input to PutEdge.
type PutEdgeRequest struct {
	FromID   string
	ToID     string
	Label    string
	Weight   float64
	Source   string
	Metadata map[string]string
}

// PutEdge creates a new directed edge and returns its assigned ID.
// It writes three LSM keys atomically (best-effort; the canonical e: key is written first).
func (s *Store) PutEdge(ctx context.Context, req PutEdgeRequest) (string, error) {
	_ = ctx
	if req.FromID == "" || req.ToID == "" {
		return "", fmt.Errorf("store put edge: FromID and ToID required")
	}
	if req.Source == "" {
		req.Source = "user"
	}
	id := uuid.New().String()
	e := storage.Edge{
		ID:        id,
		FromID:    req.FromID,
		ToID:      req.ToID,
		Label:     req.Label,
		Weight:    req.Weight,
		Source:    req.Source,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
	}
	data := storage.EncodeEdge(e)
	// Canonical record.
	if err := s.eng.Put(edgeKeyPrefix+id, data); err != nil {
		return "", fmt.Errorf("store put edge: %w", err)
	}
	// Forward adjacency index: ef:<fromID>:<edgeID>
	if err := s.eng.Put(edgeFwdKeyPrefix+req.FromID+":"+id, []byte(id)); err != nil {
		return "", fmt.Errorf("store put edge fwd: %w", err)
	}
	// Reverse adjacency index: et:<toID>:<edgeID>
	if err := s.eng.Put(edgeRevKeyPrefix+req.ToID+":"+id, []byte(id)); err != nil {
		return "", fmt.Errorf("store put edge rev: %w", err)
	}
	s.graph.Add(id, req.FromID, req.ToID, req.Label, req.Source, req.Weight)
	return id, nil
}

// GetEdge retrieves a single edge by ID.
func (s *Store) GetEdge(ctx context.Context, id string) (storage.Edge, error) {
	_ = ctx
	raw, ok := s.eng.Get(edgeKeyPrefix + id)
	if !ok {
		return storage.Edge{}, fmt.Errorf("store get edge: not found")
	}
	return storage.DecodeEdge(raw)
}

// DeleteEdge removes an edge and its adjacency index entries.
func (s *Store) DeleteEdge(ctx context.Context, id string) error {
	_ = ctx
	raw, ok := s.eng.Get(edgeKeyPrefix + id)
	if !ok {
		return fmt.Errorf("store delete edge: not found")
	}
	e, err := storage.DecodeEdge(raw)
	if err != nil {
		return fmt.Errorf("store delete edge decode: %w", err)
	}
	if err := s.eng.Delete(edgeKeyPrefix + id); err != nil {
		return fmt.Errorf("store delete edge: %w", err)
	}
	_ = s.eng.Delete(edgeFwdKeyPrefix + e.FromID + ":" + id)
	_ = s.eng.Delete(edgeRevKeyPrefix + e.ToID + ":" + id)
	s.graph.Remove(id)
	return nil
}

// ListEdgesFrom returns all edges whose FromID equals nodeID.
func (s *Store) ListEdgesFrom(ctx context.Context, nodeID string) ([]storage.Edge, error) {
	_ = ctx
	prefix := edgeFwdKeyPrefix + nodeID + ":"
	keys := s.eng.PrefixKeys(prefix)
	return s.loadEdgesByKeys(keys)
}

// ListEdgesTo returns all edges whose ToID equals nodeID.
func (s *Store) ListEdgesTo(ctx context.Context, nodeID string) ([]storage.Edge, error) {
	_ = ctx
	prefix := edgeRevKeyPrefix + nodeID + ":"
	keys := s.eng.PrefixKeys(prefix)
	return s.loadEdgesByKeys(keys)
}

func (s *Store) loadEdgesByKeys(adjKeys []string) ([]storage.Edge, error) {
	var out []storage.Edge
	for _, k := range adjKeys {
		// Value is the edge ID.
		idBytes, ok := s.eng.Get(k)
		if !ok {
			continue
		}
		edgeID := string(idBytes)
		raw, ok := s.eng.Get(edgeKeyPrefix + edgeID)
		if !ok {
			continue
		}
		e, err := storage.DecodeEdge(raw)
		if err != nil {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

// GraphOutgoing returns the GraphIndex outgoing EdgeRefs for fast in-memory traversal.
func (s *Store) GraphOutgoing(nodeID string) []index.EdgeRef {
	return s.graph.Outgoing(nodeID)
}

// GraphIncoming returns the GraphIndex incoming EdgeRefs.
func (s *Store) GraphIncoming(nodeID string) []index.EdgeRef {
	return s.graph.Incoming(nodeID)
}

// GraphNeighborhood returns nodes reachable from nodeID via BFS (both directions)
// up to maxDepth, scored by graph proximity. Used by REST and MCP handlers.
func (s *Store) GraphNeighborhood(nodeID string, maxDepth int) []index.RankedID {
	return s.graph.Neighborhood(nodeID, maxDepth)
}

// SubtreeEdges returns edges in a directed BFS from rootID following outgoing edges.
func (s *Store) SubtreeEdges(rootID string, maxDepth int) []index.EdgeRef {
	return s.graph.SubtreeEdges(rootID, maxDepth)
}

// SuggestedLink is a candidate relationship discovered via vector similarity.
type SuggestedLink struct {
	TargetID   string  `json:"target_id"`
	Similarity float64 `json:"similarity"`
}

// SuggestLinks finds the nearest vector neighbors of nodeID that are not already
// linked via an explicit edge, above the given similarity threshold.
func (s *Store) SuggestLinks(ctx context.Context, nodeID string, threshold float64, topK int) ([]SuggestedLink, error) {
	raw, ok := s.eng.Get(docKeyPrefix + nodeID)
	if !ok {
		return nil, fmt.Errorf("store suggest links: node not found")
	}
	if s.embed == nil {
		return nil, nil
	}
	ent, err := storage.Decode(raw)
	if err != nil {
		return nil, fmt.Errorf("store suggest links decode: %w", err)
	}
	if len(ent.Embedding) == 0 {
		return nil, nil
	}
	if topK <= 0 {
		topK = 10
	}
	vecHits := s.vec.Search(ent.Embedding, topK*3)
	existing := s.graph.AllNeighborIDs(nodeID)
	var out []SuggestedLink
	for _, h := range vecHits {
		// Strip chunk suffix if present.
		docID := h.ID
		if i := strings.Index(h.ID, "#"); i >= 0 {
			docID = h.ID[:i]
		}
		if docID == nodeID {
			continue
		}
		if _, linked := existing[docID]; linked {
			continue
		}
		if h.Score < threshold {
			continue
		}
		out = append(out, SuggestedLink{TargetID: docID, Similarity: h.Score})
		if len(out) >= topK {
			break
		}
	}
	return out, nil
}
