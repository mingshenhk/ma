package game

import (
	hz "haojiayuan/internal/rules/hongzhong"
	mj "haojiayuan/internal/rules/mahjong"
	nh "haojiayuan/internal/rules/nohongzhong"
	sc "haojiayuan/internal/rules/sichuan"
	"sort"
)

func mjPolicy(s *State) mj.Policy {
	switch s.Config.Game {
	case "hongzhong":
		return hz.Policy()
	case "nohongzhong":
		return nh.Policy()
	case "sichuan":
		return sc.Policy()
	default:
		panic("non-mahjong game used mahjong engine")
	}
}

const red = 0x35

type mjShape struct {
	Groups [][]int
	Pair   int
	Seven  bool
	Single int
}
type mjValue struct {
	Fan     int
	Reasons []string
	OK      bool
}

func tiles(s *State) []int {
	out := []int{}
	n := mjPolicy(s).Suits
	for suit := 1; suit <= n; suit++ {
		for v := 1; v <= 9; v++ {
			out = append(out, suit*16+v)
		}
	}
	return out
}
func startMJ(s *State) error {
	for _, c := range tiles(s) {
		for j := 0; j < 4; j++ {
			s.Wall = append(s.Wall, c)
		}
	}
	if mjPolicy(s).Wild {
		s.Wall = append(s.Wall, red, red, red, red)
	}
	if e := Shuffle(s.Wall); e != nil {
		return e
	}
	n := mjPolicy(s).InitialHand
	for p := range s.Seats {
		for j := 0; j < n; j++ {
			c, _ := take(s, false)
			s.Seats[p].Hand = append(s.Seats[p].Hand, c)
		}
		sort.Ints(s.Seats[p].Hand)
	}
	c, _ := take(s, false)
	s.Seats[s.Dealer].Hand = append(s.Seats[s.Dealer].Hand, c)
	s.LastCard = c
	s.LastSource = "initial"
	s.Phase = "declare"
	if mjPolicy(s).MissingSuit {
		s.Phase = "missing"
	}
	return nil
}
func mjShapes(s *State, hand []int, p int) []mjShape {
	if len(hand)%3 != 2 && !(len(hand) == 10 && mjPolicy(s).AskPairs) {
		return nil
	}
	extra := []mjShape{}
	if len(hand) == 11 && mjPolicy(s).AskPairs && len(groupMelds(s.Seats[p])) == 0 {
		for _, c := range unique(hand) {
			for _, sh := range mjShapes(s, remove(hand, c), p) {
				if !sh.Seven {
					continue
				}
				choices := []int{c}
				if c == red && mjPolicy(s).Wild {
					choices = tiles(s)
				}
				for _, logical := range choices {
					n := 0
					if sh.Pair == logical {
						n += 2
					}
					for _, g := range sh.Groups {
						for _, v := range g {
							if v == logical {
								n++
							}
						}
					}
					if n < 4 {
						copyShape := sh
						copyShape.Single = logical
						extra = append(extra, copyShape)
					}
				}
			}
		}
	}
	counts := map[int]int{}
	w := 0
	for _, c := range hand {
		if c == red && mjPolicy(s).Wild {
			w++
		} else {
			counts[c]++
		}
	}
	real := map[int]int{}
	for _, c := range hand {
		if c != red || !mjPolicy(s).Wild {
			real[c]++
		}
	}
	for _, m := range groupMelds(s.Seats[p]) {
		for _, c := range m.Cards {
			real[c]++
		}
	}
	for _, n := range real {
		if n > 4 {
			return nil
		}
	}
	out := extra
	all := tiles(s)
	limit := 4000
	check := func(g [][]int, pair int, seven bool) {
		logical := map[int]int{}
		for _, m := range groupMelds(s.Seats[p]) {
			for _, c := range m.Cards {
				logical[c]++
			}
		}
		logical[pair] += 2
		for _, row := range g {
			for _, c := range row {
				logical[c]++
			}
		}
		for _, v := range logical {
			if v > 4 {
				return
			}
		}
		copyG := make([][]int, len(g))
		for i := range g {
			copyG[i] = append([]int{}, g[i]...)
		}
		out = append(out, mjShape{Groups: copyG, Pair: pair, Seven: seven})
	}
	var meld func(int, [][]int, int)
	meld = func(wild int, g [][]int, pair int) {
		if len(out) > limit {
			return
		}
		first := 0
		for _, c := range all {
			if counts[c] > 0 {
				first = c
				break
			}
		}
		if first == 0 {
			if wild == 0 {
				check(g, pair, false)
				return
			}
			if wild == 3 {
				for _, c := range all {
					check(append(g, []int{c, c, c}), pair, false)
				}
				for _, c := range all {
					if c&15 <= 7 {
						check(append(g, []int{c, c + 1, c + 2}), pair, false)
					}
				}
			}
			return
		}
		for n := 1; n <= min(3, counts[first]); n++ {
			if 3-n <= wild {
				counts[first] -= n
				meld(wild-3+n, append(g, []int{first, first, first}), pair)
				counts[first] += n
			}
		}
		for start := max(first-2, (first>>4)*16+1); start <= first && start&15 <= 7; start++ {
			row := []int{start, start + 1, start + 2}
			for mask := 0; mask < 8; mask++ {
				use := []int{}
				need := 0
				hasFirst := false
				ok := true
				for j, c := range row {
					if mask&(1<<j) != 0 {
						if counts[c] == 0 {
							ok = false
							break
						}
						use = append(use, c)
						hasFirst = hasFirst || c == first
					} else {
						need++
					}
				}
				if !ok || !hasFirst || need > wild {
					continue
				}
				for _, c := range use {
					counts[c]--
				}
				meld(wild-need, append(g, row), pair)
				for _, c := range use {
					counts[c]++
				}
			}
		}
	}
	if len(hand)%3 == 2 {
		for _, c := range all {
			for n := 0; n <= min(2, counts[c]); n++ {
				if 2-n > w {
					continue
				}
				counts[c] -= n
				meld(w-2+n, nil, c)
				counts[c] += n
			}
		}
	}
	target := mjPolicy(s).PairCount
	if len(groupMelds(s.Seats[p])) == 0 && len(hand) == target*2 {
		var pairs func(int, [][]int)
		pairs = func(wild int, g [][]int) {
			first := 0
			for _, c := range all {
				if counts[c] > 0 {
					first = c
					break
				}
			}
			if first == 0 {
				if wild == 0 && len(g) == target {
					check(g[:len(g)-1], g[len(g)-1][0], true)
				}
				if wild >= 2 && len(g) < target {
					for _, c := range all {
						pairs(wild-2, append(g, []int{c, c}))
					}
				}
				return
			}
			for n := 1; n <= min(2, counts[first]); n++ {
				if 2-n > wild {
					continue
				}
				counts[first] -= n
				pairs(wild-2+n, append(g, []int{first, first}))
				counts[first] += n
			}
		}
		pairs(w, nil)
	}
	return out
}
func mjScore(s *State, p, card int, source string, preview bool) mjValue {
	seat := s.Seats[p]
	h := append([]int{}, seat.Hand...)
	if card != 0 {
		h = append(h, card)
	}
	want := mjPolicy(s).InitialHand + 1
	if len(h)+3*len(groupMelds(seat)) != want {
		return mjValue{}
	}
	if mjPolicy(s).Wild && card == red && source != "self" && source != "initial" {
		return mjValue{}
	}
	if mjPolicy(s).MissingSuit {
		for _, c := range h {
			if c>>4 == seat.Missing {
				return mjValue{}
			}
		}
		for _, m := range seat.Melds {
			for _, c := range m.Cards {
				if c>>4 == seat.Missing {
					return mjValue{}
				}
			}
		}
	}
	best := mjValue{}
	for _, sh := range mjShapes(s, h, p) {
		suits := map[int]bool{}
		no19 := true
		terminal := sh.Pair&15 == 1 || sh.Pair&15 == 9
		pp := true
		jiang := sh.Pair&15 == 2 || sh.Pair&15 == 5 || sh.Pair&15 == 8
		cards := []int{sh.Pair, sh.Pair}
		if sh.Single != 0 {
			cards = append(cards, sh.Single)
		}
		groups := append([][]int{}, sh.Groups...)
		for _, m := range groupMelds(seat) {
			groups = append(groups, m.Cards)
		}
		ka := 0
		for _, g := range groups {
			cards = append(cards, g...)
			isTrip := len(g) > 0
			hasTerminal := false
			for _, c := range g {
				isTrip = isTrip && c == g[0]
				hasTerminal = hasTerminal || c&15 == 1 || c&15 == 9
			}
			if !sh.Seven {
				pp = pp && isTrip
				terminal = terminal && hasTerminal
			}
			for _, c := range g {
				jiang = jiang && (c&15 == 2 || c&15 == 5 || c&15 == 8)
			}
			if len(g) == 3 && g[0]&15 == 4 && g[1] == g[0]+1 && g[2] == g[1]+1 {
				ka++
			}
		}
		for _, c := range cards {
			suits[c>>4] = true
			no19 = no19 && c&15 != 1 && c&15 != 9
		}
		clear := len(suits) == 1
		f, baseName := mjPolicy(s).Base(mj.Features{Pungs: pp, Clear: clear, Pairs: sh.Seven, Terminal: terminal, Jiang: jiang})
		why := []string{baseName}
		set := func(n int, t string) {
			if n > f {
				f = n
				why = []string{t}
			}
		}
		if !preview && p == s.Dealer && seat.Discards == 0 && seat.Draws == 0 && s.PublicActions == 0 {
			set(6, "天胡")
		}
		if !preview && p != s.Dealer && seat.Draws <= 1 && s.Seats[s.Dealer].Discards == 1 && s.PublicActions == 0 && (mjPolicy(s).MissingSuit || seat.Report > 0) {
			set(6, "地胡")
		}
		add := func(n int, t string) {
			if n > 0 {
				f += n
				why = append(why, t)
			}
		}
		if mjPolicy(s).ExtraFan && no19 {
			add(1, "断幺九")
		}
		physical := map[int]int{}
		for _, c := range h {
			if c != red || !mjPolicy(s).Wild {
				physical[c]++
			}
		}
		for _, m := range groupMelds(seat) {
			for _, c := range m.Cards {
				physical[c]++
			}
		}
		roots := 0
		for _, n := range physical {
			if n == 4 {
				roots++
			}
		}
		if Opt(s, "root", 1) == 1 {
			add(min(3, roots), "根")
		}
		if mjPolicy(s).ExtraFan && Opt(s, "ka", 1) == 1 {
			if source == "self" || source == "initial" {
				add(ka, "456卡")
			} else if card&15 == 5 && count(h, card-1) > 0 && count(h, card+1) > 0 && ka > 0 {
				add(1, "456卡")
			}
		}
		if seat.Report > 0 {
			add(2+seat.ReportKongs, "报胡")
			if seat.Report == 2 && !sh.Seven {
				continue
			}
		}
		if !preview {
			if s.KongSeat >= 0 {
				switch source {
				case "self":
					if s.KongSeat == p {
						add(1, "杠上花")
					}
				case "discard":
					if s.KongSeat != p {
						add(1, "杠上炮")
					}
				case "rob":
					add(1, "抢杠胡")
				}
			}
			if len(s.Wall) == 0 && source != "initial" {
				add(1, "海底")
			}
			if s.Offer != nil {
				from := s.Seats[s.Offer.From]
				if s.Offer.From != p {
					if from.Report == 1 {
						add(2+from.ReportKongs, "抓报胡")
					}
					if s.Offer.Source == "ask" {
						if from.Report == 2 {
							add(4, "抓报请胡")
						} else {
							add(2, "抓请胡")
						}
					}
				}
			}
		}
		if !preview && source != "self" && source != "initial" && f == 1 && mjPolicy(s).AskPairs && source != "ask" && s.KongSeat < 0 {
			continue
		}
		if f < Opt(s, "minFan", 1) {
			continue
		}
		if !best.OK || min(6, f) > best.Fan {
			best = mjValue{min(6, f), why, true}
		}
	}
	return best
}
func waitsMJ(s *State, p int, hand []int) []int {
	old := s.Seats[p].Hand
	s.Seats[p].Hand = hand
	out := []int{}
	for _, c := range tiles(s) {
		if mjScore(s, p, c, "preview", true).OK {
			out = append(out, c)
		}
	}
	s.Seats[p].Hand = old
	return out
}
func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func reportKongOK(s *State, p int, cards []int, kind string) bool {
	seat := &s.Seats[p]
	if seat.Report == 0 {
		return true
	}
	if seat.Report == 2 || seat.ReportKongs >= 3 {
		return false
	}
	waiting := seat.Hand
	if s.Offer == nil {
		waiting = remove(waiting, s.LastCard)
	}
	before := waitsMJ(s, p, waiting)
	oldH, oldM := seat.Hand, seat.Melds
	seat.Melds = append([]Meld{}, seat.Melds...)
	for i := range seat.Melds {
		seat.Melds[i].Cards = append([]int{}, seat.Melds[i].Cards...)
	}
	seat.Hand = remove(seat.Hand, cards...)
	if kind == "addkong" {
		for i := range seat.Melds {
			if seat.Melds[i].Kind == "peng" && seat.Melds[i].Cards[0] == cards[0] {
				seat.Melds[i].Cards = append(seat.Melds[i].Cards, cards[0])
				seat.Melds[i].Kind = "kong"
				break
			}
		}
	} else {
		kong := append([]int{}, cards...)
		if len(kong) == 3 {
			kong = append(kong, cards[0])
		}
		seat.Melds = append(append([]Meld{}, seat.Melds...), Meld{Kind: "kong", Cards: kong})
	}
	after := waitsMJ(s, p, seat.Hand)
	seat.Hand, seat.Melds = oldH, oldM
	return len(after) > 0 && sameInts(before, after)
}
func legalMJ(s *State, p int) []Action {
	seat := &s.Seats[p]
	out := []Action{}
	if s.Phase == "missing" {
		if seat.Missing == -1 {
			for i := 1; i <= 3; i++ {
				out = append(out, Action{Kind: "missing", Value: i, Label: []string{"", "缺筒", "缺条", "缺万"}[i]})
			}
		}
		return out
	}
	if s.Phase == "declare" {
		if p != s.Dealer && seat.Missing == -1 {
			out = append(out, a("declarepass", "不报胡"))
			if len(waitsMJ(s, p, seat.Hand)) > 0 {
				out = append(out, Action{Kind: "declare", Value: 1, Label: "报胡 +2番"})
			}
			pairs := 0
			for _, c := range unique(seat.Hand) {
				pairs += count(seat.Hand, c) / 2
			}
			if pairs >= 4 {
				out = append(out, Action{Kind: "declare", Value: 2, Label: "报请胡"})
			}
		}
		return out
	}
	if s.Offer != nil {
		if _, ok := s.Offer.Replies[p]; ok {
			return nil
		}
		eligible := false
		for _, n := range s.Offer.Eligible {
			eligible = eligible || p == n
		}
		if !eligible {
			return nil
		}
		o := s.Offer
		out = append(out, a("pass", "过"))
		if seat.Passed < 0 && mjScore(s, p, o.Card, o.Source, false).OK {
			out = append(out, a("hu", "胡"))
		}
		if o.Source == "rob" || o.Source == "ask" || len(s.Wall) == 0 {
			return out
		}
		if o.Card == red && mjPolicy(s).Wild || mjPolicy(s).MissingSuit && o.Card>>4 == seat.Missing {
			return out
		}
		n := count(seat.Hand, o.Card)
		if n >= 2 && seat.Report == 0 {
			out = append(out, a("peng", "碰", o.Card, o.Card))
		}
		if n >= 3 && reportKongOK(s, p, []int{o.Card, o.Card, o.Card}, "kong") {
			out = append(out, a("kong", "直杠", o.Card, o.Card, o.Card))
		}
		return out
	}
	if s.Turn != p {
		return nil
	}
	if mjScore(s, p, 0, s.LastSource, false).OK {
		sh := mjShapes(s, seat.Hand, p)
		five := false
		for _, v := range sh {
			five = five || v.Seven
		}
		if five && mjPolicy(s).AskPairs {
			for _, c := range askCards(s, p) {
				out = append(out, a("ask", "请胡", c))
			}
		} else {
			out = append(out, a("hu", "自摸"))
		}
	}
	missing := false
	if mjPolicy(s).MissingSuit {
		for _, c := range seat.Hand {
			missing = missing || c>>4 == seat.Missing
		}
	}
	for _, c := range unique(seat.Hand) {
		if seat.Report > 0 && s.LastSource != "initial" && c != s.LastCard {
			continue
		}
		if missing && c>>4 != seat.Missing {
			continue
		}
		if c == red && mjPolicy(s).Wild && len(unique(seat.Hand)) > 1 {
			continue
		}
		out = append(out, a("discard", "出牌", c))
		if mjPolicy(s).AskPairs && p == s.Dealer && seat.Discards == 0 && seat.Draws == 0 && seat.Report == 0 {
			h := remove(seat.Hand, c)
			if len(waitsMJ(s, p, h)) > 0 {
				out = append(out, a("report", "报胡并出牌", c))
			}
			pairs := 0
			for _, v := range unique(h) {
				pairs += count(h, v) / 2
			}
			if pairs >= 4 {
				out = append(out, a("reportask", "报请并出牌", c))
			}
		}
	}
	if len(s.Wall) > 0 && seat.Report != 2 {
		for _, c := range unique(seat.Hand) {
			if c == red && mjPolicy(s).Wild || missing && c>>4 == seat.Missing {
				continue
			}
			if count(seat.Hand, c) == 4 && reportKongOK(s, p, []int{c, c, c, c}, "concealed") {
				out = append(out, a("concealed", "暗杠", c, c, c, c))
			}
			for _, m := range seat.Melds {
				if m.Kind == "peng" && m.Cards[0] == c && reportKongOK(s, p, []int{c}, "addkong") {
					out = append(out, a("addkong", "补杠", c))
				}
			}
		}
	}
	return out
}
func askCards(s *State, p int) []int {
	out := []int{}
	for _, c := range unique(s.Seats[p].Hand) {
		for _, sh := range mjShapes(s, s.Seats[p].Hand, p) {
			if sh.Seven {
				remaining := remove(s.Seats[p].Hand, c)
				wild := count(remaining, red)
				odd := 0
				for _, v := range unique(remaining) {
					if v != red {
						odd += count(remaining, v) % 2
					}
				}
				if odd <= wild && (wild-odd)%2 == 0 {
					out = append(out, c)
				}
				break
			}
		}
	}
	return out
}
func offerMJ(s *State, card, from, nxt int, source string) {
	s.Offer = &Offer{Card: card, From: from, Source: source, Next: nxt, Replies: map[int]Action{}}
	for p := range s.Seats {
		if p != from && !s.Seats[p].Won {
			s.Offer.Eligible = append(s.Offer.Eligible, p)
		}
	}
	s.Phase = "respond"
}
func drawMJ(s *State, p int, tail bool) {
	c, ok := take(s, tail)
	if !ok {
		flowMJ(s)
		return
	}
	seat := &s.Seats[p]
	seat.Hand = append(seat.Hand, c)
	sort.Ints(seat.Hand)
	seat.Draws++
	seat.Passed = -1
	s.Turn = p
	s.Phase = "discard"
	s.LastCard = c
	s.LastSource = "self"
	if mjPolicy(s).Wild && seat.Report > 0 {
		for c == red {
			seat.Hand = remove(seat.Hand, c)
			seat.Melds = append(seat.Melds, Meld{Kind: "red", Cards: []int{red}, From: p})
			logEvent(s, p, "red", []int{red}, "请出新摸红中")
			c, ok = take(s, true)
			if !ok {
				flowMJ(s)
				return
			}
			seat.Hand = append(seat.Hand, c)
			s.LastCard = c
		}
	}
	sort.Ints(seat.Hand)
}
func applyMJ(s *State, p int, act Action) {
	seat := &s.Seats[p]
	switch act.Kind {
	case "missing":
		seat.Missing = act.Value
		all := true
		for _, x := range s.Seats {
			all = all && x.Missing > 0
		}
		if all {
			s.Phase = "discard"
		}
		return
	case "declare", "declarepass":
		seat.Missing = 0
		if act.Kind == "declare" {
			seat.Report = act.Value
		}
		s.Phase = "discard"
		return
	case "pass", "hu", "peng", "kong":
		if s.Offer != nil {
			if act.Kind == "pass" && mjScore(s, p, s.Offer.Card, s.Offer.Source, false).OK {
				seat.Passed = 1
			}
			s.Offer.Replies[p] = act
			if len(s.Offer.Replies) == len(s.Offer.Eligible) {
				resolveMJ(s)
			}
			return
		}
		if act.Kind == "hu" {
			winMJ(s, p, p, 0, s.LastSource)
			if s.End == "" {
				s.KongSeat = -1
				drawMJ(s, next(s, p), false)
			}
			return
		}
	case "ask":
		offerMJ(s, act.Cards[0], p, p, "ask")
		logEvent(s, p, "ask", act.Cards, "请胡：先询问对手")
		return
	case "report", "reportask", "discard":
		if act.Kind == "report" {
			seat.Report = 1
		}
		if act.Kind == "reportask" {
			seat.Report = 2
		}
		c := act.Cards[0]
		seat.Hand = remove(seat.Hand, c)
		seat.Discards++
		s.Discards[p] = append(s.Discards[p], c)
		s.First = false
		logEvent(s, p, "discard", []int{c}, "出牌")
		offerMJ(s, c, p, next(s, p), "discard")
		return
	case "concealed":
		seat.Hand = remove(seat.Hand, act.Cards...)
		seat.Melds = append(seat.Melds, Meld{Kind: "kong", Cards: act.Cards, From: p, Hidden: true})
		for n := range s.Seats {
			if n != p && !s.Seats[n].Won {
				pay(s, n, p, 2*s.Config.Base, "暗杠", true)
			}
		}
		s.PublicActions++
		s.KongSeat = p
		if seat.Report > 0 {
			seat.ReportKongs++
		}
		logEvent(s, p, "concealed", nil, "暗杠")
		drawMJ(s, p, true)
		return
	case "addkong":
		s.KongSeat = p
		offerMJ(s, act.Cards[0], p, p, "rob")
		logEvent(s, p, "addkong", act.Cards, "申请补杠")
		return
	}
}
func resolveMJ(s *State) {
	o := s.Offer
	winners := []int{}
	for i := 1; i < len(s.Seats); i++ {
		p := (o.From + i) % len(s.Seats)
		if o.Replies[p].Kind == "hu" {
			winners = append(winners, p)
		}
	}
	if len(winners) > 0 {
		if o.Source == "ask" {
			s.Seats[o.From].Hand = remove(s.Seats[o.From].Hand, o.Card)
		}
		if o.Source == "rob" {
			s.Seats[o.From].Hand = remove(s.Seats[o.From].Hand, o.Card)
		}
		for _, p := range winners {
			winMJ(s, p, o.From, o.Card, o.Source)
		}
		if s.End == "" {
			s.Offer = nil
			s.KongSeat = -1
			drawMJ(s, next(s, o.From), false)
		}
		return
	}
	if o.Source == "ask" {
		s.Offer = nil
		winMJ(s, o.From, o.From, 0, s.LastSource)
		return
	}
	if o.Source == "rob" {
		p := o.From
		c := o.Card
		seat := &s.Seats[p]
		seat.Hand = remove(seat.Hand, c)
		for i := range seat.Melds {
			if seat.Melds[i].Kind == "peng" && seat.Melds[i].Cards[0] == c {
				seat.Melds[i].Kind = "kong"
				seat.Melds[i].Cards = append(seat.Melds[i].Cards, c)
			}
		}
		for n := range s.Seats {
			if n != p && !s.Seats[n].Won {
				pay(s, n, p, s.Config.Base, "补杠", true)
			}
		}
		if seat.Report > 0 {
			seat.ReportKongs++
		}
		s.PublicActions++
		s.Offer = nil
		drawMJ(s, p, true)
		return
	}
	for _, priority := range []string{"kong", "peng"} {
		for i := 1; i < len(s.Seats); i++ {
			p := (o.From + i) % len(s.Seats)
			act := o.Replies[p]
			if act.Kind != priority {
				continue
			}
			seat := &s.Seats[p]
			seat.Hand = remove(seat.Hand, act.Cards...)
			cards := append(append([]int{}, act.Cards...), o.Card)
			kind := "peng"
			if act.Kind == "kong" {
				kind = "kong"
			}
			seat.Melds = append(seat.Melds, Meld{Kind: kind, Cards: cards, From: o.From})
			if len(s.Discards[o.From]) > 0 {
				s.Discards[o.From] = s.Discards[o.From][:len(s.Discards[o.From])-1]
			}
			s.PublicActions++
			s.Offer = nil
			s.Turn = p
			s.Phase = "discard"
			s.LastSource = "claim"
			logEvent(s, p, kind, cards, act.Label)
			if kind == "kong" {
				pay(s, o.From, p, 2*s.Config.Base, "直杠", true)
				if seat.Report > 0 {
					seat.ReportKongs++
				}
				s.KongSeat = p
				drawMJ(s, p, true)
			} else {
				s.KongSeat = -1
			}
			return
		}
	}
	s.Offer = nil
	s.KongSeat = -1
	drawMJ(s, o.Next, false)
}
func winMJ(s *State, p, from, card int, source string) {
	v := mjScore(s, p, card, source, false)
	if !v.OK {
		return
	}
	self := p == from
	amount := s.Config.Base * fanMultiplier(v.Fan)
	total := 0
	if self {
		for i := range s.Seats {
			if i != p && !s.Seats[i].Won {
				n := amount + 2*s.Config.Base
				pay(s, i, p, n, "自摸", false)
				total += n
			}
		}
	} else {
		pay(s, from, p, amount, "点炮", false)
		total = amount
	}
	h := append([]int{}, s.Seats[p].Hand...)
	if card != 0 {
		h = append(h, card)
	}
	s.Wins = append(s.Wins, Win{p, from, v.Fan, total, v.Reasons, h})
	s.Seats[p].Won = true
	logEvent(s, p, "hu", []int{card}, describeScore(v.Fan))
	if len(s.Wins) >= mjPolicy(s).EndAfterWins {
		finish(s, "胡牌结算")
	}
}
func flowMJ(s *State) {
	flower := make([]bool, len(s.Seats))
	listening := make([]mjValue, len(s.Seats))
	for p := range s.Seats {
		if s.Seats[p].Won {
			continue
		}
		if mjPolicy(s).MissingSuit {
			for _, c := range s.Seats[p].Hand {
				if c>>4 == s.Seats[p].Missing {
					flower[p] = true
				}
			}
		}
		if !flower[p] {
			for _, c := range tiles(s) {
				v := mjScore(s, p, c, "preview", true)
				if v.OK && (!listening[p].OK || v.Fan > listening[p].Fan) {
					listening[p] = v
				}
			}
		}
	}
	for p := range s.Seats {
		if s.Seats[p].Won {
			continue
		}
		for q := range s.Seats {
			if q == p || s.Seats[q].Won {
				continue
			}
			if flower[p] && !flower[q] {
				pay(s, p, q, 32*s.Config.Base, "查花猪", false)
			} else if !flower[p] && !listening[p].OK && listening[q].OK {
				pay(s, p, q, fanMultiplier(listening[q].Fan)*s.Config.Base, "查大叫", false)
			}
		}
	}
	original := append([]Transfer{}, s.Ledger...)
	for _, t := range original {
		if t.Gang && !s.Seats[t.To].Won && (flower[t.To] || !listening[t.To].OK) {
			pay(s, t.To, t.From, t.Amount, "退杠", false)
		}
	}
	finish(s, "牌墙用尽 · 流局查叫")
}
