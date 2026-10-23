// Package doushisi models the 21 evidenced tile kinds, not an invented full deck.
package doushisi

import "errors"

type Tile struct {
	Code int    `json:"code"`
	Name string `json:"name"`
	Pips int    `json:"pips"`
	Red  int    `json:"red"`
}

var Tiles = []Tile{{0x1111, "地牌", 2, 4}, {0x1102, "丁丁", 3, 0}, {0x0202, "长二", 4, 0}, {0x1103, "和牌", 4, 1}, {0x0203, "拐子", 5, 0}, {0x1114, "幺四", 5, 10}, {0x0303, "长三", 6, 0}, {0x1105, "珠珠", 6, 1}, {0x0214, "二红", 6, 4}, {0x0205, "二五", 7, 0}, {0x1106, "幺六", 7, 1}, {0x0314, "三四", 7, 4}, {0x0206, "二六", 8, 0}, {0x0305, "三五", 8, 0}, {0x1414, "人牌", 8, 16}, {0x0306, "黑九", 9, 0}, {0x1405, "红九", 9, 4}, {0x0505, "梅子", 10, 0}, {0x1406, "四六", 10, 4}, {0x0506, "斧头", 11, 0}, {0x1606, "天牌", 12, 6}}

func Find(code int) (Tile, bool) {
	for _, t := range Tiles {
		if t.Code == code {
			return t, true
		}
	}
	return Tile{}, false
}

type Result struct {
	Pairs14   bool   `json:"pairs14"`
	RedPoints int    `json:"redPoints"`
	Warning   string `json:"warning"`
}

func Analyze(cards []int) (Result, error) {
	r := Result{Warning: "仅分析余手14点配对及牌面红点，非完整胡牌/计分"}
	if len(cards) == 0 || len(cards) > 12 || len(cards)%2 != 0 {
		return r, errors.New("余手须为2..12张偶数")
	}
	var n [13]int
	for _, c := range cards {
		t, ok := Find(c)
		if !ok {
			return r, errors.New("未知骨牌牌码")
		}
		n[t.Pips]++
		r.RedPoints += t.Red
	}
	r.Pairs14 = true
	for i := 2; i <= 12; i++ {
		if i == 7 {
			r.Pairs14 = r.Pairs14 && n[i]%2 == 0
		} else {
			r.Pairs14 = r.Pairs14 && n[i] == n[14-i]
		}
	}
	return r, nil
}
