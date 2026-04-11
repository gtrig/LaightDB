package storage

import (
	"path/filepath"
	"testing"
)

func TestWALAppendReplay(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "wal.log")
	w, err := OpenWAL(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	if err := w.AppendPut("k1", []byte("v1")); err != nil {
		t.Fatal(err)
	}
	if err := w.AppendDelete("k2"); err != nil {
		t.Fatal(err)
	}
	if err := w.Sync(); err != nil {
		t.Fatal(err)
	}
	var ops []string
	err = w.Replay(func(typ byte, key string, value []byte) error {
		switch typ {
		case walTypePut:
			ops = append(ops, "put:"+key+":"+string(value))
		case walTypeDelete:
			ops = append(ops, "del:"+key)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 || ops[0] != "put:k1:v1" || ops[1] != "del:k2" {
		t.Fatalf("%v", ops)
	}
}
