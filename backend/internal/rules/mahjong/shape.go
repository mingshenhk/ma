// Package mahjong checks a structural three-meld-and-pair or five-pairs shape.
// It is NOT an original-game hu, turn, fan or payment adjudicator.
package mahjong

import "errors"

type Result struct {
	Standard  bool   `json:"standardShape"`
	FivePairs bool   `json:"fivePairsShape"`
	Warning   string `json:"warning"`
}

func Analyze(cards []int, red bool) (Result, error) {
	r := Result{Warning: "仅检查牌形；未验证原服胡牌限制、番数、优先级及结算"}
	if len(cards) < 2 || len(cards) > 11 || (len(cards)-2)%3 != 0 {
		return r, errors.New("结构分析需要2/5/8/11张牌")
	}
	var n [18]int
	wild := 0
	for _, c := range cards {
		if c == 0x35 && red {
			wild++
			if wild > 4 {
				return r, errors.New("红中超过4张")
			}
			continue
		}
		s, v := c>>4, c&15
		if s < 1 || s > 2 || v < 1 || v > 9 {
			return r, errors.New("候选玩法仅两门1..9及可选红中")
		}
		i := (s-1)*9 + v - 1
		n[i]++
		if n[i] > 4 {
			return r, errors.New("同牌超过4张")
		}
	}
	for i := 0; i < 18; i++ {
		for real := 0; real <= 2; real++ {
			if n[i] < real || 2-real > wild {
				continue
			}
			n[i] -= real
			if meld(n, wild-(2-real)) {
				r.Standard = true
			}
			n[i] += real
		}
	}
	if len(cards) == 11 {
		pairs, odd := 0, 0
		for _, v := range n {
			pairs += v / 2
			odd += v % 2
		}
		r.FivePairs = pairs+min(odd, wild)+(wild-min(odd, wild))/2 >= 5
	}
	return r, nil
}
func meld(n [18]int, w int) bool {
	i := -1
	for j, v := range n {
		if v > 0 {
			i = j
			break
		}
	}
	if i < 0 {
		return w%3 == 0
	}
	for real := 1; real <= min(3, n[i]); real++ {
		need := 3 - real
		if need <= w {
			a := n
			a[i] -= real
			if meld(a, w-need) {
				return true
			}
		}
	}
	// The lowest real tile can occupy any position when wild tiles precede it.
	for start := max((i/9)*9, i-2); start <= i && start%9 <= 6; start++ {
		a := n
		need := 0
		for j := start; j < start+3; j++ {
			if a[j] > 0 {
				a[j]--
			} else {
				need++
			}
		}
		if need <= w && meld(a, w-need) {
			return true
		}
	}
	return false
}
