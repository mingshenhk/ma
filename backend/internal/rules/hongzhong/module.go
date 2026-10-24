package hongzhong

import "haojiayuan/internal/rules/mahjong"

const OriginalID = 150
const Status = "customer-approved two-player Duangouka policy"

func Policy() mahjong.Policy {
	return mahjong.Policy{OriginalID: OriginalID, Players: 2, Suits: 2,
		InitialHand: 10, PairCount: 5, EndAfterWins: 1, Wild: true,
		AskPairs: true, ExtraFan: true, Base: mahjong.DuangoukaBase}
}

func Analyze(cards []int) (mahjong.Result, error) { return mahjong.Analyze(cards, true) }
