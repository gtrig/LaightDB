package storage

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"
)

const codecVersion = byte(1)

const (
	tagID = iota + 1
	tagCollection
	tagContent
	tagContentType
	tagSummary
	tagChunk
	tagMetadataKV
	tagEmbedding
	tagCreatedAt
	tagUpdatedAt
	tagTokenCount
	tagCompactContent
	tagCompactTokenCount
)

// Encode serializes ContextEntry to binary.
func Encode(e ContextEntry) []byte {
	var buf []byte
	buf = append(buf, codecVersion)
	buf = appendStringField(buf, tagID, e.ID)
	buf = appendStringField(buf, tagCollection, e.Collection)
	buf = appendStringField(buf, tagContent, e.Content)
	buf = appendStringField(buf, tagContentType, e.ContentType)
	buf = appendStringField(buf, tagSummary, e.Summary)
	keys := make([]string, 0, len(e.Metadata))
	for k := range e.Metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		buf = appendMetaKV(buf, k, e.Metadata[k])
	}
	for _, c := range e.Chunks {
		buf = appendChunk(buf, c)
	}
	buf = appendFloatSlice(buf, tagEmbedding, e.Embedding)
	buf = appendTime(buf, tagCreatedAt, e.CreatedAt)
	buf = appendTime(buf, tagUpdatedAt, e.UpdatedAt)
	buf = appendVarintTagged(buf, tagTokenCount, uint64(e.TokenCount))
	if e.CompactContent != "" {
		buf = appendStringField(buf, tagCompactContent, e.CompactContent)
		buf = appendVarintTagged(buf, tagCompactTokenCount, uint64(e.CompactTokenCount))
	}
	return buf
}

func appendVarintTagged(buf []byte, tag byte, v uint64) []byte {
	buf = append(buf, tag)
	return appendUvarint(buf, v)
}

func appendStringField(buf []byte, tag byte, s string) []byte {
	buf = append(buf, tag)
	buf = appendUvarint(buf, uint64(len(s)))
	buf = append(buf, s...)
	return buf
}

func appendMetaKV(buf []byte, k, v string) []byte {
	buf = append(buf, tagMetadataKV)
	buf = appendUvarint(buf, uint64(len(k)))
	buf = append(buf, k...)
	buf = appendUvarint(buf, uint64(len(v)))
	buf = append(buf, v...)
	return buf
}

func appendChunk(buf []byte, c Chunk) []byte {
	buf = append(buf, tagChunk)
	buf = appendUvarint(buf, uint64(c.Index))
	buf = appendStringSub(buf, c.ParentID)
	buf = appendStringSub(buf, c.Text)
	buf = appendFloatSliceEmbedded(buf, c.Embedding)
	return buf
}

func appendStringSub(buf []byte, s string) []byte {
	buf = appendUvarint(buf, uint64(len(s)))
	buf = append(buf, s...)
	return buf
}

func appendFloatSlice(buf []byte, tag byte, fs []float32) []byte {
	buf = append(buf, tag)
	buf = appendUvarint(buf, uint64(len(fs)))
	for _, f := range fs {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], math.Float32bits(f))
		buf = append(buf, b[:]...)
	}
	return buf
}

func appendFloatSliceEmbedded(buf []byte, fs []float32) []byte {
	buf = appendUvarint(buf, uint64(len(fs)))
	for _, f := range fs {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], math.Float32bits(f))
		buf = append(buf, b[:]...)
	}
	return buf
}

func appendTime(buf []byte, tag byte, tm time.Time) []byte {
	buf = append(buf, tag)
	n := tm.UnixNano()
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(n))
	return append(buf, b[:]...)
}

func appendUvarint(buf []byte, x uint64) []byte {
	var scratch [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(scratch[:], x)
	return append(buf, scratch[:n]...)
}

// Decode deserializes binary from Encode.
func Decode(data []byte) (ContextEntry, error) {
	var e ContextEntry
	if len(data) < 1 {
		return e, errCodecTrunc
	}
	if data[0] != codecVersion {
		return e, fmt.Errorf("%w: %d", errCodecVersion, data[0])
	}
	p := 1
	for p < len(data) {
		tag := data[p]
		p++
		switch tag {
		case tagID:
			s, np, err := readString(data, p)
			if err != nil {
				return e, err
			}
			e.ID, p = s, np
		case tagCollection:
			s, np, err := readString(data, p)
			if err != nil {
				return e, err
			}
			e.Collection, p = s, np
		case tagContent:
			s, np, err := readString(data, p)
			if err != nil {
				return e, err
			}
			e.Content, p = s, np
		case tagContentType:
			s, np, err := readString(data, p)
			if err != nil {
				return e, err
			}
			e.ContentType, p = s, np
		case tagSummary:
			s, np, err := readString(data, p)
			if err != nil {
				return e, err
			}
			e.Summary, p = s, np
		case tagMetadataKV:
			k, np, err := readString(data, p)
			if err != nil {
				return e, err
			}
			v, np2, err := readString(data, np)
			if err != nil {
				return e, err
			}
			if e.Metadata == nil {
				e.Metadata = make(map[string]string)
			}
			e.Metadata[k] = v
			p = np2
		case tagChunk:
			var c Chunk
			idx, np, err := readUvarint(data, p)
			if err != nil {
				return e, err
			}
			c.Index = int(idx)
			pid, np, err := readString(data, np)
			if err != nil {
				return e, err
			}
			c.ParentID = pid
			txt, np, err := readString(data, np)
			if err != nil {
				return e, err
			}
			c.Text = txt
			em, np, err := readFloatSliceEmbedded(data, np)
			if err != nil {
				return e, err
			}
			c.Embedding = em
			e.Chunks = append(e.Chunks, c)
			p = np
		case tagEmbedding:
			fs, np, err := readFloatSlice(data, p)
			if err != nil {
				return e, err
			}
			e.Embedding = fs
			p = np
		case tagCreatedAt:
			if len(data[p:]) < 8 {
				return e, errCodecTrunc
			}
			n := int64(binary.LittleEndian.Uint64(data[p : p+8]))
			e.CreatedAt = time.Unix(0, n)
			p += 8
		case tagUpdatedAt:
			if len(data[p:]) < 8 {
				return e, errCodecTrunc
			}
			n := int64(binary.LittleEndian.Uint64(data[p : p+8]))
			e.UpdatedAt = time.Unix(0, n)
			p += 8
		case tagTokenCount:
			v, np, err := readUvarint(data, p)
			if err != nil {
				return e, err
			}
			e.TokenCount = int(v)
			p = np
		case tagCompactContent:
			s, np, err := readString(data, p)
			if err != nil {
				return e, err
			}
			e.CompactContent, p = s, np
		case tagCompactTokenCount:
			v, np, err := readUvarint(data, p)
			if err != nil {
				return e, err
			}
			e.CompactTokenCount = int(v)
			p = np
		default:
			return e, fmt.Errorf("storage: unknown tag %d", tag)
		}
	}
	return e, nil
}

func readString(data []byte, p int) (string, int, error) {
	v, np, err := readUvarint(data, p)
	if err != nil {
		return "", p, err
	}
	n := int(v)
	if np+n > len(data) {
		return "", p, errCodecTrunc
	}
	return string(data[np : np+n]), np + n, nil
}

func readUvarint(data []byte, p int) (uint64, int, error) {
	if p >= len(data) {
		return 0, p, errCodecTrunc
	}
	v, n := binary.Uvarint(data[p:])
	if n <= 0 {
		return 0, p, errCodecTrunc
	}
	return v, p + n, nil
}

func readFloatSlice(data []byte, p int) ([]float32, int, error) {
	cnt, np, err := readUvarint(data, p)
	if err != nil {
		return nil, p, err
	}
	n := int(cnt)
	need := n * 4
	if np+need > len(data) {
		return nil, p, errCodecTrunc
	}
	out := make([]float32, n)
	for i := range out {
		b := binary.LittleEndian.Uint32(data[np+i*4:])
		out[i] = math.Float32frombits(b)
	}
	return out, np + need, nil
}

func readFloatSliceEmbedded(data []byte, p int) ([]float32, int, error) {
	return readFloatSlice(data, p)
}
