package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Engine coordinates WAL, memtable, and SSTables on disk.
type Engine struct {
	mu         sync.RWMutex
	dir        string
	wal        *WAL
	mem        *MemTable
	sstPaths   []string // oldest first
	nextSSTSeq uint64
}

// OpenEngine opens or creates a database in dir.
func OpenEngine(dir string, memtableMaxBytes int) (*Engine, error) {
	if err := os.MkdirAll(filepath.Join(dir, "sst"), 0o755); err != nil {
		return nil, fmt.Errorf("engine mkdir: %w", err)
	}
	walPath := filepath.Join(dir, "wal.log")
	w, err := OpenWAL(walPath)
	if err != nil {
		return nil, err
	}
	e := &Engine{
		dir: dir,
		wal: w,
		mem: NewMemTable(memtableMaxBytes),
	}
	if err := e.loadSSTList(); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := e.replayWAL(); err != nil {
		_ = w.Close()
		return nil, err
	}
	return e, nil
}

func (e *Engine) loadSSTList() error {
	pattern := filepath.Join(e.dir, "sst", "*.sst")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("engine glob: %w", err)
	}
	sort.Strings(matches)
	e.sstPaths = matches
	var max uint64
	for _, p := range matches {
		base := filepath.Base(p)
		base = strings.TrimSuffix(base, ".sst")
		n, err := strconv.ParseUint(base, 10, 64)
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	e.nextSSTSeq = max + 1
	return nil
}

func (e *Engine) replayWAL() error {
	return e.wal.Replay(func(typ byte, key string, value []byte) error {
		switch typ {
		case walTypePut:
			e.mem.Put(key, value)
		case walTypeDelete:
			e.mem.Delete(key)
		}
		return nil
	})
}

// Close flushes resources.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.wal != nil {
		err := e.wal.Close()
		e.wal = nil
		return err
	}
	return nil
}

// Get returns the value for key from memtable or SSTables (newest wins).
func (e *Engine) Get(key string) ([]byte, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.mem.IsTombstone(key) {
		return nil, false
	}
	if v, ok := e.mem.Get(key); ok {
		return v, true
	}
	for i := len(e.sstPaths) - 1; i >= 0; i-- {
		r, err := OpenSSTReader(e.sstPaths[i])
		if err != nil {
			continue
		}
		v, ok := r.Get(key)
		if ok {
			if isTombstone(v) {
				return nil, false
			}
			return v, true
		}
	}
	return nil, false
}

// Put stores key/value.
func (e *Engine) Put(key string, value []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.wal.AppendPut(key, value); err != nil {
		return fmt.Errorf("engine wal put: %w", err)
	}
	if err := e.wal.Sync(); err != nil {
		return fmt.Errorf("engine wal sync: %w", err)
	}
	e.mem.Put(key, value)
	if e.mem.ShouldFlush() {
		if err := e.flushLocked(); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a key.
func (e *Engine) Delete(key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.wal.AppendDelete(key); err != nil {
		return fmt.Errorf("engine wal del: %w", err)
	}
	if err := e.wal.Sync(); err != nil {
		return fmt.Errorf("engine wal sync: %w", err)
	}
	e.mem.Delete(key)
	return nil
}

// KV is a single key/value pair for PutBatch.
type KV struct {
	Key string
	Val []byte
}

// PutBatch appends multiple puts to the WAL, syncs once, then applies all to the memtable.
// If a flush is needed mid-batch, flushLocked runs between keys; all WAL records are durable after the first sync.
func (e *Engine) PutBatch(pairs []KV) error {
	if len(pairs) == 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range pairs {
		if err := e.wal.AppendPut(pairs[i].Key, pairs[i].Val); err != nil {
			return fmt.Errorf("engine wal put: %w", err)
		}
	}
	if err := e.wal.Sync(); err != nil {
		return fmt.Errorf("engine wal sync: %w", err)
	}
	for i := range pairs {
		e.mem.Put(pairs[i].Key, pairs[i].Val)
		if e.mem.ShouldFlush() {
			if err := e.flushLocked(); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteBatch appends multiple deletes to the WAL, syncs once, then applies all to the memtable.
func (e *Engine) DeleteBatch(keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, k := range keys {
		if err := e.wal.AppendDelete(k); err != nil {
			return fmt.Errorf("engine wal del: %w", err)
		}
	}
	if err := e.wal.Sync(); err != nil {
		return fmt.Errorf("engine wal sync: %w", err)
	}
	for _, k := range keys {
		e.mem.Delete(k)
	}
	return nil
}

func (e *Engine) flushLocked() error {
	path := filepath.Join(e.dir, "sst", fmt.Sprintf("%06d.sst", e.nextSSTSeq))
	w, err := NewSSTWriter(path)
	if err != nil {
		return err
	}
	var keys []string
	e.mem.RawScan(func(key string, value []byte) bool {
		keys = append(keys, key)
		return true
	})
	sort.Strings(keys)
	for _, k := range keys {
		v, ok := e.mem.GetRaw(k)
		if !ok {
			continue
		}
		if err := w.Append(k, v); err != nil {
			_ = w.Close()
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	e.sstPaths = append(e.sstPaths, path)
	sort.Strings(e.sstPaths)
	e.nextSSTSeq++
	e.mem.Reset()
	if err := e.wal.Truncate(); err != nil {
		return fmt.Errorf("engine wal truncate: %w", err)
	}
	if err := e.wal.Sync(); err != nil {
		return fmt.Errorf("engine wal sync: %w", err)
	}
	return nil
}

// Flush forces a memtable flush if non-empty.
func (e *Engine) Flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.mem.Len() == 0 {
		return nil
	}
	return e.flushLocked()
}

// SSTPaths returns current SST files (oldest first).
func (e *Engine) SSTPaths() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, len(e.sstPaths))
	copy(out, e.sstPaths)
	return out
}

// MemLen exposes memtable entry count (for tests).
func (e *Engine) MemLen() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.mem.Len()
}

// SSTFileInfo holds size metadata for a single SSTable file.
type SSTFileInfo struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Seq   uint64 `json:"seq"`
}

// EngineDiagnostics contains a snapshot of storage layer sizes.
type EngineDiagnostics struct {
	DataDir    string        `json:"data_dir"`
	WALBytes   int64         `json:"wal_bytes"`
	MemEntries int           `json:"mem_entries"`
	SSTables   []SSTFileInfo `json:"sstables"`
}

// Diagnostics gathers WAL size, memtable count, and per-SSTable sizes.
// It reads the filesystem while holding only the read lock.
func (e *Engine) Diagnostics() (EngineDiagnostics, error) {
	e.mu.RLock()
	paths := make([]string, len(e.sstPaths))
	copy(paths, e.sstPaths)
	memEntries := e.mem.Len()
	dir := e.dir
	e.mu.RUnlock()

	diag := EngineDiagnostics{
		DataDir:    dir,
		MemEntries: memEntries,
	}

	// WAL size.
	walPath := filepath.Join(dir, "wal.log")
	if fi, err := os.Stat(walPath); err == nil {
		diag.WALBytes = fi.Size()
	}

	// SSTable sizes.
	for _, p := range paths {
		info := SSTFileInfo{Path: p}
		if fi, err := os.Stat(p); err == nil {
			info.Bytes = fi.Size()
		}
		// Parse sequence number from filename.
		base := strings.TrimSuffix(filepath.Base(p), ".sst")
		if n, err := strconv.ParseUint(base, 10, 64); err == nil {
			info.Seq = n
		}
		diag.SSTables = append(diag.SSTables, info)
	}
	if diag.SSTables == nil {
		diag.SSTables = []SSTFileInfo{}
	}
	return diag, nil
}
