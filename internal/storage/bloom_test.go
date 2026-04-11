package storage

import (
	"strconv"
	"testing"
)

func TestBloomBasic(t *testing.T) {
	t.Parallel()
	b := NewBloomFilter(100)
	b.Add("hello")
	if !b.MaybeContains("hello") {
		t.Fatal("should contain")
	}
	// unlikely but possible false positive on random string
	if b.MaybeContains("totally-absent-key-xyz-12345") {
		t.Log("false positive (acceptable)")
	}
}

func TestBloomEncodeRoundTrip(t *testing.T) {
	t.Parallel()
	b := NewBloomFilter(50)
	for i := 0; i < 40; i++ {
		b.Add(strconv.Itoa(i))
	}
	data := b.Encode()
	b2, err := DecodeBloomFilter(data)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if !b2.MaybeContains(strconv.Itoa(i)) {
			t.Fatalf("missing %d", i)
		}
	}
}
