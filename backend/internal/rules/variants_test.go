package rules_test

import (
	"testing"

	"haojiayuan/internal/rules/capture"
	"haojiayuan/internal/rules/doushisi"
	"haojiayuan/internal/rules/hongershi"
	"haojiayuan/internal/rules/hongzhong"
	"haojiayuan/internal/rules/mahjong"
	"haojiayuan/internal/rules/nohongzhong"
	"haojiayuan/internal/rules/sichuan"
)

// These examples use different physical decks and expected customer-rule totals.
// A poker implementation copied into bone tiles must fail these cases.
func TestCaptureVariantSettlementExamples(t *testing.T) {
	he := capture.Context{Hand: []int{0x11, 0x0d, 0x12, 0x0c, 0x17, 0x27, 0x19, 0x05}, Base: 1}
	ds := capture.Context{Hand: []int{0x1111, 0x1606, 0x1111, 0x1606, 0x0202, 0x1406, 0x0202, 0x1406, 0x0214, 0x1414, 0x0214, 0x1414}, Base: 1}
	for _, tc := range []struct {
		name             string
		ctx              capture.Context
		score            func(capture.Context) capture.Value
		red, fan, amount int
	}{
		{"红二十普通胡", he, hongershi.Score, 26, 0, 2},
		{"红二十自摸", func() capture.Context { c := he; c.SelfDraw = true; return c }(), hongershi.Score, 26, 1, 3},
		{"红二十报胡", func() capture.Context { c := he; c.Report = 1; return c }(), hongershi.Score, 26, 3, 9},
		{"红二十抓报六番封顶", func() capture.Context { c := he; c.CatchReport = true; c.Options = map[string]int{"cap": 6}; return c }(), hongershi.Score, 26, 6, 65},
		{"斗十四按红点支付", ds, doushisi.Score, 68, 0, 68},
		{"斗十四不套红二十抓报自摸杠番", func() capture.Context { c := ds; c.SelfDraw, c.CatchReport, c.KongEvent = true, true, true; return c }(), doushisi.Score, 68, 0, 68},
		{"斗十四报胡", func() capture.Context { c := ds; c.Report = 1; return c }(), doushisi.Score, 68, 1, 136},
		{"斗十四双方漂", func() capture.Context { c := ds; c.Float, c.OtherFloat = 1, 1; return c }(), doushisi.Score, 68, 0, 272},
		{"斗十四天胡", func() capture.Context { c := ds; c.Heavenly = true; return c }(), doushisi.Score, 68, 2, 272},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.score(tc.ctx)
			if !v.OK || v.R != tc.red || v.F != tc.fan || v.Amount != tc.amount {
				t.Fatalf("got %+v; want red=%d fan=%d amount=%d", v, tc.red, tc.fan, tc.amount)
			}
		})
	}
	if !hongershi.PairShape([]int{0x07, 0x17}) || doushisi.PairShape([]int{0x0205, 0x0205}) || !doushisi.PairShape([]int{0x0205, 0x1106}) {
		t.Fatal("poker 7+7 and bone-tile different-kind pairing rules were conflated")
	}
	if hongershi.Score(ds).OK || doushisi.Score(he).OK {
		t.Fatal("a hand from a different game must not be accepted")
	}
	if len(hongershi.Deck(6)) != 58 || len(doushisi.Deck()) != 84 {
		t.Fatal("incorrect independent deck sizes")
	}
}

func TestMahjongPoliciesHaveDifferentDeclarationsAndFanTables(t *testing.T) {
	hz, no, sc := hongzhong.Policy(), nohongzhong.Policy(), sichuan.Policy()
	if hz.OriginalID != 150 || no.OriginalID != 114 || !hz.Wild || no.Wild || hz.Players != 2 || no.EndAfterWins != 1 {
		t.Fatal("incorrect two-player policies")
	}
	if sc.Players != 4 || sc.Suits != 3 || sc.InitialHand != 13 || sc.PairCount != 7 || sc.EndAfterWins != 3 || !sc.MissingSuit || sc.AskPairs || sc.ExtraFan || sc.Wild {
		t.Fatal("four-player policy inherited Duangouka rules")
	}
	for _, tc := range []struct {
		feature mahjong.Features
		dkg, sc int
	}{
		{mahjong.Features{Terminal: true}, 5, 1},
		{mahjong.Features{Jiang: true, Pungs: true}, 6, 4},
	} {
		a, _ := hz.Base(tc.feature)
		b, _ := no.Base(tc.feature)
		c, _ := sc.Base(tc.feature)
		if a != tc.dkg || b != tc.dkg || c != tc.sc {
			t.Fatalf("fan mismatch: %d/%d/%d", a, b, c)
		}
	}
}
