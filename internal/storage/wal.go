package storage

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

const (
	walTypePut    = byte(1)
	walTypeDelete = byte(2)
)

// WAL is an append-only write-ahead log.
type WAL struct {
	path string
	f    *os.File
}

// OpenWAL opens or creates a WAL file.
func OpenWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("wal open: %w", err)
	}
	return &WAL{path: path, f: f}, nil
}

// Path returns the file path.
func (w *WAL) Path() string { return w.path }

// Close releases the file.
func (w *WAL) Close() error {
	if w.f == nil {
		return nil
	}
	err := w.f.Close()
	w.f = nil
	return err
}

func (w *WAL) appendRecord(typ byte, key string, value []byte) error {
	var body []byte
	body = append(body, typ)
	body = appendUvarint(body, uint64(len(key)))
	body = append(body, key...)
	body = appendUvarint(body, uint64(len(value)))
	body = append(body, value...)
	crc := crc32.ChecksumIEEE(body)
	var hdr [8]byte
	binary.LittleEndian.PutUint32(hdr[0:4], uint32(len(body)))
	binary.LittleEndian.PutUint32(hdr[4:8], crc)
	if _, err := w.f.Write(hdr[:]); err != nil {
		return fmt.Errorf("wal write hdr: %w", err)
	}
	if _, err := w.f.Write(body); err != nil {
		return fmt.Errorf("wal write body: %w", err)
	}
	return nil
}

// AppendPut logs a put operation.
func (w *WAL) AppendPut(key string, value []byte) error {
	return w.appendRecord(walTypePut, key, value)
}

// AppendDelete logs a delete.
func (w *WAL) AppendDelete(key string) error {
	return w.appendRecord(walTypeDelete, key, nil)
}

// Sync flushes to disk.
func (w *WAL) Sync() error {
	if w.f == nil {
		return nil
	}
	return w.f.Sync()
}

// Replay reads from the beginning and invokes fn for each record.
func (w *WAL) Replay(fn func(typ byte, key string, value []byte) error) error {
	if w.f == nil {
		return nil
	}
	if _, err := w.f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("wal seek: %w", err)
	}
	for {
		var hdr [8]byte
		if _, err := io.ReadFull(w.f, hdr[:]); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("wal read hdr: %w", err)
		}
		bodyLen := int(binary.LittleEndian.Uint32(hdr[0:4]))
		crcWant := binary.LittleEndian.Uint32(hdr[4:8])
		if bodyLen < 1 {
			return errWALCorrupt
		}
		body := make([]byte, bodyLen)
		if _, err := io.ReadFull(w.f, body); err != nil {
			return fmt.Errorf("wal read body: %w", err)
		}
		if crc32.ChecksumIEEE(body) != crcWant {
			return errWALCorrupt
		}
		if len(body) < 1 {
			return errWALCorrupt
		}
		typ := body[0]
		p := 1
		kl, np, err := readUvarint(body, p)
		if err != nil {
			return err
		}
		p = np
		if p+int(kl) > len(body) {
			return errWALCorrupt
		}
		key := string(body[p : p+int(kl)])
		p += int(kl)
		vl, np2, err := readUvarint(body, p)
		if err != nil {
			return err
		}
		p = np2
		if p+int(vl) > len(body) {
			return errWALCorrupt
		}
		val := body[p : p+int(vl)]
		if err := fn(typ, key, val); err != nil {
			return err
		}
	}
}

// Truncate resets the WAL file to empty.
func (w *WAL) Truncate() error {
	if w.f == nil {
		return nil
	}
	if err := w.f.Truncate(0); err != nil {
		return fmt.Errorf("wal truncate: %w", err)
	}
	_, err := w.f.Seek(0, io.SeekStart)
	return err
}
