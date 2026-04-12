package storage

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"
)

const edgeCodecVersion = byte(1)

const (
	etagID = iota + 1
	etagFromID
	etagToID
	etagLabel
	etagWeight
	etagSource
	etagMetadataKV
	etagCreatedAt
)

// EncodeEdge serializes an Edge to binary.
func EncodeEdge(e Edge) []byte {
	var buf []byte
	buf = append(buf, edgeCodecVersion)
	buf = edgeAppendString(buf, etagID, e.ID)
	buf = edgeAppendString(buf, etagFromID, e.FromID)
	buf = edgeAppendString(buf, etagToID, e.ToID)
	buf = edgeAppendString(buf, etagLabel, e.Label)
	buf = edgeAppendFloat64(buf, etagWeight, e.Weight)
	buf = edgeAppendString(buf, etagSource, e.Source)
	keys := make([]string, 0, len(e.Metadata))
	for k := range e.Metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		buf = edgeAppendMetaKV(buf, k, e.Metadata[k])
	}
	buf = edgeAppendTime(buf, etagCreatedAt, e.CreatedAt)
	return buf
}

// DecodeEdge deserializes binary produced by EncodeEdge.
func DecodeEdge(data []byte) (Edge, error) {
	var e Edge
	if len(data) < 1 {
		return e, fmt.Errorf("edge codec: truncated")
	}
	if data[0] != edgeCodecVersion {
		return e, fmt.Errorf("edge codec: unknown version %d", data[0])
	}
	p := 1
	for p < len(data) {
		tag := data[p]
		p++
		switch tag {
		case etagID:
			s, np, err := edgeReadString(data, p)
			if err != nil {
				return e, err
			}
			e.ID, p = s, np
		case etagFromID:
			s, np, err := edgeReadString(data, p)
			if err != nil {
				return e, err
			}
			e.FromID, p = s, np
		case etagToID:
			s, np, err := edgeReadString(data, p)
			if err != nil {
				return e, err
			}
			e.ToID, p = s, np
		case etagLabel:
			s, np, err := edgeReadString(data, p)
			if err != nil {
				return e, err
			}
			e.Label, p = s, np
		case etagWeight:
			f, np, err := edgeReadFloat64(data, p)
			if err != nil {
				return e, err
			}
			e.Weight, p = f, np
		case etagSource:
			s, np, err := edgeReadString(data, p)
			if err != nil {
				return e, err
			}
			e.Source, p = s, np
		case etagMetadataKV:
			k, np, err := edgeReadString(data, p)
			if err != nil {
				return e, err
			}
			v, np2, err := edgeReadString(data, np)
			if err != nil {
				return e, err
			}
			if e.Metadata == nil {
				e.Metadata = make(map[string]string)
			}
			e.Metadata[k] = v
			p = np2
		case etagCreatedAt:
			if len(data[p:]) < 8 {
				return e, fmt.Errorf("edge codec: truncated timestamp")
			}
			n := int64(binary.LittleEndian.Uint64(data[p : p+8]))
			e.CreatedAt = time.Unix(0, n)
			p += 8
		default:
			return e, fmt.Errorf("edge codec: unknown tag %d", tag)
		}
	}
	return e, nil
}

func edgeAppendString(buf []byte, tag byte, s string) []byte {
	buf = append(buf, tag)
	buf = edgeAppendUvarint(buf, uint64(len(s)))
	return append(buf, s...)
}

func edgeAppendFloat64(buf []byte, tag byte, f float64) []byte {
	buf = append(buf, tag)
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], math.Float64bits(f))
	return append(buf, b[:]...)
}

func edgeAppendMetaKV(buf []byte, k, v string) []byte {
	buf = append(buf, etagMetadataKV)
	buf = edgeAppendUvarint(buf, uint64(len(k)))
	buf = append(buf, k...)
	buf = edgeAppendUvarint(buf, uint64(len(v)))
	return append(buf, v...)
}

func edgeAppendTime(buf []byte, tag byte, tm time.Time) []byte {
	buf = append(buf, tag)
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(tm.UnixNano()))
	return append(buf, b[:]...)
}

func edgeAppendUvarint(buf []byte, x uint64) []byte {
	var scratch [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(scratch[:], x)
	return append(buf, scratch[:n]...)
}

func edgeReadString(data []byte, p int) (string, int, error) {
	v, np, err := edgeReadUvarint(data, p)
	if err != nil {
		return "", p, err
	}
	n := int(v)
	if np+n > len(data) {
		return "", p, fmt.Errorf("edge codec: truncated string")
	}
	return string(data[np : np+n]), np + n, nil
}

func edgeReadFloat64(data []byte, p int) (float64, int, error) {
	if p+8 > len(data) {
		return 0, p, fmt.Errorf("edge codec: truncated float64")
	}
	bits := binary.LittleEndian.Uint64(data[p : p+8])
	return math.Float64frombits(bits), p + 8, nil
}

func edgeReadUvarint(data []byte, p int) (uint64, int, error) {
	if p >= len(data) {
		return 0, p, fmt.Errorf("edge codec: truncated uvarint")
	}
	v, n := binary.Uvarint(data[p:])
	if n <= 0 {
		return 0, p, fmt.Errorf("edge codec: bad uvarint")
	}
	return v, p + n, nil
}
