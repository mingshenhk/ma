// Package sichuan defines the provisional four-player, no-wildcard blood-battle
// variant. It does not inherit Duangouka declarations or its extra-fan table.
package sichuan

import "haojiayuan/internal/rules/mahjong"

const OriginalID = 0 // No authoritative original server ID has been established.

func Policy() mahjong.Policy {
	return mahjong.Policy{OriginalID: OriginalID, Players: 4, Suits: 3,
		InitialHand: 13, PairCount: 7, EndAfterWins: 3, MissingSuit: true, Base: base}
}

func base(v mahjong.Features) (int, string) {
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
		set(3, "七对")
	}
	if v.Clear && v.Pungs && !v.Pairs {
		set(4, "清对对")
	}
	if v.Clear && v.Pairs {
		set(5, "清七对")
	}
	if v.Jiang && v.Pungs && !v.Pairs {
		set(4, "将对")
	}
	return f, name
}
