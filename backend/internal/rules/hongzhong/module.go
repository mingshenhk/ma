package hongzhong

import "haojiayuan/internal/rules/mahjong"

const OriginalID = 150
const Status = "shape-only; name mapping and wall count unverified"

func Analyze(cards []int) (mahjong.Result, error) { return mahjong.Analyze(cards, true) }
