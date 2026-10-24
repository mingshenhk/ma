package doushisi

import "haojiayuan/internal/rules/capture"

const OriginalID = 95
const Players = 2
const HandSize = 12
const MinimumRed = 30

func Ordinary() []int {
	out := []int{}
	for _, t := range Tiles {
		if !Bonus(t.Code) {
			out = append(out, t.Code)
		}
	}
	return out
}
func Deck() []int {
	out := []int{}
	for _, t := range Tiles {
		for i := 0; i < 4; i++ {
			out = append(out, t.Code)
		}
	}
	return out
}
func Pips(card int) int      { t, _ := Find(card); return t.Pips }
func Red(card int) int       { t, _ := Find(card); return t.Red }
func Bonus(card int) bool    { return card == 0x1102 || card == 0x0506 }
func Colored(card int) bool  { return Red(card) > 0 }
func SameKind(a, b int) bool { return a == b }
func PairShape(h []int) bool {
	if len(h) == 0 {
		return true
	}
	if len(h)%2 != 0 {
		return false
	}
	for i := 1; i < len(h); i++ {
		if h[0] != h[i] && Pips(h[0])+Pips(h[i]) == 14 {
			rest := append([]int{}, h[1:i]...)
			rest = append(rest, h[i+1:]...)
			if PairShape(rest) {
				return true
			}
		}
	}
	return false
}

// Score implements the 84-tile, red-point and floating-stake rules of 斗十四.
func Score(c capture.Context) capture.Value {
	groups := 0
	for _, m := range c.Melds {
		if m.Kind != "bonus" && m.Kind != "red" {
			groups++
		}
	}
	if len(c.Hand)+2*groups != HandSize || !PairShape(c.Hand) {
		return capture.Value{}
	}
	r, f := 0, 0
	all := append([]int{}, c.Hand...)
	why := []string{}
	peng := map[int]bool{}
	for _, m := range c.Melds {
		if m.Kind == "bonus" {
			continue
		}
		all = append(all, m.Cards...)
		mr := 0
		for _, v := range m.Cards {
			mr += Red(v)
		}
		if m.Kind == "peng" {
			r += max(10, mr)
			if len(m.Cards) > 0 {
				peng[m.Cards[0]] = true
			}
		} else {
			r += mr
		}
	}
	for _, v := range c.Hand {
		r += Red(v)
	}
	allRed, allBlack := true, true
	counts := map[int]int{}
	for _, v := range all {
		if _, ok := Find(v); !ok || Bonus(v) {
			return capture.Value{}
		}
		counts[v]++
		if counts[v] > 4 {
			return capture.Value{}
		}
		allRed = allRed && Colored(v)
		allBlack = allBlack && !Colored(v)
	}
	if r < MinimumRed && !allRed && !allBlack {
		return capture.Value{}
	}
	if c.Report > 0 {
		f++
		why = append(why, "报胡")
	}
	if !c.Preview {
		if c.Heavenly {
			f += 2
			why = append(why, "天胡")
		} else if c.Earthly {
			f += 2
			why = append(why, "地胡")
		}
	}
	bonusFan := 0
	for v, n := range counts {
		special := v == 0x1606 || v == 0x1111 || v == 0x1414 || v == 0x1103 || v == 0x1114
		if special && peng[v] {
			bonusFan++
			if n == 4 {
				bonusFan++
			}
		} else if peng[v] && n == 4 {
			bonusFan++
		}
	}
	f += bonusFan
	if bonusFan > 0 {
		why = append(why, "拉牌/四张奖励")
	}
	cap := c.Opt("cap", 500)
	drift := 1 << (c.Float + c.OtherFloat)
	amount := 0
	if allRed || allBlack {
		amount = cap
		why = append(why, "全红/全黑封顶")
	} else if c.Opt("ladder", 0) == 1 {
		amount = min(cap, (r+10*f)*c.Base*drift)
	} else {
		amount = min(cap, r*(1<<min(20, f))*c.Base*drift)
	}
	return capture.Value{OK: true, R: r, F: f, Amount: amount, Reasons: why, ColorOnly: allRed || allBlack}
}
