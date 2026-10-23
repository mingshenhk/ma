// Package runfast implements the ID115 two-player proposal's legal patterns.
package runfast

import (
	"crypto/rand"
	"errors"
	"math/big"
	"sort"
)

type Pattern struct {
	Kind  string `json:"kind"`
	High  int    `json:"high"`
	Count int    `json:"count"`
}

func Value(c int) int {
	if c&15 == 1 {
		return 14
	}
	return c & 15
}
func Deck() []int {
	d := []int{}
	for s := 0; s < 4; s++ {
		for _, v := range []int{1, 5, 6, 7, 8, 9, 10, 11, 12, 13} {
			d = append(d, s*16+v)
		}
	}
	return d
}
func Valid(c int) bool { return c >= 0 && c < 64 && (c&15 == 1 || c&15 >= 5 && c&15 <= 13) }
func Sort(a []int) {
	sort.Slice(a, func(i, j int) bool {
		if Value(a[i]) == Value(a[j]) {
			return a[i] < a[j]
		}
		return Value(a[i]) < Value(a[j])
	})
}
func Shuffle(d []int) error {
	for i := len(d) - 1; i > 0; i-- {
		j, e := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if e != nil {
			return e
		}
		d[i], d[j.Int64()] = d[j.Int64()], d[i]
	}
	return nil
}
func Classify(a []int) (Pattern, error) {
	bad := errors.New("不支持的牌型")
	p := Pattern{Count: len(a)}
	if len(a) == 0 || len(a) > 10 {
		return p, bad
	}
	counts := map[int]int{}
	seen := map[int]bool{}
	for _, c := range a {
		if !Valid(c) || seen[c] {
			return p, bad
		}
		seen[c] = true
		counts[Value(c)]++
	}
	vals := []int{}
	for v := range counts {
		vals = append(vals, v)
	}
	sort.Ints(vals)
	p.High = vals[len(vals)-1]
	if len(vals) == 1 {
		switch len(a) {
		case 1:
			p.Kind = "single"
		case 2:
			p.Kind = "pair"
		case 3:
			p.Kind = "triple"
		case 4:
			p.Kind = "bomb"
		default:
			return p, bad
		}
		return p, nil
	}
	n := counts[vals[0]]
	for i, v := range vals {
		if counts[v] != n || i > 0 && v != vals[i-1]+1 {
			return p, bad
		}
	}
	switch {
	case n == 1 && len(vals) >= 3:
		p.Kind = "straight"
	case n == 2 && len(vals) >= 2:
		p.Kind = "pairs"
	default:
		return p, bad
	}
	return p, nil
}
func Beats(a, b Pattern) bool {
	if a.Kind == "bomb" && b.Kind != "bomb" {
		return true
	}
	return a.Kind == b.Kind && a.Count == b.Count && a.High > b.High
}
func Remove(hand, cards []int) ([]int, error) {
	h := append([]int{}, hand...)
	for _, c := range cards {
		found := -1
		for i, x := range h {
			if c == x {
				found = i
				break
			}
		}
		if found < 0 {
			return nil, errors.New("所选牌不在手中")
		}
		h = append(h[:found], h[found+1:]...)
	}
	return h, nil
}
func Validate(hand, cards, table []int, first, nextOne bool) ([]int, Pattern, error) {
	p, e := Classify(cards)
	if e != nil {
		return nil, p, e
	}
	h, e := Remove(hand, cards)
	if e != nil {
		return nil, p, e
	}
	if first {
		ok := false
		for _, c := range cards {
			ok = ok || c == 0x35
		}
		if !ok {
			return nil, p, errors.New("首出必须包含黑桃5")
		}
	}
	if nextOne && p.Kind == "single" && len(table) <= 1 {
		for _, c := range hand {
			if Value(c) > p.High {
				return nil, p, errors.New("下家报单，单张须出最大")
			}
		}
	}
	if len(table) > 0 {
		b, e := Classify(table)
		if e != nil || !Beats(p, b) {
			return nil, p, errors.New("没有压过桌面牌")
		}
	}
	return h, p, nil
}

// Candidate enumerates actual legal subsets; useful for a hint and lawful test bots.
func Candidate(hand, table []int, first, nextOne bool) []int {
	for size := 1; size <= len(hand); size++ {
		for mask := 1; mask < (1 << len(hand)); mask++ {
			a := []int{}
			for i, c := range hand {
				if mask&(1<<i) != 0 {
					a = append(a, c)
				}
			}
			if len(a) != size {
				continue
			}
			if _, _, e := Validate(hand, a, table, first, nextOne); e == nil {
				return a
			}
		}
	}
	return nil
}
