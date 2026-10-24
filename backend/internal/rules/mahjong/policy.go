package mahjong

// Policy contains variant decisions. Tile decomposition and turn transport can
// be shared, while decks, declarations, win conditions and scoring come from it.
type Policy struct {
	OriginalID, Players, Suits, InitialHand, PairCount, EndAfterWins int
	Wild, MissingSuit, AskPairs, ExtraFan                            bool
	Base                                                             func(Features) (int, string)
}

type Features struct{ Pungs, Clear, Pairs, Terminal, Jiang bool }

// DuangoukaBase is intentionally shared by the two customer-approved variants:
// removing the red wildcard does not change their scoring table.
func DuangoukaBase(v Features) (int, string) {
	f, name := 1, "鸡胡"
	set := func(n int, text string) {
		if n > f {
			f, name = n, text
		}
	}
	if v.Pungs && !v.Pairs {
		set(2, "对对胡")
	}
	if v.Clear {
		set(3, "清一色")
	}
	if v.Pairs {
		set(3, "五对")
	}
	if v.Clear && v.Pungs && !v.Pairs {
		set(4, "清对对")
	}
	if v.Clear && v.Pairs {
		set(5, "清请胡")
	}
	if v.Terminal && !v.Pairs {
		set(5, "带幺九")
		if v.Clear {
			set(6, "清幺九")
		}
	}
	if v.Jiang && v.Pungs && !v.Pairs {
		set(6, "将对")
	}
	return f, name
}
