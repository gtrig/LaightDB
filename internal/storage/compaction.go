package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// CompactMerge merges two SST files (older, newer) into outPath; same key keeps newer value.
func CompactMerge(olderPath, newerPath, outPath string) error {
	a, err := OpenSSTReader(olderPath)
	if err != nil {
		return fmt.Errorf("compaction open older: %w", err)
	}
	b, err := OpenSSTReader(newerPath)
	if err != nil {
		return fmt.Errorf("compaction open newer: %w", err)
	}
	w, err := NewSSTWriter(outPath)
	if err != nil {
		return err
	}
	ia, ib := 0, 0
	for ia < len(a.index) && ib < len(b.index) {
		ka, kb := a.index[ia].key, b.index[ib].key
		switch {
		case ka < kb:
			va, ok := a.Get(ka)
			if ok {
				if err := w.Append(ka, va); err != nil {
					_ = w.Close()
					return err
				}
			}
			ia++
		case kb < ka:
			vb, ok := b.Get(kb)
			if ok {
				if err := w.Append(kb, vb); err != nil {
					_ = w.Close()
					return err
				}
			}
			ib++
		default:
			vb, ok := b.Get(kb)
			if ok {
				if err := w.Append(kb, vb); err != nil {
					_ = w.Close()
					return err
				}
			}
			ia++
			ib++
		}
	}
	for ia < len(a.index) {
		ka := a.index[ia].key
		va, ok := a.Get(ka)
		if ok {
			if err := w.Append(ka, va); err != nil {
				_ = w.Close()
				return err
			}
		}
		ia++
	}
	for ib < len(b.index) {
		kb := b.index[ib].key
		vb, ok := b.Get(kb)
		if ok {
			if err := w.Append(kb, vb); err != nil {
				_ = w.Close()
				return err
			}
		}
		ib++
	}
	return w.Close()
}

// RunCompaction merges the oldest pair of SSTables when at least 2 exist.
func (e *Engine) RunCompaction() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.sstPaths) < 2 {
		return nil
	}
	old := e.sstPaths[0]
	newer := e.sstPaths[1]
	out := filepath.Join(e.dir, "sst", fmt.Sprintf("%06d.sst", e.nextSSTSeq))
	e.nextSSTSeq++
	_ = os.Remove(out)
	if err := CompactMerge(old, newer, out); err != nil {
		e.nextSSTSeq--
		return err
	}
	_ = os.Remove(old)
	_ = os.Remove(newer)
	rest := append([]string{}, e.sstPaths[2:]...)
	rest = append(rest, out)
	sort.Strings(rest)
	e.sstPaths = rest
	return nil
}
