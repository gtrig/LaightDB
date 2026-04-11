package storage

import (
	"bytes"
	"math/rand/v2"
	"sync"
)

const (
	maxLevel = 32
	p        = 0.25
)

type skipNode struct {
	key   string
	value []byte
	next  []*skipNode
}

// SkipList is a concurrent sorted map from string key to byte value.
type SkipList struct {
	mu       sync.RWMutex
	head     *skipNode
	rng      *rand.Rand
	level    int
	length   int
	byteSize int
}

// NewSkipList returns an empty skip list.
func NewSkipList() *SkipList {
	h := &skipNode{next: make([]*skipNode, maxLevel)}
	return &SkipList{
		head: h,
		rng:  rand.New(rand.NewPCG(1, 2)),
	}
}

func (s *SkipList) randomLevel() int {
	lvl := 1
	for lvl < maxLevel && s.rng.Float64() < p {
		lvl++
	}
	return lvl
}

// Len returns the number of entries.
func (s *SkipList) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.length
}

// ByteSize estimates memory used by stored values and keys.
func (s *SkipList) ByteSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byteSize
}

// Put inserts or replaces key.
func (s *SkipList) Put(key string, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	update := make([]*skipNode, maxLevel)
	cur := s.head
	for i := s.level - 1; i >= 0; i-- {
		for cur.next[i] != nil && bytes.Compare([]byte(cur.next[i].key), []byte(key)) < 0 {
			cur = cur.next[i]
		}
		update[i] = cur
	}
	cur = cur.next[0]
	if cur != nil && cur.key == key {
		oldLen := len(cur.key) + len(cur.value)
		newLen := len(key) + len(value)
		s.byteSize += newLen - oldLen
		cur.value = append([]byte(nil), value...)
		return
	}
	lvl := s.randomLevel()
	if lvl > s.level {
		for i := s.level; i < lvl; i++ {
			update[i] = s.head
		}
		s.level = lvl
	}
	n := &skipNode{
		key:   key,
		value: append([]byte(nil), value...),
		next:  make([]*skipNode, lvl),
	}
	for i := 0; i < lvl; i++ {
		n.next[i] = update[i].next[i]
		update[i].next[i] = n
	}
	s.length++
	s.byteSize += len(key) + len(value)
}

// Get returns a copy of the value or nil if missing.
func (s *SkipList) Get(key string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cur := s.head
	for i := s.level - 1; i >= 0; i-- {
		for cur.next[i] != nil && bytes.Compare([]byte(cur.next[i].key), []byte(key)) < 0 {
			cur = cur.next[i]
		}
	}
	cur = cur.next[0]
	if cur != nil && cur.key == key {
		return append([]byte(nil), cur.value...), true
	}
	return nil, false
}

// Delete removes key; reports whether it existed.
func (s *SkipList) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	update := make([]*skipNode, maxLevel)
	cur := s.head
	for i := s.level - 1; i >= 0; i-- {
		for cur.next[i] != nil && bytes.Compare([]byte(cur.next[i].key), []byte(key)) < 0 {
			cur = cur.next[i]
		}
		update[i] = cur
	}
	cur = cur.next[0]
	if cur == nil || cur.key != key {
		return false
	}
	for i := 0; i < s.level; i++ {
		if update[i].next[i] != cur {
			break
		}
		update[i].next[i] = cur.next[i]
	}
	s.byteSize -= len(cur.key) + len(cur.value)
	s.length--
	for s.level > 0 && s.head.next[s.level-1] == nil {
		s.level--
	}
	return true
}

// Scan calls fn for each key in [start, end) lexicographic order.
func (s *SkipList) Scan(start, end string, fn func(key string, value []byte) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cur := s.head.next[0]
	for cur != nil {
		if bytes.Compare([]byte(cur.key), []byte(start)) < 0 {
			cur = cur.next[0]
			continue
		}
		if end != "" && bytes.Compare([]byte(cur.key), []byte(end)) >= 0 {
			break
		}
		if !fn(cur.key, cur.value) {
			break
		}
		cur = cur.next[0]
	}
}
