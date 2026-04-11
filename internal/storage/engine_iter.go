package storage

import (
	"sort"
	"strings"
)

// PrefixKeys returns sorted unique keys starting with prefix (may include tombstones in list; use Get to resolve).
func (e *Engine) PrefixKeys(prefix string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	seen := make(map[string]struct{})
	var keys []string
	e.mem.RawScan(func(key string, _ []byte) bool {
		if strings.HasPrefix(key, prefix) {
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				keys = append(keys, key)
			}
		}
		return true
	})
	for i := len(e.sstPaths) - 1; i >= 0; i-- {
		r, err := OpenSSTReader(e.sstPaths[i])
		if err != nil {
			continue
		}
		for _, ent := range r.index {
			if strings.HasPrefix(ent.key, prefix) {
				if _, ok := seen[ent.key]; !ok {
					seen[ent.key] = struct{}{}
					keys = append(keys, ent.key)
				}
			}
		}
	}
	sort.Strings(keys)
	return keys
}
