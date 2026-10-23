package game

import (
	"fmt"
	"sort"
)

type bone struct {
	Code      int
	Name      string
	Pips, Red int
}

var bones = []bone{{0x1111, "地牌", 2, 4}, {0x1102, "丁丁", 3, 0}, {0x0202, "长二", 4, 0}, {0x1103, "和牌", 4, 1}, {0x0203, "拐子", 5, 0}, {0x1114, "幺四", 5, 10}, {0x0303, "长三", 6, 0}, {0x1105, "珠珠", 6, 1}, {0x0214, "二红", 6, 4}, {0x0205, "二五", 7, 0}, {0x1106, "幺六", 7, 1}, {0x0314, "三四", 7, 4}, {0x0206, "二六", 8, 0}, {0x0305, "三五", 8, 0}, {0x1414, "人牌", 8, 16}, {0x0306, "黑九", 9, 0}, {0x1405, "红九", 9, 4}, {0x0505, "梅子", 10, 0}, {0x1406, "四六", 10, 4}, {0x0506, "斧头", 11, 0}, {0x1606, "天牌", 12, 6}}

func isDS(s *State) bool { return s.Config.Game == "doushisi" }
func ordinary(s *State) []int {
	out := []int{}
	if isDS(s) {
		for _, b := range bones {
			if b.Code != 0x1102 && b.Code != 0x0506 {
				out = append(out, b.Code)
			}
		}
	} else {
		for suit := 0; suit < 4; suit++ {
			for v := 1; v <= 13; v++ {
				out = append(out, suit*16+v)
			}
		}
	}
	return out
}
func cpips(s *State, c int) int {
	if isDS(s) {
		for _, b := range bones {
			if b.Code == c {
				return b.Pips
			}
		}
		return 0
	}
	return c & 15
}
func cred(s *State, c int) int {
	if isDS(s) {
		for _, b := range bones {
			if b.Code == c {
				return b.Red
			}
		}
		return 0
	}
	v := c & 15
	r := 0
	if c>>4 == 1 || c>>4 == 2 {
		r = v
		if v > 10 {
			r = 1
		}
	}
	if v == 5 {
		if r > 0 && Opt(s, "redFive", 0) == 1 {
			r += 5
		}
		if r == 0 && Opt(s, "blackFive", 0) == 1 {
			r += 5
		}
	}
	return r
}
func colored(s *State, c int) bool {
	if isDS(s) {
		return cred(s, c) > 0
	}
	return c>>4 == 1 || c>>4 == 2
}
func bonus(s *State, c int) bool {
	if isDS(s) {
		return c == 0x1102 || c == 0x0506
	}
	return c == 0x4f
}
func sameKind(s *State, c, d int) bool {
	if isDS(s) {
		return c == d
	}
	return cpips(s, c) == cpips(s, d)
}
func groupCards(s *State, h []int, c int, n int) []int {
	out := []int{}
	for _, d := range h {
		if sameKind(s, c, d) {
			out = append(out, d)
			if len(out) == n {
				return out
			}
		}
	}
	return nil
}

// Different suits affect red points, so retain every distinct matching choice.
func groupChoices(s *State, h []int, c, n int) [][]int {
	matching := []int{}
	for _, d := range h {
		if sameKind(s, c, d) {
			matching = append(matching, d)
		}
	}
	out := [][]int{}
	var visit func(int, []int)
	visit = func(start int, chosen []int) {
		if len(chosen) == n {
			out = append(out, append([]int{}, chosen...))
			return
		}
		for i := start; i < len(matching); i++ {
			if i > start && matching[i] == matching[i-1] {
				continue
			}
			visit(i+1, append(chosen, matching[i]))
		}
	}
	sort.Ints(matching)
	visit(0, nil)
	return out
}
func groupMelds(seat Seat) []Meld {
	out := []Meld{}
	for _, m := range seat.Melds {
		if m.Kind != "bonus" && m.Kind != "red" {
			out = append(out, m)
		}
	}
	return out
}
func captureSize(s *State) int {
	if isDS(s) {
		return 12
	}
	return 8
}
func logical(s *State, p int) int { return len(s.Seats[p].Hand) + 2*len(groupMelds(s.Seats[p])) }
func startCapture(s *State) error {
	if isDS(s) {
		for _, b := range bones {
			for i := 0; i < 4; i++ {
				s.Wall = append(s.Wall, b.Code)
			}
		}
	} else {
		for _, c := range ordinary(s) {
			s.Wall = append(s.Wall, c)
		}
		for i := 0; i < Opt(s, "kings", 6); i++ {
			s.Wall = append(s.Wall, 0x4f)
		}
	}
	if e := Shuffle(s.Wall); e != nil {
		return e
	}
	for p := range s.Seats {
		n := captureSize(s) - 1
		if p == s.Dealer {
			n++
		}
		for j := 0; j < n; j++ {
			c, _ := take(s, false)
			s.Seats[p].Hand = append(s.Seats[p].Hand, c)
		}
		normalizeBonus(s, p)
		if s.End != "" {
			return nil
		}
		sort.Ints(s.Seats[p].Hand)
	}
	s.LastSource = "initial"
	s.Phase = "prepare"
	s.Turn = s.Dealer
	if isDS(s) && Opt(s, "allowFloat", 1) == 1 {
		s.Phase = "float"
	}
	return nil
}
func normalizeBonus(s *State, p int) {
	for {
		found := 0
		for _, c := range s.Seats[p].Hand {
			if bonus(s, c) {
				found = c
				break
			}
		}
		if found == 0 {
			return
		}
		s.Seats[p].Hand = remove(s.Seats[p].Hand, found)
		s.Seats[p].Melds = append(s.Seats[p].Melds, Meld{Kind: "bonus", Cards: []int{found}, From: p})
		c, ok := take(s, true)
		if !ok {
			finish(s, "补牌耗尽")
			return
		}
		s.Seats[p].Hand = append(s.Seats[p].Hand, c)
		s.LastCard = c
	}
}
func pairShape(s *State, h []int) bool {
	if len(h) == 0 {
		return true
	}
	if len(h)%2 != 0 {
		return false
	}
	c := h[0]
	for i := 1; i < len(h); i++ {
		d := h[i]
		if cpips(s, c)+cpips(s, d) == 14 && (!isDS(s) || c != d) {
			rest := append([]int{}, h[1:i]...)
			rest = append(rest, h[i+1:]...)
			if pairShape(s, rest) {
				return true
			}
		}
	}
	return false
}

type captureValue struct {
	OK           bool
	R, F, Amount int
	Reasons      []string
	ColorOnly    bool
}

func captureScore(s *State, p, card int, source string, preview bool) captureValue {
	seat := s.Seats[p]
	h := append([]int{}, seat.Hand...)
	if card != 0 {
		h = append(h, card)
	}
	if len(h)+2*len(groupMelds(seat)) != captureSize(s) || !pairShape(s, h) {
		return captureValue{}
	}
	r := 0
	allRed, allBlack := true, true
	all := append([]int{}, h...)
	f := 0
	why := []string{}
	for _, m := range seat.Melds {
		if m.Kind == "bonus" {
			if !isDS(s) {
				f++
				why = append(why, "王副露")
			}
			continue
		}
		all = append(all, m.Cards...)
		mr := 0
		for _, c := range m.Cards {
			mr += cred(s, c)
		}
		if isDS(s) && m.Kind == "peng" {
			r += max(10, mr)
		} else {
			r += mr
		}
		if !isDS(s) {
			if m.Kind == "peng" {
				f++
				why = append(why, "碰")
			}
			if m.Kind == "kong" {
				f += 2
				why = append(why, "杠")
			}
		}
	}
	for _, c := range h {
		r += cred(s, c)
	}
	for _, c := range all {
		allRed = allRed && colored(s, c)
		allBlack = allBlack && !colored(s, c)
	}
	copies := map[int]int{}
	for _, c := range all {
		copies[c]++
		limit := 1
		if isDS(s) {
			limit = 4
		}
		if copies[c] > limit {
			return captureValue{}
		}
	}
	threshold := 20
	if isDS(s) {
		threshold = 30
	}
	if r < threshold && !allRed && !allBlack {
		return captureValue{}
	}
	if seat.Report > 0 {
		f += 3
		if isDS(s) {
			f -= 2
		}
		why = append(why, "报胡")
	}
	if !preview && s.Offer != nil && s.Offer.From != p && s.Seats[s.Offer.From].Report > 0 {
		if isDS(s) { /* 斗十四无抓报加番 */
		} else {
			f += 6
			why = append(why, "抓报胡")
		}
	}
	if !preview {
		if p == s.Dealer && seat.Discards == 0 && seat.Draws == 0 && s.PublicActions == 0 {
			v := 3
			if isDS(s) {
				v = 2
			}
			f += v
			why = append(why, "天胡")
		} else if p != s.Dealer && seat.Draws <= 1 && s.Seats[s.Dealer].Discards == 1 && s.PublicActions == 0 {
			v := 3
			if isDS(s) {
				v = 2
			}
			f += v
			why = append(why, "地胡")
		}
	}
	if !isDS(s) {
		if allRed {
			f += 3
			why = append(why, "全红")
		}
		if allBlack {
			f += 4
			why = append(why, "全黑")
		}
		if !preview && (source == "flipself" || source == "private" || source == "initial") {
			f++
			why = append(why, "自摸")
		}
		if !preview && s.KongSeat >= 0 {
			f++
			why = append(why, "杠事件")
		}
	}
	if isDS(s) {
		names := map[int]bool{}
		counts := map[int]int{}
		for _, c := range all {
			counts[c]++
		}
		for _, m := range groupMelds(seat) {
			if m.Kind == "peng" {
				names[m.Cards[0]] = true
			}
		}
		five := []int{0x1606, 0x1111, 0x1414, 0x1103, 0x1114}
		for c, n := range counts {
			special := false
			for _, v := range five {
				special = special || v == c
			}
			if special && names[c] {
				f++
				if n == 4 {
					f++
				}
			} else if names[c] && n == 4 {
				f++
			}
		}
		if f > 0 {
			why = append(why, "拉牌/四张奖励")
		}
	}
	amount := 0
	if isDS(s) {
		cap := Opt(s, "cap", 500)
		drift := 1 << (seat.Float + s.Seats[1-p].Float)
		if allRed || allBlack {
			amount = cap
			why = append(why, "全红/全黑封顶")
		} else if Opt(s, "ladder", 0) == 1 {
			amount = min(cap, (r+10*f)*s.Config.Base*drift)
		} else {
			amount = min(cap, r*(1<<min(20, f))*s.Config.Base*drift)
		}
	} else {
		f = min(Opt(s, "cap", 4), f)
		mult := (1 << f) + 1
		if Opt(s, "ladder", 0) == 1 {
			mult = f + 1
		}
		amount = mult * s.Config.Base
	}
	return captureValue{OK: true, R: r, F: f, Amount: amount, Reasons: why, ColorOnly: allRed || allBlack}
}
func captureListen(s *State, p int) bool {
	for _, c := range ordinary(s) {
		if captureScore(s, p, c, "preview", true).OK {
			return true
		}
	}
	return false
}
func redTotal(s *State, p int) int {
	r := 0
	for _, c := range s.Seats[p].Hand {
		r += cred(s, c)
	}
	for _, m := range groupMelds(s.Seats[p]) {
		x := 0
		for _, c := range m.Cards {
			x += cred(s, c)
		}
		if isDS(s) && m.Kind == "peng" {
			x = max(10, x)
		}
		r += x
	}
	return r
}
func concealActions(s *State, p int) []Action {
	out := []Action{}
	if s.Seats[p].Report > 0 {
		return out
	}
	seen := map[int]bool{}
	for _, c := range s.Seats[p].Hand {
		k := c
		if !isDS(s) {
			k = cpips(s, c)
		}
		if seen[k] || bonus(s, c) {
			continue
		}
		seen[k] = true
		if len(s.Wall) > 0 {
			for _, cards := range groupChoices(s, s.Seats[p].Hand, c, 3) {
				out = append(out, a("concealed", "暗碰/暗拉", cards...))
			}
		}
		if !isDS(s) {
			if cards := groupCards(s, s.Seats[p].Hand, c, 4); len(cards) == 4 && len(s.Wall) >= 2 {
				out = append(out, a("concealedkong", "暗杠", cards...))
			}
		}
	}
	if !isDS(s) && s.LastSource == "private" && count(s.Seats[p].Hand, s.LastCard) > 0 {
		for _, c := range unique(s.Seats[p].Hand) {
			if c != s.LastCard && cpips(s, c)+cpips(s, s.LastCard) == 14 {
				out = append(out, a("privateeat", "补牌自吃", s.LastCard, c))
			}
		}
	}
	return out
}
func legalCapture(s *State, p int) []Action {
	seat := s.Seats[p]
	out := []Action{}
	if s.Phase == "float" {
		if seat.Missing == -1 {
			return []Action{{Kind: "float", Value: 0, Label: "不漂"}, {Kind: "float", Value: 1, Label: "漂：倍数×2"}}
		}
		return nil
	}
	if s.Phase == "prepare" {
		if s.Turn != p {
			return nil
		}
		out = append(out, concealActions(s, p)...)
		return append(out, a("prepared", "整理完成"))
	}
	if s.Phase == "declare" {
		if p == s.Dealer {
			return nil
		}
		out = append(out, a("declarepass", "不报胡"))
		threshold := 20
		if isDS(s) {
			threshold = 30
		}
		if redTotal(s, p) >= threshold && captureListen(s, p) {
			out = append(out, a("declare", "报胡"))
		}
		return out
	}
	if s.Offer != nil {
		o := s.Offer
		eligible := false
		for _, n := range o.Eligible {
			eligible = eligible || n == p
		}
		if !eligible {
			return nil
		}
		if _, ok := o.Replies[p]; ok {
			return nil
		}
		out = append(out, a("pass", "过"))
		source := o.Source
		if source == "flip" && o.From == p {
			source = "flipself"
		}
		if seat.Passed != cpips(s, o.Card) && captureScore(s, p, o.Card, source, false).OK {
			out = append(out, a("hu", "胡"))
		}
		if o.Source == "rob" || len(s.Wall) == 0 || seat.Report > 0 {
			return out
		}
		for _, cards := range groupChoices(s, seat.Hand, o.Card, 2) {
			out = append(out, a("peng", "碰/拉", cards...))
		}
		if !isDS(s) && len(groupCards(s, seat.Hand, o.Card, 3)) == 3 && len(s.Wall) >= 2 {
			out = append(out, a("kong", "直杠", groupCards(s, seat.Hand, o.Card, 3)...))
		}
		if !isDS(s) && o.Source == "flip" {
			for _, m := range groupMelds(seat) {
				if m.Kind == "peng" && sameKind(s, m.Cards[0], o.Card) && (m.Hidden || o.From == p) {
					out = append(out, a("upgrade", "升杠", o.Card))
				}
			}
		}
		for _, c := range unique(seat.Hand) {
			if cpips(s, c)+cpips(s, o.Card) == 14 && (!isDS(s) || c != o.Card) && (!(!isDS(s) && Opt(s, "selfEat", 0) == 1) || o.Source == "flip" && o.From == p) {
				out = append(out, a("eat", "吃", c))
			}
		}
		return out
	}
	if s.Turn != p {
		return nil
	}
	if s.Phase == "flip" {
		return append([]Action{a("draw", "翻牌")}, concealActions(s, p)...)
	}
	if s.Phase == "discard" {
		if captureScore(s, p, 0, s.LastSource, false).OK {
			out = append(out, a("hu", "胡"))
		}
		out = append(out, concealActions(s, p)...)
		for _, c := range unique(seat.Hand) {
			if seat.Report > 0 && c != s.LastCard {
				continue
			}
			out = append(out, a("discard", "出牌", c))
			if isDS(s) && p == s.Dealer && seat.Discards == 0 && seat.Draws == 0 && seat.Report == 0 {
				old := s.Seats[p].Hand
				s.Seats[p].Hand = remove(old, c)
				if redTotal(s, p) >= 30 && captureListen(s, p) {
					out = append(out, a("report", "报胡并出牌", c))
				}
				s.Seats[p].Hand = old
			}
		}
		if !isDS(s) && len(s.Wall) > 0 && seat.Report == 0 {
			for _, m := range groupMelds(seat) {
				if m.Kind == "peng" {
					for _, c := range seat.Hand {
						if c == s.LastCard && s.LastSource == "private" && sameKind(s, c, m.Cards[0]) {
							out = append(out, a("addkong", "补杠", c))
						}
					}
				}
			}
		}
	}
	return out
}
func capOffer(s *State, c, from, nxt int, source string) {
	o := &Offer{Card: c, From: from, Next: nxt, Source: source, Replies: map[int]Action{}}
	if source == "flip" {
		o.Eligible = []int{from, 1 - from}
	} else {
		o.Eligible = []int{1 - from}
	}
	s.Offer = o
	s.Phase = "respond"
}
func supplement(s *State, p, n int) {
	for i := 0; i < n; i++ {
		c, ok := take(s, true)
		if !ok {
			finish(s, "补牌耗尽 · 流局")
			return
		}
		s.Seats[p].Hand = append(s.Seats[p].Hand, c)
		s.LastCard = c
		normalizeBonus(s, p)
		if s.End != "" {
			return
		}
	}
	s.Seats[p].Passed = -1
	s.LastSource = "private"
	sort.Ints(s.Seats[p].Hand)
}
func applyCapture(s *State, p int, act Action) {
	seat := &s.Seats[p]
	switch act.Kind {
	case "float":
		seat.Float = act.Value
		seat.Missing = 0
		if s.Seats[1-p].Missing == 0 {
			s.Phase = "prepare"
		}
		return
	case "prepared":
		if p == s.Dealer {
			s.Turn = 1 - p
		} else {
			s.Turn = s.Dealer
			s.Phase = "declare"
		}
		return
	case "declare", "declarepass":
		if act.Kind == "declare" {
			seat.Report = 1
		}
		s.Phase = "discard"
		return
	case "draw":
		seat.Draws++
		seat.Passed = -1
		s.KongSeat = -1
		c, ok := take(s, false)
		if !ok {
			flowCapture(s)
			return
		}
		if bonus(s, c) {
			seat.Melds = append(seat.Melds, Meld{Kind: "bonus", Cards: []int{c}, From: p})
			supplement(s, p, 1)
			if s.End == "" {
				s.Phase = "discard"
			}
		} else {
			capOffer(s, c, p, 1-p, "flip")
			logEvent(s, p, "flip", []int{c}, "公翻")
		}
		return
	case "pass", "hu", "eat", "peng", "kong", "upgrade":
		if s.Offer != nil {
			if act.Kind == "pass" {
				src := s.Offer.Source
				if src == "flip" && s.Offer.From == p {
					src = "flipself"
				}
				if captureScore(s, p, s.Offer.Card, src, false).OK {
					seat.Passed = cpips(s, s.Offer.Card)
				}
			}
			s.Offer.Replies[p] = act
			if len(s.Offer.Replies) == len(s.Offer.Eligible) {
				resolveCapture(s)
			}
			return
		}
		if act.Kind == "hu" {
			winCapture(s, p, p, 0, s.LastSource)
			return
		}
	case "privateeat":
		seat.Hand = remove(seat.Hand, act.Cards...)
		seat.Melds = append(seat.Melds, Meld{Kind: "eat", Cards: act.Cards, From: p})
		s.LastSource = "claim"
		s.PublicActions++
		return
	case "concealed", "concealedkong":
		phase := s.Phase
		kind := "peng"
		n := 1
		if act.Kind == "concealedkong" {
			kind = "kong"
			n = 2
		}
		s.KongSeat = -1
		if kind == "kong" {
			s.KongSeat = p
		}
		seat.Hand = remove(seat.Hand, act.Cards...)
		seat.Melds = append(seat.Melds, Meld{Kind: kind, Cards: act.Cards, From: p, Hidden: true})
		supplement(s, p, n)
		if s.End == "" {
			s.Phase = phase
		}
		logEvent(s, p, act.Kind, nil, "暗组整理")
		return
	case "addkong":
		s.KongSeat = p
		capOffer(s, act.Cards[0], p, p, "rob")
		return
	case "discard", "report":
		if act.Kind == "report" {
			seat.Report = 1
		}
		c := act.Cards[0]
		seat.Hand = remove(seat.Hand, c)
		seat.Discards++
		s.Discards[p] = append(s.Discards[p], c)
		logEvent(s, p, "discard", []int{c}, "出牌")
		capOffer(s, c, p, 1-p, "discard")
		return
	}
}
func resolveCapture(s *State) {
	o := s.Offer
	order := []int{1 - o.From, o.From}
	if o.Source == "flip" {
		order = []int{o.From, 1 - o.From}
	}
	for p, reply := range o.Replies {
		if reply.Kind == "upgrade" {
			s.KongSeat = p
		}
	}
	for _, p := range order {
		if o.Replies[p].Kind == "hu" {
			src := o.Source
			if o.Source == "flip" && o.From == p {
				src = "flipself"
			}
			if o.Source == "rob" {
				s.Seats[o.From].Hand = remove(s.Seats[o.From].Hand, o.Card)
			}
			winCapture(s, p, o.From, o.Card, src)
			return
		}
	}
	for _, p := range order {
		if o.Replies[p].Kind != "upgrade" {
			continue
		}
		for i, m := range s.Seats[p].Melds {
			if m.Kind == "peng" && sameKind(s, m.Cards[0], o.Card) {
				s.Seats[p].Melds[i].Kind = "kong"
				s.Seats[p].Melds[i].Cards = append(m.Cards, o.Card)
				s.Offer = nil
				s.Turn = p
				s.Phase = "discard"
				s.KongSeat = p
				s.PublicActions++
				supplement(s, p, 1)
				return
			}
		}
	}
	if o.Source == "rob" {
		p := o.From
		s.Seats[p].Hand = remove(s.Seats[p].Hand, o.Card)
		for i, m := range s.Seats[p].Melds {
			if m.Kind == "peng" && sameKind(s, m.Cards[0], o.Card) {
				s.Seats[p].Melds[i].Kind = "kong"
				s.Seats[p].Melds[i].Cards = append(m.Cards, o.Card)
				break
			}
		}
		s.Offer = nil
		s.Turn = p
		s.Phase = "discard"
		supplement(s, p, 1)
		return
	}
	for _, kind := range []string{"kong", "peng", "eat"} {
		for _, p := range order {
			act := o.Replies[p]
			if act.Kind != kind {
				continue
			}
			seat := &s.Seats[p]
			seat.Hand = remove(seat.Hand, act.Cards...)
			cards := append(append([]int{}, act.Cards...), o.Card)
			seat.Melds = append(seat.Melds, Meld{Kind: kind, Cards: cards, From: o.From})
			if o.Source == "discard" && len(s.Discards[o.From]) > 0 {
				s.Discards[o.From] = s.Discards[o.From][:len(s.Discards[o.From])-1]
			}
			s.Offer = nil
			s.Turn = p
			s.Phase = "discard"
			s.PublicActions++
			s.LastSource = "claim"
			s.KongSeat = -1
			n := 0
			if kind == "peng" {
				n = 1
			}
			if kind == "kong" {
				n = 2
				s.KongSeat = p
			}
			if n > 0 {
				supplement(s, p, n)
			}
			logEvent(s, p, kind, cards, act.Label)
			return
		}
	}
	s.Offer = nil
	s.Discards[o.From] = append(s.Discards[o.From], o.Card)
	if o.Source == "discard" {
		s.Discards[o.From] = s.Discards[o.From][:len(s.Discards[o.From])-1]
	}
	s.Turn = o.Next
	s.Phase = "flip"
	s.KongSeat = -1
	if len(s.Wall) == 0 {
		flowCapture(s)
	}
}
func winCapture(s *State, p, from, card int, source string) {
	v := captureScore(s, p, card, source, false)
	if !v.OK {
		return
	}
	pay(s, 1-p, p, v.Amount, "胡牌", false)
	h := append([]int{}, s.Seats[p].Hand...)
	if card != 0 {
		h = append(h, card)
	}
	s.Wins = append(s.Wins, Win{Seat: p, From: from, Fan: v.F, Points: v.Amount, Reasons: append([]string{fmt.Sprintf("红点%d", v.R)}, v.Reasons...), Hand: h})
	s.Seats[p].Won = true
	finish(s, "胡牌结算")
}
func flowCapture(s *State) {
	if !isDS(s) && Opt(s, "checkListen", 1) == 1 {
		values := make([]captureValue, 2)
		for p := range s.Seats {
			for _, c := range ordinary(s) {
				v := captureScore(s, p, c, "preview", true)
				if v.OK && v.Amount > values[p].Amount {
					if Opt(s, "listenMode", 0) == 0 && redTotal(s, p) < 20 && !v.ColorOnly {
						continue
					}
					values[p] = v
				}
			}
		}
		for p := range s.Seats {
			if !values[p].OK && values[1-p].OK {
				pay(s, p, 1-p, values[1-p].Amount, "流局查叫", false)
			}
		}
	}
	finish(s, "牌墙用尽 · 流局")
}
