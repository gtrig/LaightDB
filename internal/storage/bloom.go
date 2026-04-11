package storage

import (
	"encoding/binary"
	"hash/fnv"
)

const bloomK = 10

// BloomFilter is a compact probabilistic set for keys (may false positive, never false negative).
type BloomFilter struct {
	bits []byte
	m    uint32 // bits length
	n    uint32 // number of keys added (for stats)
}

// NewBloomFilter creates a filter sized for expected n keys and false positive rate ~0.01.
func NewBloomFilter(expectedKeys int) *BloomFilter {
	if expectedKeys < 1 {
		expectedKeys = 1
	}
	// m bits, k=10: m ≈ -n*ln(p)/(ln2)^2 with p small
	m := uint32(expectedKeys * 16)
	if m < 64 {
		m = 64
	}
	m = (m + 7) &^ 7
	return &BloomFilter{bits: make([]byte, m/8), m: m}
}

func hashPair(key string) (h1, h2 uint64) {
	h := fnv.New128a()
	_, _ = h.Write([]byte(key))
	sum := h.Sum(nil)
	h1 = binary.LittleEndian.Uint64(sum[:8])
	h2 = binary.LittleEndian.Uint64(sum[8:16])
	if h2 == 0 {
		h2 = 0x9e3779b97f4a7c15
	}
	return h1, h2
}

// Add inserts a key into the set.
func (b *BloomFilter) Add(key string) {
	h1, h2 := hashPair(key)
	n := uint64(b.m)
	for i := uint32(0); i < bloomK; i++ {
		idx := (h1 + uint64(i)*h2) % n
		byteIdx := idx / 8
		bitIdx := idx % 8
		b.bits[byteIdx] |= 1 << bitIdx
	}
	b.n++
}

// MaybeContains returns false if key is definitely absent.
func (b *BloomFilter) MaybeContains(key string) bool {
	h1, h2 := hashPair(key)
	n := uint64(b.m)
	for i := uint32(0); i < bloomK; i++ {
		idx := (h1 + uint64(i)*h2) % n
		byteIdx := idx / 8
		bitIdx := idx % 8
		if b.bits[byteIdx]&(1<<bitIdx) == 0 {
			return false
		}
	}
	return true
}

// Encode serializes the bloom filter.
func (b *BloomFilter) Encode() []byte {
	out := make([]byte, 8+len(b.bits))
	binary.LittleEndian.PutUint32(out[0:4], b.m)
	binary.LittleEndian.PutUint32(out[4:8], b.n)
	copy(out[8:], b.bits)
	return out
}

// DecodeBloomFilter restores from Encode output.
func DecodeBloomFilter(data []byte) (*BloomFilter, error) {
	if len(data) < 8 {
		return nil, errBloomCorrupt
	}
	m := binary.LittleEndian.Uint32(data[0:4])
	n := binary.LittleEndian.Uint32(data[4:8])
	rest := data[8:]
	if m%8 != 0 || uint32(len(rest))*8 != m {
		return nil, errBloomCorrupt
	}
	return &BloomFilter{bits: append([]byte(nil), rest...), m: m, n: n}, nil
}
