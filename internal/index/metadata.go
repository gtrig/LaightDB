package index

import (
	"fmt"
	"sort"
	"strings"
)

// MetadataIndex maps metadata key:value -> document IDs.
type MetadataIndex struct {
	idx map[string]map[string]map[string]struct{} // key -> value -> docIDs
}

// NewMetadataIndex creates an empty index.
func NewMetadataIndex() *MetadataIndex {
	return &MetadataIndex{idx: make(map[string]map[string]map[string]struct{})}
}

// Set replaces metadata for a document (clears old key/value pairs for that doc first).
func (m *MetadataIndex) Set(docID string, meta map[string]string) {
	m.Remove(docID)
	for k, v := range meta {
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		if k == "" {
			continue
		}
		if m.idx[k] == nil {
			m.idx[k] = make(map[string]map[string]struct{})
		}
		if m.idx[k][v] == nil {
			m.idx[k][v] = make(map[string]struct{})
		}
		m.idx[k][v][docID] = struct{}{}
	}
}

// Remove drops doc from all postings.
func (m *MetadataIndex) Remove(docID string) {
	for _, vm := range m.idx {
		for v, set := range vm {
			delete(set, docID)
			if len(set) == 0 {
				delete(vm, v)
			}
		}
	}
}

// EncodeSnapshot serializes the metadata index for persistence.
func (m *MetadataIndex) EncodeSnapshot() []byte {
	var buf []byte
	// sort keys for deterministic output
	keys := make([]string, 0, len(m.idx))
	for k := range m.idx {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	buf = appendUvarint(buf, uint64(len(keys)))
	for _, k := range keys {
		buf = appendString(buf, k)
		vm := m.idx[k]
		vals := make([]string, 0, len(vm))
		for v := range vm {
			vals = append(vals, v)
		}
		sort.Strings(vals)
		buf = appendUvarint(buf, uint64(len(vals)))
		for _, v := range vals {
			buf = appendString(buf, v)
			set := vm[v]
			docIDs := make([]string, 0, len(set))
			for id := range set {
				docIDs = append(docIDs, id)
			}
			sort.Strings(docIDs)
			buf = appendUvarint(buf, uint64(len(docIDs)))
			for _, id := range docIDs {
				buf = appendString(buf, id)
			}
		}
	}
	return buf
}

// DecodeMetadataSnapshot restores from EncodeSnapshot.
func DecodeMetadataSnapshot(data []byte) (*MetadataIndex, error) {
	m := NewMetadataIndex()
	p := 0
	nk, np, err := readUvarint(data, p)
	if err != nil {
		return nil, fmt.Errorf("metadata snapshot keys: %w", err)
	}
	p = np
	for i := uint64(0); i < nk; i++ {
		k, np2, err := readString(data, p)
		if err != nil {
			return nil, fmt.Errorf("metadata snapshot key: %w", err)
		}
		p = np2
		nv, np3, err := readUvarint(data, p)
		if err != nil {
			return nil, fmt.Errorf("metadata snapshot values count: %w", err)
		}
		p = np3
		vm := make(map[string]map[string]struct{}, nv)
		for j := uint64(0); j < nv; j++ {
			v, np4, err := readString(data, p)
			if err != nil {
				return nil, fmt.Errorf("metadata snapshot value: %w", err)
			}
			p = np4
			nd, np5, err := readUvarint(data, p)
			if err != nil {
				return nil, fmt.Errorf("metadata snapshot docIDs count: %w", err)
			}
			p = np5
			set := make(map[string]struct{}, nd)
			for l := uint64(0); l < nd; l++ {
				id, np6, err := readString(data, p)
				if err != nil {
					return nil, fmt.Errorf("metadata snapshot docID: %w", err)
				}
				p = np6
				set[id] = struct{}{}
			}
			vm[v] = set
		}
		m.idx[k] = vm
	}
	return m, nil
}

// Match returns doc IDs that satisfy all filters (AND).
func (m *MetadataIndex) Match(filters map[string]string) map[string]struct{} {
	if len(filters) == 0 {
		return nil
	}
	var out map[string]struct{}
	first := true
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		want := filters[k]
		k = strings.ToLower(strings.TrimSpace(k))
		want = strings.TrimSpace(want)
		set := m.idx[k][want]
		if len(set) == 0 {
			return map[string]struct{}{}
		}
		if first {
			out = make(map[string]struct{}, len(set))
			for id := range set {
				out[id] = struct{}{}
			}
			first = false
			continue
		}
		for id := range out {
			if _, ok := set[id]; !ok {
				delete(out, id)
			}
		}
	}
	return out
}
