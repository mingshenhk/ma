package game

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
)

type Config struct {
	Game    string         `json:"game"`
	Base    int            `json:"base"`
	Rounds  int            `json:"rounds"`
	Options map[string]int `json:"options"`
}
type Definition struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Players     int    `json:"players"`
	ID          int    `json:"id"`
	Description string `json:"description"`
}

var Definitions = []Definition{
	{"hongzhong", "红中断勾卡", 2, 150, "76张 · 红中癞子 · 三坎一对"},
	{"nohongzhong", "二人断勾卡", 2, 114, "72张 · 无红中 · 根与卡"},
	{"hongershi", "红二十", 2, 110, "凑十四 · 红点20起胡"},
	{"runfast", "二人跑得快", 2, 115, "10张手牌 · 有牌必压"},
	{"doushisi", "斗十四", 2, 95, "84张骨牌 · 红点30起胡"},
	{"sichuan", "四人普通麻将", 4, 0, "108张 · 定缺 · 无红中血战"},
}

func Def(key string) (Definition, bool) {
	for _, d := range Definitions {
		if d.Key == key {
			return d, true
		}
	}
	return Definition{}, false
}
func Normalize(c Config) (Config, error) {
	d, ok := Def(c.Game)
	if !ok {
		return c, errors.New("未知游戏")
	}
	_ = d
	if c.Base == 0 {
		c.Base = 1
	}
	if c.Base != 1 && c.Base != 2 && c.Base != 5 && c.Base != 10 {
		return c, errors.New("底分必须是1/2/5/10")
	}
	if c.Rounds == 0 {
		c.Rounds = 8
		if c.Game == "hongershi" {
			c.Rounds = 4
		}
		if c.Game == "runfast" {
			c.Rounds = 10
		}
	}
	valid := map[string][]int{"runfast": {10, 20}, "hongershi": {4, 8, 12}, "doushisi": {8, 12, 16}}
	a := valid[c.Game]
	if a == nil {
		a = []int{8, 16, 32}
	}
	found := false
	for _, v := range a {
		found = found || c.Rounds == v
	}
	if !found {
		return c, errors.New("局数选项不支持")
	}
	if c.Options == nil {
		c.Options = map[string]int{}
	}
	return c, validateOptions(c)
}
func Opt(s *State, k string, def int) int {
	v, ok := s.Config.Options[k]
	if ok {
		return v
	}
	return def
}
func validateOptions(c Config) error {
	allowed := map[string][]int{}
	switch c.Game {
	case "hongzhong", "nohongzhong":
		allowed = map[string][]int{"root": {0, 1}, "ka": {0, 1}, "minFan": {1, 2, 3}}
	case "sichuan":
		allowed = map[string][]int{"root": {0, 1}}
	case "hongershi":
		allowed = map[string][]int{"kings": {6, 9, 12}, "redFive": {0, 1}, "blackFive": {0, 1}, "selfEat": {0, 1}, "cap": {4, 5, 6}, "ladder": {0, 1}, "checkListen": {0, 1}, "listenMode": {0, 1}}
	case "doushisi":
		allowed = map[string][]int{"allowFloat": {0, 1}, "cap": {500, 1000}, "ladder": {0, 1}}
	case "runfast":
		allowed = map[string][]int{"random": {0, 1}, "guan": {0, 1}, "bombCap": {-1, 1, 3}}
	}
	for k, v := range c.Options {
		values, ok := allowed[k]
		if !ok {
			return fmt.Errorf("不支持的规则选项：%s", k)
		}
		found := false
		for _, n := range values {
			found = found || n == v
		}
		if !found {
			return fmt.Errorf("规则选项值无效：%s", k)
		}
	}
	return nil
}

type Meld struct {
	Kind   string `json:"kind"`
	Cards  []int  `json:"cards"`
	From   int    `json:"from"`
	Hidden bool   `json:"hidden,omitempty"`
}
type Seat struct {
	Hand        []int  `json:"hand"`
	Melds       []Meld `json:"melds"`
	Score       int    `json:"score"`
	Won         bool   `json:"won"`
	Report      int    `json:"report"`
	ReportKongs int    `json:"reportKongs"`
	Missing     int    `json:"missing"`
	Draws       int    `json:"draws"`
	Discards    int    `json:"discards"`
	Passed      int    `json:"passed"`
	Float       int    `json:"float"`
}
type Action struct {
	Kind  string `json:"kind"`
	Cards []int  `json:"cards,omitempty"`
	Value int    `json:"value,omitempty"`
	Label string `json:"label"`
}
type Offer struct {
	Card     int            `json:"card"`
	From     int            `json:"from"`
	Source   string         `json:"source"`
	Next     int            `json:"next"`
	Replies  map[int]Action `json:"replies"`
	Eligible []int          `json:"eligible"`
}
type Transfer struct {
	From   int    `json:"from"`
	To     int    `json:"to"`
	Amount int    `json:"amount"`
	Reason string `json:"reason"`
	Gang   bool   `json:"gang,omitempty"`
}
type Win struct {
	Seat    int      `json:"seat"`
	From    int      `json:"from"`
	Fan     int      `json:"fan"`
	Points  int      `json:"points"`
	Reasons []string `json:"reasons"`
	Hand    []int    `json:"hand"`
}
type Event struct {
	Seat  int    `json:"seat"`
	Kind  string `json:"kind"`
	Cards []int  `json:"cards,omitempty"`
	Text  string `json:"text"`
}
type State struct {
	Config        Config     `json:"config"`
	Seats         []Seat     `json:"seats"`
	Wall          []int      `json:"wall"`
	Turn          int        `json:"turn"`
	Dealer        int        `json:"dealer"`
	Phase         string     `json:"phase"`
	Offer         *Offer     `json:"offer,omitempty"`
	Discards      [][]int    `json:"discards"`
	Table         []int      `json:"table,omitempty"`
	TableSeat     int        `json:"tableSeat"`
	First         bool       `json:"first"`
	LastCard      int        `json:"lastCard"`
	LastSource    string     `json:"lastSource"`
	KongSeat      int        `json:"kongSeat"`
	PublicActions int        `json:"publicActions"`
	Bombs         int        `json:"bombs"`
	Ledger        []Transfer `json:"ledger"`
	Wins          []Win      `json:"wins"`
	Events        []Event    `json:"events"`
	End           string     `json:"end,omitempty"`
	PendingPhase  string     `json:"pendingPhase,omitempty"`
}

func Shuffle(a []int) error {
	if len(a) < 2 {
		return nil
	}
	// Batch secure random bytes so dealing does not make one crypto/rand.Int
	// request and big.Int allocation for every card. Rejection sampling keeps
	// Fisher-Yates unbiased while retaining crypto/rand's CSPRNG.
	const batchSize = 4096
	var pool [batchSize]byte
	poolLen := len(a) - 1
	if poolLen > batchSize/8 {
		poolLen = batchSize / 8
	}
	poolLen *= 8
	buf := pool[:poolLen]
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	position := 0
	for i := len(a) - 1; i > 0; i-- {
		bound := uint64(i) + 1
		threshold := -bound % bound
		var value uint64
		for {
			if position+8 > len(buf) {
				if _, err := rand.Read(buf); err != nil {
					return err
				}
				position = 0
			}
			value = binary.LittleEndian.Uint64(buf[position : position+8])
			position += 8
			if value >= threshold {
				break
			}
		}
		j := int(value % bound)
		a[i], a[j] = a[j], a[i]
	}
	return nil
}
func Random(n int) (int, error) {
	return randomFrom(rand.Reader, n)
}
func randomFrom(source io.Reader, n int) (int, error) {
	if n <= 0 {
		return 0, errors.New("random bound must be positive")
	}
	v, e := rand.Int(source, big.NewInt(int64(n)))
	if e != nil {
		return 0, e
	}
	return int(v.Int64()), nil
}
func New(c Config, dealer int) (*State, error) {
	c, e := Normalize(c)
	if e != nil {
		return nil, e
	}
	d, _ := Def(c.Game)
	if dealer < 0 || dealer >= d.Players {
		dealer, e = Random(d.Players)
		if e != nil {
			return nil, fmt.Errorf("choose dealer: %w", e)
		}
	}
	s := &State{Config: c, Seats: make([]Seat, d.Players), Discards: make([][]int, d.Players), Dealer: dealer, Turn: dealer, First: true, TableSeat: -1, KongSeat: -1, Phase: "discard"}
	for i := range s.Seats {
		s.Seats[i].Passed = -1
		s.Seats[i].Missing = -1
	}
	switch c.Game {
	case "runfast":
		e = startRF(s)
	case "hongershi", "doushisi":
		e = startCapture(s)
	default:
		e = startMJ(s)
	}
	return s, e
}
func Clone(s *State) *State {
	b, _ := json.Marshal(s)
	var out State
	_ = json.Unmarshal(b, &out)
	return &out
}
func Legal(s *State, p int) []Action {
	if s == nil || s.End != "" || p < 0 || p >= len(s.Seats) || s.Seats[p].Won {
		return nil
	}
	switch s.Config.Game {
	case "runfast":
		return legalRF(s, p)
	case "hongershi", "doushisi":
		return legalCapture(s, p)
	default:
		return legalMJ(s, p)
	}
}
func Same(a, b Action) bool {
	if a.Kind != b.Kind || a.Value != b.Value || len(a.Cards) != len(b.Cards) {
		return false
	}
	x := append([]int{}, a.Cards...)
	y := append([]int{}, b.Cards...)
	sort.Ints(x)
	sort.Ints(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
func Apply(s *State, p int, a Action) error {
	allowed := false
	for _, v := range Legal(s, p) {
		if Same(a, v) {
			a = v
			allowed = true
			break
		}
	}
	if !allowed {
		return errors.New("当前状态不允许此操作")
	}
	switch s.Config.Game {
	case "runfast":
		applyRF(s, p, a)
	case "hongershi", "doushisi":
		applyCapture(s, p, a)
	default:
		applyMJ(s, p, a)
	}
	return nil
}
func Auto(s *State, p int) Action {
	a := Legal(s, p)
	for _, v := range a {
		if v.Kind == "hu" {
			return v
		}
	}
	for _, v := range a {
		if v.Kind == "ask" {
			return v
		}
	}
	for _, v := range a {
		if v.Kind == "pass" || v.Kind == "draw" {
			return v
		}
	}
	for _, v := range a {
		if v.Kind == "discard" || v.Kind == "play" || v.Kind == "missing" || v.Kind == "float" {
			return v
		}
	}
	if len(a) > 0 {
		return a[0]
	}
	return Action{}
}
func remove(a []int, cards ...int) []int {
	out := append([]int{}, a...)
	for _, c := range cards {
		for i, v := range out {
			if v == c {
				out = append(out[:i], out[i+1:]...)
				break
			}
		}
	}
	return out
}
func count(a []int, c int) int {
	n := 0
	for _, v := range a {
		if c == v {
			n++
		}
	}
	return n
}
func unique(a []int) []int {
	out := []int{}
	m := map[int]bool{}
	for _, c := range a {
		if !m[c] {
			m[c] = true
			out = append(out, c)
		}
	}
	sort.Ints(out)
	return out
}
func logEvent(s *State, p int, k string, c []int, t string) {
	s.Events = append(s.Events, Event{p, k, append([]int{}, c...), t})
	if len(s.Events) > 200 {
		s.Events = s.Events[len(s.Events)-200:]
	}
}
func pay(s *State, from, to, n int, why string, gang bool) {
	if from == to || n <= 0 {
		return
	}
	s.Seats[from].Score -= n
	s.Seats[to].Score += n
	s.Ledger = append(s.Ledger, Transfer{from, to, n, why, gang})
}
func next(s *State, p int) int {
	for i := 1; i <= len(s.Seats); i++ {
		n := (p + i) % len(s.Seats)
		if !s.Seats[n].Won {
			return n
		}
	}
	return p
}
func take(s *State, tail bool) (int, bool) {
	if len(s.Wall) == 0 {
		return 0, false
	}
	i := 0
	if tail {
		i = len(s.Wall) - 1
	}
	c := s.Wall[i]
	s.Wall = append(s.Wall[:i], s.Wall[i+1:]...)
	return c, true
}
func a(k, l string, c ...int) Action { return Action{Kind: k, Label: l, Cards: c} }
func finish(s *State, why string) {
	s.End = why
	s.Phase = "ended"
	s.Offer = nil
	logEvent(s, -1, "end", nil, why)
}
func fanMultiplier(f int) int    { f = max(1, min(6, f)); return 1 << (f - 1) }
func describeScore(f int) string { return fmt.Sprintf("%d番", f) }
