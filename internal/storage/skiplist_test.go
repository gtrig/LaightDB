package storage

import (
	"strconv"
	"sync"
	"testing"
)

func TestSkipListPutGetDelete(t *testing.T) {
	t.Parallel()
	s := NewSkipList()
	s.Put("a", []byte("1"))
	s.Put("b", []byte("2"))
	v, ok := s.Get("a")
	if !ok || string(v) != "1" {
		t.Fatalf("Get a: ok=%v v=%q", ok, v)
	}
	s.Put("a", []byte("3"))
	v, ok = s.Get("a")
	if !ok || string(v) != "3" {
		t.Fatalf("replace: %q", v)
	}
	if !s.Delete("a") {
		t.Fatal("delete a")
	}
	if _, ok := s.Get("a"); ok {
		t.Fatal("expected miss")
	}
	if s.Len() != 1 {
		t.Fatalf("len want 1 got %d", s.Len())
	}
}

func TestSkipListOrder(t *testing.T) {
	t.Parallel()
	s := NewSkipList()
	for i := 10; i >= 0; i-- {
		s.Put(strconv.Itoa(i), []byte("x"))
	}
	var got []string
	s.Scan("", "", func(key string, _ []byte) bool {
		got = append(got, key)
		return true
	})
	// Lexicographic order: "0","1","10",...
	want := []string{"0", "1", "10", "2", "3", "4", "5", "6", "7", "8", "9"}
	if len(got) != len(want) {
		t.Fatalf("len got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order: got %v want %v", got, want)
		}
	}
}

func TestSkipListScanRange(t *testing.T) {
	t.Parallel()
	s := NewSkipList()
	for _, k := range []string{"a", "b", "c", "d"} {
		s.Put(k, []byte(k))
	}
	var keys []string
	s.Scan("b", "d", func(key string, _ []byte) bool {
		keys = append(keys, key)
		return true
	})
	if len(keys) != 2 || keys[0] != "b" || keys[1] != "c" {
		t.Fatalf("range: %v", keys)
	}
}

func TestSkipListConcurrent(t *testing.T) {
	t.Parallel()
	s := NewSkipList()
	const n = 100
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Go(func() {
			s.Put(strconv.Itoa(i), []byte("v"))
		})
	}
	wg.Wait()
	if s.Len() != n {
		t.Fatalf("len %d want %d", s.Len(), n)
	}
}
