package game

import (
	rf "haojiayuan/internal/rules/runfast"
	"sort"
)

func startRF(s *State) error {
	deck := rf.Deck()
	if Opt(s, "random", 0) == 0 {
		deck = nil
		for suit := 0; suit < 4; suit++ {
			deck = append(deck, suit*16+5, suit*16+1)
		}
		for _, suit := range []int{0, 2, 3} {
			for v := 9; v <= 13; v++ {
				deck = append(deck, suit*16+v)
			}
		}
	}
	for {
		if e := Shuffle(deck); e != nil {
			return e
		}
		if count(deck[:20], 0x35) > 0 {
			break
		}
	}
	s.Seats[0].Hand = append([]int{}, deck[:10]...)
	s.Seats[1].Hand = append([]int{}, deck[10:20]...)
	s.Wall = append([]int{}, deck[20:]...)
	for p := range s.Seats {
		rf.Sort(s.Seats[p].Hand)
		if count(s.Seats[p].Hand, 0x35) > 0 {
			s.Turn = p
			s.Dealer = p
		}
	}
	s.Phase = "play"
	if Opt(s, "guan", 1) == 1 {
		s.Phase = "guan"
	}
	return nil
}
func legalRF(s *State, p int) []Action {
	if s.Phase == "guan" {
		if s.Seats[p].Missing >= 0 {
			return nil
		}
		out := []Action{a("guanskip", "不关牌")}
		for _, v := range []int{5, 1} {
			cards := []int{}
			for _, c := range s.Seats[p].Hand {
				if c&15 == v {
					cards = append(cards, c)
				}
			}
			if len(cards) == 4 {
				out = append(out, a("guan", "关牌：20底分", cards...))
			}
		}
		return out
	}
	if s.Turn != p {
		return nil
	}
	h := s.Seats[p].Hand
	out := []Action{}
	for mask := 1; mask < (1 << len(h)); mask++ {
		cards := []int{}
		for i, c := range h {
			if mask&(1<<i) != 0 {
				cards = append(cards, c)
			}
		}
		if _, _, e := rf.Validate(h, cards, s.Table, s.First, len(s.Seats[1-p].Hand) == 1); e == nil {
			out = append(out, a("play", "出牌", cards...))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].Cards) < len(out[j].Cards) })
	if len(out) == 0 && len(s.Table) > 0 {
		out = append(out, a("pass", "要不起"))
	}
	return out
}
func applyRF(s *State, p int, act Action) {
	switch act.Kind {
	case "guanskip":
		s.Seats[p].Missing = 0
	case "guan":
		s.Seats[p].Missing = 1
	case "pass":
		s.Table = nil
		s.Turn = 1 - p
		logEvent(s, p, "pass", nil, "要不起，对手领出")
	case "play":
		pattern, _ := rf.Classify(act.Cards)
		if pattern.Kind == "bomb" {
			s.Bombs++
		}
		s.Seats[p].Hand = remove(s.Seats[p].Hand, act.Cards...)
		s.Seats[p].Discards++
		s.Table = append([]int{}, act.Cards...)
		s.TableSeat = p
		s.First = false
		logEvent(s, p, "play", act.Cards, "出牌")
		if len(s.Seats[p].Hand) == 0 {
			n := len(s.Seats[1-p].Hand)
			amount := 0
			cap := Opt(s, "bombCap", 1)
			bombs := s.Bombs
			if cap >= 0 {
				bombs = min(bombs, cap)
			}
			if n >= 2 {
				amount = n * s.Config.Base * (1 << bombs)
			}
			pay(s, 1-p, p, amount, "剩牌与炸弹", false)
			s.Wins = append(s.Wins, Win{Seat: p, From: 1 - p, Points: amount, Reasons: []string{"先出完"}, Hand: []int{}})
			finish(s, "跑得快结算")
		} else {
			s.Turn = 1 - p
		}
	}
	if s.Phase == "guan" && s.Seats[0].Missing >= 0 && s.Seats[1].Missing >= 0 {
		for _, winner := range []int{s.Dealer, 1 - s.Dealer} {
			if s.Seats[winner].Missing == 1 {
				pay(s, 1-winner, winner, 20*s.Config.Base, "关牌", false)
				s.Wins = append(s.Wins, Win{Seat: winner, From: 1 - winner, Points: 20 * s.Config.Base, Reasons: []string{"四5/四A关牌"}, Hand: append([]int{}, s.Seats[winner].Hand...)})
				finish(s, "关牌结算")
				return
			}
		}
		s.Phase = "play"
	}
}
