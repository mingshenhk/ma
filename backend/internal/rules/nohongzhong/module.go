package nohongzhong

import "haojiayuan/internal/rules/mahjong"

const OriginalID = 57
const Status = "shape-only; client PLAYER_COUNT=3; name mapping unverified"

func Analyze(cards []int) (mahjong.Result, error) { return mahjong.Analyze(cards, false) }
