package game

import (
	"encoding/json"
	"testing"
)

func TestAllGamesTerminateAndConserveScore(t *testing.T) {
	for _, d := range Definitions {
		t.Run(d.Key, func(t *testing.T) {
			for trial := 0; trial < 12; trial++ {
				s, e := New(Config{Game: d.Key}, -1)
				if e != nil {
					t.Fatal(e)
				}
				for step := 0; s.End == ""; step++ {
					if step > 2000 {
						t.Fatalf("stuck: phase=%s turn=%d", s.Phase, s.Turn)
					}
					acted := false
					for p := range s.Seats {
						before, _ := json.Marshal(s)
						actions := Legal(s, p)
						after, _ := json.Marshal(s)
						if string(before) != string(after) {
							t.Fatal("Legal changed state")
						}
						if len(actions) == 0 {
							continue
						}
						a := Auto(s, p)
						if e := Apply(s, p, a); e != nil {
							t.Fatalf("%s: %v", a.Kind, e)
						}
						total := 0
						for _, seat := range s.Seats {
							total += seat.Score
						}
						if total != 0 {
							t.Fatal("non-zero-sum")
						}
						acted = true
						break
					}
					if !acted {
						t.Fatalf("no legal action phase=%s turn=%d offer=%+v", s.Phase, s.Turn, s.Offer)
					}
				}
			}
		})
	}
}

func TestRejectIllegalAndNormalize(t *testing.T) {
	for _, d := range Definitions {
		s, e := New(Config{Game: d.Key}, 0)
		if e != nil {
			t.Fatal(e)
		}
		if len(s.Seats) != d.Players {
			t.Fatal(d.Key)
		}
		b, _ := json.Marshal(s)
		if Apply(s, 0, Action{Kind: "fabricated"}) == nil {
			t.Fatal("illegal accepted")
		}
		a, _ := json.Marshal(s)
		if string(a) != string(b) {
			t.Fatal("illegal mutated")
		}
	}
	if _, e := Normalize(Config{Game: "runfast", Options: map[string]int{"random": 9}}); e == nil {
		t.Fatal("invalid option")
	}
}
