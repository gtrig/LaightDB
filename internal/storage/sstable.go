package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

var magicSST = []byte("LAdbSST1")

// SSTWriter writes a sorted immutable table.
type SSTWriter struct {
	path   string
	f      *os.File
	keys   []string
	offset []uint64
	pos    int64
}

// NewSSTWriter creates a writer; keys must be appended in sorted order.
func NewSSTWriter(path string) (*SSTWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("sst create: %w", err)
	}
	if _, err := f.Write(magicSST); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &SSTWriter{path: path, f: f, pos: int64(len(magicSST))}, nil
}

// Append writes one key/value pair (keys must be strictly increasing).
func (w *SSTWriter) Append(key string, value []byte) error {
	if len(w.keys) > 0 && key <= w.keys[len(w.keys)-1] {
		return fmt.Errorf("sst: keys not sorted")
	}
	off := w.pos
	w.keys = append(w.keys, key)
	w.offset = append(w.offset, uint64(off))
	rec := encodeKV(key, value)
	n, err := w.f.Write(rec)
	if err != nil {
		return err
	}
	w.pos += int64(n)
	return nil
}

func encodeKV(key string, value []byte) []byte {
	var buf []byte
	buf = binary.AppendUvarint(buf, uint64(len(key)))
	buf = append(buf, key...)
	buf = binary.AppendUvarint(buf, uint64(len(value)))
	buf = append(buf, value...)
	return buf
}

// Close finalizes index and bloom in the file.
func (w *SSTWriter) Close() error {
	if w.f == nil {
		return nil
	}
	dataEnd := w.pos
	bf := NewBloomFilter(len(w.keys))
	for _, k := range w.keys {
		bf.Add(k)
	}
	bloomData := bf.Encode()
	bloomOff := w.pos
	if _, err := w.f.Write(bloomData); err != nil {
		return err
	}
	w.pos += int64(len(bloomData))

	// index: uvarint n, then for each key (uvarint len, key, uvarint offset)
	var idx []byte
	idx = binary.AppendUvarint(idx, uint64(len(w.keys)))
	for i, k := range w.keys {
		idx = binary.AppendUvarint(idx, uint64(len(k)))
		idx = append(idx, k...)
		idx = binary.AppendUvarint(idx, w.offset[i])
	}
	indexOff := w.pos
	if _, err := w.f.Write(idx); err != nil {
		return err
	}
	w.pos += int64(len(idx))

	foot := make([]byte, 8+8+8+4)
	binary.LittleEndian.PutUint64(foot[0:8], uint64(indexOff))
	binary.LittleEndian.PutUint64(foot[8:16], uint64(bloomOff))
	binary.LittleEndian.PutUint64(foot[16:24], uint64(dataEnd))
	binary.LittleEndian.PutUint32(foot[24:28], uint32(len(w.keys)))
	if _, err := w.f.Write(foot); err != nil {
		return err
	}
	if err := w.f.Sync(); err != nil {
		return err
	}
	err := w.f.Close()
	w.f = nil
	return err
}

// SSTReader reads an SSTable.
type SSTReader struct {
	path   string
	data   []byte
	index  []indexEntry
	bloom  *BloomFilter
	dataEnd int64
}

type indexEntry struct {
	key string
	off uint64
}

// OpenSSTReader mmap-less: reads whole file for simplicity.
func OpenSSTReader(path string) (*SSTReader, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("sst read: %w", err)
	}
	if len(b) < len(magicSST)+28 {
		return nil, errSSTCorrupt
	}
	if !bytes.Equal(b[:len(magicSST)], magicSST) {
		return nil, errSSTCorrupt
	}
	foot := b[len(b)-28:]
	indexOff := int64(binary.LittleEndian.Uint64(foot[0:8]))
	bloomOff := int64(binary.LittleEndian.Uint64(foot[8:16]))
	dataEnd := int64(binary.LittleEndian.Uint64(foot[16:24]))
	_ = binary.LittleEndian.Uint32(foot[24:28])

	if bloomOff >= indexOff {
		return nil, errSSTCorrupt
	}
	bloom, err := DecodeBloomFilter(b[bloomOff:indexOff])
	if err != nil {
		return nil, err
	}
	idxBuf := b[indexOff : int64(len(b))-28]
	entries, err := parseIndex(idxBuf)
	if err != nil {
		return nil, err
	}
	return &SSTReader{
		path:    path,
		data:    b,
		index:   entries,
		bloom:   bloom,
		dataEnd: dataEnd,
	}, nil
}

func parseIndex(buf []byte) ([]indexEntry, error) {
	p := 0
	n, np, err := readUvarint(buf, p)
	if err != nil {
		return nil, err
	}
	p = np
	var out []indexEntry
	for i := uint64(0); i < n; i++ {
		kl, np2, err := readUvarint(buf, p)
		if err != nil {
			return nil, err
		}
		p = np2
		if p+int(kl) > len(buf) {
			return nil, errSSTCorrupt
		}
		key := string(buf[p : p+int(kl)])
		p += int(kl)
		off, np3, err := readUvarint(buf, p)
		if err != nil {
			return nil, err
		}
		p = np3
		out = append(out, indexEntry{key: key, off: off})
	}
	return out, nil
}

// Get looks up key; returns false if absent (uses bloom short-circuit).
func (r *SSTReader) Get(key string) ([]byte, bool) {
	if !r.bloom.MaybeContains(key) {
		return nil, false
	}
	// binary search index
	ix := sort.Search(len(r.index), func(i int) bool {
		return r.index[i].key >= key
	})
	if ix >= len(r.index) || r.index[ix].key != key {
		return nil, false
	}
	off := int(r.index[ix].off)
	var end int
	if ix+1 < len(r.index) {
		end = int(r.index[ix+1].off)
	} else {
		end = int(r.dataEnd)
	}
	rec := r.data[off:end]
	k, v, err := decodeKV(rec, 0)
	if err != nil || k != key {
		return nil, false
	}
	return v, true
}

func decodeKV(data []byte, p int) (key string, value []byte, err error) {
	kl, np, err := readUvarint(data, p)
	if err != nil {
		return "", nil, err
	}
	p = np
	if p+int(kl) > len(data) {
		return "", nil, errSSTCorrupt
	}
	key = string(data[p : p+int(kl)])
	p += int(kl)
	vl, np2, err := readUvarint(data, p)
	if err != nil {
		return "", nil, err
	}
	p = np2
	if p+int(vl) > len(data) {
		return "", nil, errSSTCorrupt
	}
	value = append([]byte(nil), data[p:p+int(vl)]...)
	return key, value, nil
}

// ScanPrefix calls fn for keys in [start, end) in this SST only.
func (r *SSTReader) ScanPrefix(start, end string, fn func(key string, value []byte) bool) {
	for _, ent := range r.index {
		if ent.key < start {
			continue
		}
		if end != "" && ent.key >= end {
			break
		}
		v, ok := r.Get(ent.key)
		if !ok {
			continue
		}
		if !fn(ent.key, v) {
			break
		}
	}
}

