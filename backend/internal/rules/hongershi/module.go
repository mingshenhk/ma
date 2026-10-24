// Package hongershi implements the two-player 红二十 rule proposal.
package hongershi

import "haojiayuan/internal/rules/capture"

const OriginalID = 110
const Players = 2
const HandSize = 8
const MinimumRed = 20

func Ordinary() []int {
	out := []int{}
	for suit := 0; suit < 4; suit++ {
		for v := 1; v <= 13; v++ {
			out = append(out, suit*16+v)
		}
	}
	return out
}
func Deck(kings int) []int {
	out := Ordinary()
	for i := 0; i < kings; i++ {
		out = append(out, 0x4f)
	}
	return out
}
func Pips(card int) int      { return card & 15 }
func Bonus(card int) bool    { return card == 0x4f }
func Colored(card int) bool  { return card>>4 == 1 || card>>4 == 2 }
func SameKind(a, b int) bool { return Pips(a) == Pips(b) }
func Red(card int, options map[string]int) int {
	v := Pips(card)
	r := 0
	if Colored(card) {
		r = v
		if v > 10 {
			r = 1
		}
	}
	if v == 5 {
		if r > 0 && options["redFive"] == 1 {
			r += 5
		}
		if r == 0 && options["blackFive"] == 1 {
			r += 5
		}
	}
	return r
}
func PairShape(h []int) bool {
	if len(h) == 0 {
		return true
	}
	if len(h)%2 != 0 {
		return false
	}
	for i := 1; i < len(h); i++ {
		if Pips(h[0])+Pips(h[i]) == 14 {
			rest := append([]int{}, h[1:i]...)
			rest = append(rest, h[i+1:]...)
			if PairShape(rest) {
				return true
			}
		}
	}
	return false
}

// Score implements poker red points, king melds and the independent 红二十 fan table.
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
	why := []string{}
	all := append([]int{}, c.Hand...)
	for _, m := range c.Melds {
		if m.Kind == "bonus" {
			f++
			why = append(why, "王副露")
			continue
		}
		all = append(all, m.Cards...)
		for _, v := range m.Cards {
			r += Red(v, c.Options)
		}
		if m.Kind == "peng" {
			f++
			why = append(why, "碰")
		}
		if m.Kind == "kong" {
			f += 2
			why = append(why, "杠")
		}
	}
	for _, v := range c.Hand {
		r += Red(v, c.Options)
	}
	allRed, allBlack := true, true
	counts := map[int]int{}
	for _, v := range all {
		if v>>4 > 3 || Pips(v) < 1 || Pips(v) > 13 {
			return capture.Value{}
		}
		counts[v]++
		if counts[v] > 1 {
			return capture.Value{}
		}
		allRed = allRed && Colored(v)
		allBlack = allBlack && !Colored(v)
	}
	if r < MinimumRed && !allRed && !allBlack {
		return capture.Value{}
	}
	if c.Report > 0 {
		f += 3
		why = append(why, "报胡")
	}
	if !c.Preview {
		if c.CatchReport {
			f += 6
			why = append(why, "抓报胡")
		}
		if c.Heavenly {
			f += 3
			why = append(why, "天胡")
		} else if c.Earthly {
			f += 3
			why = append(why, "地胡")
		}
		if c.SelfDraw {
			f++
			why = append(why, "自摸")
		}
		if c.KongEvent {
			f++
			why = append(why, "杠事件")
		}
	}
	if allRed {
		f += 3
		why = append(why, "全红")
	}
	if allBlack {
		f += 4
		why = append(why, "全黑")
	}
	f = min(c.Opt("cap", 4), f)
	mult := (1 << f) + 1
	if c.Opt("ladder", 0) == 1 {
		mult = f + 1
	}
	return capture.Value{OK: true, R: r, F: f, Amount: mult * c.Base, Reasons: why, ColorOnly: allRed || allBlack}
}
