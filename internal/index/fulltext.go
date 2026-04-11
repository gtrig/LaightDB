package index

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// Posting is a BM25 posting.
type Posting struct {
	DocID string
	TF    int
}

// FullText maintains an in-memory BM25 inverted index.
type FullText struct {
	terms map[string]map[string]int // term -> docID -> term frequency
	dl    map[string]int            // docID -> document length (token count)
	N     int                       // document count
	k1    float64
	b     float64
}

// NewFullText creates an empty index with Okapi BM25 defaults.
func NewFullText() *FullText {
	return &FullText{
		terms: make(map[string]map[string]int),
		dl:    make(map[string]int),
		k1:    1.2,
		b:     0.75,
	}
}

// Tokenize splits text into lowercase terms.
func Tokenize(text string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		out = append(out, cur.String())
		cur.Reset()
	}
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func avgDL(dl map[string]int, n int) float64 {
	if n == 0 {
		return 1
	}
	var sum int
	for _, v := range dl {
		sum += v
	}
	return float64(sum) / float64(n)
}

// AddDocument indexes docID with given text (replaces prior index for doc).
func (f *FullText) AddDocument(docID, text string) {
	f.RemoveDocument(docID)
	toks := Tokenize(text)
	if len(toks) == 0 {
		return
	}
	f.dl[docID] = len(toks)
	f.N++
	tf := make(map[string]int)
	for _, t := range toks {
		tf[t]++
	}
	for term, c := range tf {
		if f.terms[term] == nil {
			f.terms[term] = make(map[string]int)
		}
		f.terms[term][docID] = c
	}
}

// RemoveDocument drops docID from the index.
func (f *FullText) RemoveDocument(docID string) {
	if _, ok := f.dl[docID]; !ok {
		return
	}
	for term, m := range f.terms {
		delete(m, docID)
		if len(m) == 0 {
			delete(f.terms, term)
		}
	}
	delete(f.dl, docID)
	f.N--
}

// IDF computes inverse document frequency.
func (f *FullText) IDF(term string) float64 {
	n := len(f.terms[term])
	if n == 0 || f.N == 0 {
		return 0
	}
	return math.Log(1.0 + (float64(f.N)-float64(n)+0.5)/(float64(n)+0.5))
}

// Score returns BM25 score for term in document.
func (f *FullText) Score(term, docID string) float64 {
	tf := 0
	if m := f.terms[term]; m != nil {
		tf = m[docID]
	}
	if tf == 0 {
		return 0
	}
	dlen := float64(f.dl[docID])
	avgdl := avgDL(f.dl, f.N)
	idf := f.IDF(term)
	num := float64(tf) * (f.k1 + 1)
	den := float64(tf) + f.k1*(1-f.b+f.b*(dlen/avgdl))
	return idf * (num / den)
}

// Search returns ranked doc IDs by BM25 sum for the query.
func (f *FullText) Search(query string, topK int) []RankedID {
	qterms := Tokenize(query)
	type acc struct {
		id    string
		score float64
	}
	var scores map[string]float64
	for _, qt := range qterms {
		for docID := range f.terms[qt] {
			if scores == nil {
				scores = make(map[string]float64)
			}
			scores[docID] += f.Score(qt, docID)
		}
	}
	if scores == nil {
		return nil
	}
	var list []acc
	for id, s := range scores {
		list = append(list, acc{id: id, score: s})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].score == list[j].score {
			return list[i].id < list[j].id
		}
		return list[i].score > list[j].score
	})
	if topK > 0 && len(list) > topK {
		list = list[:topK]
	}
	out := make([]RankedID, len(list))
	for i := range list {
		out[i] = RankedID{ID: list[i].id, Score: list[i].score}
	}
	return out
}

// RankedID is a search hit with score.
type RankedID struct {
	ID    string
	Score float64
}

// EncodeSnapshot serializes the index for persistence.
func (f *FullText) EncodeSnapshot() []byte {
	var buf []byte
	buf = appendUvarint(buf, uint64(f.N))
	buf = appendUvarint(buf, uint64(len(f.dl)))
	for id, l := range f.dl {
		buf = appendString(buf, id)
		buf = appendUvarint(buf, uint64(l))
	}
	buf = appendUvarint(buf, uint64(len(f.terms)))
	for term, m := range f.terms {
		buf = appendString(buf, term)
		buf = appendUvarint(buf, uint64(len(m)))
		for doc, tf := range m {
			buf = appendString(buf, doc)
			buf = appendUvarint(buf, uint64(tf))
		}
	}
	return buf
}

// DecodeSnapshot restores from EncodeSnapshot.
func DecodeSnapshot(data []byte) (*FullText, error) {
	f := NewFullText()
	p := 0
	n, np, err := readUvarint(data, p)
	if err != nil {
		return nil, err
	}
	p = np
	f.N = int(n)
	ndl, np, err := readUvarint(data, p)
	if err != nil {
		return nil, err
	}
	p = np
	for i := uint64(0); i < ndl; i++ {
		id, np2, err := readString(data, p)
		if err != nil {
			return nil, err
		}
		p = np2
		l, np3, err := readUvarint(data, p)
		if err != nil {
			return nil, err
		}
		p = np3
		f.dl[id] = int(l)
	}
	nt, np, err := readUvarint(data, p)
	if err != nil {
		return nil, err
	}
	p = np
	for i := uint64(0); i < nt; i++ {
		term, np2, err := readString(data, p)
		if err != nil {
			return nil, err
		}
		p = np2
		nm, np3, err := readUvarint(data, p)
		if err != nil {
			return nil, err
		}
		p = np3
		m := make(map[string]int)
		for j := uint64(0); j < nm; j++ {
			doc, np4, err := readString(data, p)
			if err != nil {
				return nil, err
			}
			p = np4
			tf, np5, err := readUvarint(data, p)
			if err != nil {
				return nil, err
			}
			p = np5
			m[doc] = int(tf)
		}
		f.terms[term] = m
	}
	return f, nil
}

func appendUvarint(buf []byte, x uint64) []byte {
	var scratch [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(scratch[:], x)
	return append(buf, scratch[:n]...)
}

func appendString(buf []byte, s string) []byte {
	buf = appendUvarint(buf, uint64(len(s)))
	return append(buf, s...)
}

func readUvarint(data []byte, p int) (uint64, int, error) {
	if p >= len(data) {
		return 0, p, fmt.Errorf("truncated")
	}
	v, n := binary.Uvarint(data[p:])
	if n <= 0 {
		return 0, p, fmt.Errorf("bad uvarint")
	}
	return v, p + n, nil
}

func readString(data []byte, p int) (string, int, error) {
	l, np, err := readUvarint(data, p)
	if err != nil {
		return "", p, err
	}
	if np+int(l) > len(data) {
		return "", p, fmt.Errorf("truncated string")
	}
	return string(data[np : np+int(l)]), np + int(l), nil
}
