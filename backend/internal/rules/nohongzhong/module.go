package nohongzhong

import "haojiayuan/internal/rules/mahjong"

const OriginalID = 114
const Status = "customer-approved two-player Duangouka policy without wildcards"

func Policy() mahjong.Policy {
	return mahjong.Policy{OriginalID: OriginalID, Players: 2, Suits: 2,
		InitialHand: 10, PairCount: 5, EndAfterWins: 1,
		AskPairs: true, ExtraFan: true, Base: mahjong.DuangoukaBase}
}

func Analyze(cards []int) (mahjong.Result, error) { return mahjong.Analyze(cards, false) }
