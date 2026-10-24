// Package capture contains transport types shared by two different rule engines.
package capture

type Meld struct {
	Kind  string
	Cards []int
}
type Context struct {
	Hand                                                         []int
	Melds                                                        []Meld
	Options                                                      map[string]int
	Base, Report, Float, OtherFloat                              int
	Heavenly, Earthly, CatchReport, SelfDraw, KongEvent, Preview bool
}

func (c Context) Opt(key string, fallback int) int {
	if v, ok := c.Options[key]; ok {
		return v
	}
	return fallback
}

type Value struct {
	OK           bool
	R, F, Amount int
	Reasons      []string
	ColorOnly    bool
}
