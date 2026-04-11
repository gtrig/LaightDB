---
name: lsm-storage-engine
description: Build an LSM-tree storage engine in Go with WAL, MemTable, SSTables, bloom filters, and compaction. Use when implementing or modifying internal/storage/ components, writing the WAL, MemTable, SSTable, compaction, or the storage engine orchestrator.
---

# LSM-Tree Storage Engine

## Implementation Order

1. WAL -> 2. MemTable -> 3. SSTable Writer -> 4. SSTable Reader -> 5. Engine (orchestrator) -> 6. Compaction

## WAL (`wal.go`)

Append-only binary log for crash recovery.

```go
type WAL struct {
    file *os.File
    mu   sync.Mutex
}

type EntryType byte
const (
    EntryPut    EntryType = 1
    EntryDelete EntryType = 2
)

func OpenWAL(path string) (*WAL, error) {
    f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
    // ...
}

func (w *WAL) Append(typ EntryType, key, value []byte) error {
    // Format: [4-byte len][4-byte CRC32][1-byte type][2-byte key_len][4-byte val_len][key][value]
    // Must call f.Sync() for durability
}

func (w *WAL) Replay(fn func(typ EntryType, key, value []byte) error) error {
    // Read entries sequentially, verify CRC, call fn for each
}
```

Key details:
- Use `binary.LittleEndian` for all integer encoding
- CRC32 covers type+key+value bytes (use `hash/crc32` IEEE polynomial)
- `Sync()` after every write for durability (optionally batch for throughput)
- Rotate: close current WAL, rename with sequence number, open new

## MemTable (`memtable.go`)

In-memory sorted data structure. Use a skip list for O(log n) insert/search.

```go
type MemTable struct {
    data     *SkipList  // sorted by key
    size     int64      // current byte size
    maxSize  int64      // flush threshold (default: 4MB)
}

func (m *MemTable) Put(key, value []byte)
func (m *MemTable) Get(key []byte) ([]byte, bool)
func (m *MemTable) Delete(key []byte)         // insert tombstone marker
func (m *MemTable) IsFull() bool              // size >= maxSize
func (m *MemTable) Iterator() Iterator        // sorted iteration for flushing
```

Skip list implementation: 32 max levels, probability 0.25, concurrent-safe with `sync.RWMutex`.

Tombstones: store a sentinel value (e.g., nil or special byte) to mark deletions. The tombstone propagates through SSTables and gets removed during compaction.

## SSTable (`sstable.go`)

Immutable sorted file on disk.

**Writer** -- flushes a MemTable iterator to disk:
```go
type SSTableWriter struct {
    file       *os.File
    index      []IndexEntry  // accumulated during write
    bloomBits  []byte
}

type IndexEntry struct {
    Key    []byte
    Offset int64
}

// Write sorted key-value pairs, then index block, then bloom filter, then footer
func (w *SSTableWriter) WriteEntry(key, value []byte) error
func (w *SSTableWriter) Finish() error  // writes index + bloom + footer
```

**Reader** -- reads from disk:
```go
type SSTableReader struct {
    file   *os.File
    index  []IndexEntry  // loaded from file footer
    bloom  BloomFilter
}

func OpenSSTable(path string) (*SSTableReader, error) {
    // 1. Seek to end - 16 (footer size)
    // 2. Read index_offset and bloom_offset
    // 3. Load index block and bloom filter into memory
}

func (r *SSTableReader) Get(key []byte) ([]byte, error) {
    // 1. Check bloom filter (fast negative)
    // 2. Binary search index for block containing key
    // 3. Scan block for exact key match
}
```

**Bloom Filter**: use `k=10` hash functions, `m = n * 10` bits (1% false positive rate). Implement with double hashing: `h(i) = h1 + i*h2`.

## Engine (`engine.go`)

Orchestrates all components:

```go
type Engine struct {
    dir          string
    wal          *WAL
    memtable     *MemTable
    immutable    []*MemTable  // memtables being flushed
    sstables     []*SSTableReader  // newest first
    mu           sync.RWMutex
    compactCh    chan struct{}
    log          *slog.Logger
}

func (e *Engine) Put(ctx context.Context, key, value []byte) error {
    // 1. Write to WAL
    // 2. Insert into MemTable
    // 3. If MemTable full: move to immutable, start flush goroutine, create new MemTable+WAL
}

func (e *Engine) Get(ctx context.Context, key []byte) ([]byte, error) {
    // 1. Check active MemTable
    // 2. Check immutable MemTables (newest first)
    // 3. Check SSTables (newest first, bloom filter skip)
    // Return ErrNotFound if nowhere
}
```

## Compaction (`compaction.go`)

Background process merging SSTables:

```go
func (e *Engine) runCompaction(ctx context.Context) {
    // Triggered when SSTable count exceeds threshold
    // 1. Select SSTables to merge (size-tiered: group by similar size)
    // 2. K-way merge using min-heap on keys
    // 3. Write merged output to new SSTable
    // 4. Atomically swap: remove old SSTables, add new one
    // 5. Delete old SSTable files
}
```

Use `container/heap` for the K-way merge priority queue.

## Additional Reference

- For detailed LSM-tree theory, see [reference.md](reference.md)
