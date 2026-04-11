# LSM-Tree Reference

## Write Amplification

Each byte written to the MemTable may be written to disk multiple times through flushes and compaction levels. Size-tiered compaction has write amplification ~O(T * levels) where T is the size ratio. Acceptable for write-heavy workloads.

## Read Amplification

Worst case: check MemTable + all SSTables. Mitigated by:
- Bloom filters: skip SSTables that definitely don't contain the key
- Index blocks: binary search within SSTables (avoid full scan)
- Compaction: fewer SSTables = fewer checks

## Space Amplification

Tombstones occupy space until compacted away. Multiple versions of the same key exist across SSTables until merged. Size-tiered compaction: worst case ~2x space during merge.

## File Naming Convention

```
data/
  wal/
    000001.wal        # active WAL
    000002.wal        # rotated WAL (being flushed)
  sst/
    000001.sst        # oldest SSTable
    000002.sst
    000003.sst        # newest SSTable
```

Use monotonically increasing sequence numbers. Store a MANIFEST file tracking active SSTables and their metadata.

## Recovery Procedure

1. Read MANIFEST to identify active SSTables
2. Open all listed SSTables
3. Find WAL files not yet flushed (by sequence number)
4. Replay WAL entries into new MemTable
5. Resume normal operation

## Concurrency

- Reads: `sync.RWMutex` -- multiple concurrent readers allowed
- Writes: serialize through WAL (single writer) then MemTable (which can use fine-grained locking)
- Flush: runs in separate goroutine, swaps atomic pointer from active to immutable MemTable
- Compaction: runs in background, atomically swaps SSTable list when done

## Tuning Parameters

| Parameter | Default | Effect |
|---|---|---|
| MemTable size | 4 MB | Larger = fewer flushes, more RAM |
| Bloom filter bits per key | 10 | Higher = fewer false positives, more memory |
| SSTable block size | 4 KB | Larger = better compression, worse point lookups |
| Compaction trigger | 4 SSTables | Lower = more compaction work, faster reads |
| Compaction size ratio | 4 | Lower = more levels, less space amplification |
