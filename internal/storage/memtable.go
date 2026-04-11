package storage

// tombstone is the value used for deleted keys inside the memtable / SST merge.
var tombstone = []byte{0}

// MemTable is an in-memory sorted table with a size budget.
type MemTable struct {
	sl       *SkipList
	maxBytes int
}

// NewMemTable creates a memtable with flush threshold maxBytes.
func NewMemTable(maxBytes int) *MemTable {
	if maxBytes < 1024 {
		maxBytes = 4 << 20
	}
	return &MemTable{sl: NewSkipList(), maxBytes: maxBytes}
}

// Put inserts or replaces a value.
func (m *MemTable) Put(key string, value []byte) {
	m.sl.Put(key, value)
}

// Delete marks key as deleted.
func (m *MemTable) Delete(key string) {
	m.sl.Put(key, append([]byte(nil), tombstone...))
}

// Get returns value or nil if missing or tombstoned.
func (m *MemTable) Get(key string) ([]byte, bool) {
	v, ok := m.sl.Get(key)
	if !ok {
		return nil, false
	}
	if isTombstone(v) {
		return nil, false
	}
	return v, true
}

// GetRaw returns the stored bytes including tombstones.
func (m *MemTable) GetRaw(key string) ([]byte, bool) {
	return m.sl.Get(key)
}

// IsTombstone reports whether key is present as a delete marker.
func (m *MemTable) IsTombstone(key string) bool {
	v, ok := m.sl.Get(key)
	return ok && isTombstone(v)
}

// ShouldFlush reports whether estimated size exceeds threshold.
func (m *MemTable) ShouldFlush() bool {
	return m.sl.ByteSize() >= m.maxBytes
}

// ApproxSize returns byte estimate.
func (m *MemTable) ApproxSize() int { return m.sl.ByteSize() }

// Len returns entry count including tombstones.
func (m *MemTable) Len() int { return m.sl.Len() }

// Scan iterates keys in range.
func (m *MemTable) Scan(start, end string, fn func(key string, value []byte) bool) {
	m.sl.Scan(start, end, func(key string, value []byte) bool {
		if len(value) == 1 && value[0] == 0 {
			return true
		}
		return fn(key, value)
	})
}

// RawScan iterates all keys including tombstones (for flush).
func (m *MemTable) RawScan(fn func(key string, value []byte) bool) {
	m.sl.Scan("", "", fn)
}

// Reset clears the memtable.
func (m *MemTable) Reset() {
	m.sl = NewSkipList()
}

func isTombstone(v []byte) bool {
	return len(v) == 1 && v[0] == 0
}
