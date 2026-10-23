package game

import "testing"

func fixtureMJ(key string, n int) *State {
	s := &State{Config: Config{Game: key, Base: 1, Options: map[string]int{}}, Seats: make([]Seat, n), Wall: []int{0x22}, Discards: make([][]int, n), Dealer: 0, Turn: 0, Phase: "discard", KongSeat: -1, PublicActions: 2, LastSource: "self"}
	for i := range s.Seats {
		s.Seats[i] = Seat{Missing: -1, Passed: -1, Draws: 2, Discards: 2}
	}
	return s
}
func hasReason(v mjValue, reason string) bool {
	for _, r := range v.Reasons {
		if r == reason {
			return true
		}
	}
	return false
}

func TestTwoPlayerFirstWinnerAndActualLastTile(t *testing.T) {
	for _, key := range []string{"hongzhong", "nohongzhong"} {
		s := fixtureMJ(key, 2)
		s.Seats[0].Hand = []int{0x11, 0x12, 0x13, 0x21, 0x22, 0x23, 0x26, 0x27, 0x28, 0x19, 0x19}
		v := mjScore(s, 0, 0, "self", false)
		if !v.OK || v.Fan != 1 {
			t.Fatalf("%s ordinary: %+v", key, v)
		}
		s.Wall = nil
		last := mjScore(s, 0, 0, "self", false)
		if !last.OK || last.Fan != v.Fan+1 {
			t.Fatalf("last drawable tile: %+v", last)
		}
		s.Wall = []int{0x22}
		if err := Apply(s, 0, Action{Kind: "hu"}); err != nil {
			t.Fatal(err)
		}
		if s.End == "" || len(s.Wins) != 1 || !s.Seats[0].Won || s.Seats[0].Score != 3 || s.Seats[1].Score != -3 {
			t.Fatalf("first win must end and self draw adds 2B: %+v", s)
		}
	}
}
func TestRootUsesPhysicalTilesAndWildcardDoesNotCreateRoot(t *testing.T) {
	s := fixtureMJ("hongzhong", 2)
	s.Seats[0].Hand = []int{0x12, 0x12, 0x12, 0x12, 0x13, 0x14, 0x21, 0x22, 0x23, 0x19, 0x19}
	v := mjScore(s, 0, 0, "self", false)
	if !v.OK || v.Fan != 2 {
		t.Fatalf("physical root %+v", v)
	}
	s.Seats[0].Hand[3] = red
	v = mjScore(s, 0, 0, "self", false)
	if !v.OK || v.Fan != 1 {
		t.Fatalf("wildcard root %+v", v)
	}
	s.Seats[0].Hand = s.Seats[0].Hand[:10]
	if mjScore(s, 0, red, "discard", false).OK {
		t.Fatal("discarded red may not win")
	}
}
func TestChickenCannotWinOrdinaryDiscardButCanGrabReportAsk(t *testing.T) {
	s := fixtureMJ("hongzhong", 2)
	s.Seats[1].Hand = []int{0x11, 0x12, 0x13, 0x21, 0x22, 0x23, 0x26, 0x27, 0x28, 0x19}
	if mjScore(s, 1, 0x19, "discard", false).OK {
		t.Fatal("one fan chicken discard accepted")
	}
	s.Seats[0].Report = 2
	s.Offer = &Offer{Card: 0x19, From: 0, Source: "ask", Next: 0, Eligible: []int{1}, Replies: map[int]Action{}}
	v := mjScore(s, 1, 0x19, "ask", false)
	if !v.OK || v.Fan != 5 {
		t.Fatalf("grab report ask should add four: %+v", v)
	}
}
func TestSichuanMultiWinAndFiveWanIsNotRed(t *testing.T) {
	s := fixtureMJ("sichuan", 4)
	for _, p := range []int{1, 2} {
		s.Seats[p].Missing = 3
		s.Seats[p].Hand = []int{0x11, 0x12, 0x13, 0x21, 0x22, 0x23, 0x26, 0x27, 0x28, 0x16, 0x17, 0x18, 0x19}
	}
	s.Seats[3].Won = true
	s.Wins = []Win{{Seat: 3}}
	s.Offer = &Offer{Card: 0x19, From: 0, Source: "discard", Next: 1, Eligible: []int{1, 2}, Replies: map[int]Action{}}
	s.Phase = "respond"
	if e := Apply(s, 1, Action{Kind: "hu"}); e != nil {
		t.Fatal(e)
	}
	if s.End != "" {
		t.Fatal("resolved before all simultaneous winners replied")
	}
	if e := Apply(s, 2, Action{Kind: "hu"}); e != nil {
		t.Fatal(e)
	}
	if s.End == "" || len(s.Wins) != 3 || s.Seats[0].Score != -2 || s.Seats[1].Score != 1 || s.Seats[2].Score != 1 {
		t.Fatalf("multiwin %+v", s.Wins)
	}
	s = fixtureMJ("sichuan", 4)
	s.Seats[1].Missing = 2
	s.Seats[1].Hand = []int{0x11, 0x12, 0x13, 0x16, 0x17, 0x18, 0x31, 0x32, 0x33, 0x36, 0x37, 0x38, 0x35}
	if !mjScore(s, 1, 0x35, "discard", false).OK {
		t.Fatal("five wan incorrectly treated as red")
	}
	s.Seats[1].Missing = 3
	if mjScore(s, 1, 0x35, "discard", false).OK {
		t.Fatal("missing suit present")
	}
}
func TestRunfastBlackFiveAndGuanPriority(t *testing.T) {
	for _, random := range []int{0, 1} {
		s, e := New(Config{Game: "runfast", Options: map[string]int{"random": random}}, -1)
		if e != nil {
			t.Fatal(e)
		}
		if len(s.Seats[0].Hand) != 10 || len(s.Seats[1].Hand) != 10 || count(s.Seats[s.Dealer].Hand, 0x35) != 1 {
			t.Fatal("deal/first player")
		}
		if len(s.Wall) != 3+17*random {
			t.Fatal("sealed cards")
		}
	}
	s := fixtureMJ("runfast", 2)
	s.Phase = "guan"
	s.Dealer = 1
	s.Seats[0].Hand = []int{5, 0x15, 0x25, 0x35, 1, 0x11, 0x21, 0x31, 9, 10}
	s.Seats[1].Hand = append([]int{}, s.Seats[0].Hand...)
	if e := Apply(s, 0, Action{Kind: "guan", Cards: []int{5, 0x15, 0x25, 0x35}}); e != nil {
		t.Fatal(e)
	}
	if s.End != "" {
		t.Fatal("private declarations resolved too early")
	}
	if e := Apply(s, 1, Action{Kind: "guan", Cards: []int{5, 0x15, 0x25, 0x35}}); e != nil {
		t.Fatal(e)
	}
	if s.End == "" || s.Seats[1].Score != 20 || s.Seats[0].Score != -20 {
		t.Fatal("dealer's guan must take priority")
	}
}
