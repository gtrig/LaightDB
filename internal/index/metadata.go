package index

import (
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
