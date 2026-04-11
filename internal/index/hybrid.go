package index

import (
	"sort"
)

const rrfK = 60.0

// RRF merges ranked lists using reciprocal rank fusion.
func RRF(lists [][]RankedID, topK int) []RankedID {
	score := make(map[string]float64)
	order := make(map[string]int)
	for _, list := range lists {
		for rank, item := range list {
			score[item.ID] += 1.0 / (rrfK + float64(rank+1))
			if _, ok := order[item.ID]; !ok {
				order[item.ID] = rank
			}
		}
	}
	type pair struct {
		id    string
		score float64
	}
	var ps []pair
	for id, s := range score {
		ps = append(ps, pair{id: id, score: s})
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].score == ps[j].score {
			return ps[i].id < ps[j].id
		}
		return ps[i].score > ps[j].score
	})
	if topK > 0 && len(ps) > topK {
		ps = ps[:topK]
	}
	out := make([]RankedID, len(ps))
	for i := range ps {
		out[i] = RankedID{ID: ps[i].id, Score: ps[i].score}
	}
	return out
}

// FilterByMetadata keeps only IDs present in allow (nil allow = no filter).
func FilterByMetadata(hits []RankedID, allow map[string]struct{}) []RankedID {
	if allow == nil {
		return hits
	}
	var out []RankedID
	for _, h := range hits {
		if _, ok := allow[h.ID]; ok {
			out = append(out, h)
		}
	}
	return out
}
