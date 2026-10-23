package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"haojiayuan/internal/game"
)

func request(t *testing.T, c *http.Client, method, url string, body any, want int) map[string]any {
	t.Helper()
	b, _ := json.Marshal(body)
	r, e := http.NewRequest(method, url, bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	r.Header.Set("Content-Type", "application/json")
	resp, e := c.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	defer resp.Body.Close()
	v := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&v)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: %d want %d: %+v", method, url, resp.StatusCode, want, v)
	}
	return v
}
func client() *http.Client {
	j, _ := cookiejar.New(nil)
	return &http.Client{Jar: j, Timeout: 5 * time.Second}
}
func number(m map[string]any, k string) int { return int(m[k].(float64)) }

func TestSessionCookieTransport(t *testing.T) {
	for _, tc := range []struct {
		name               string
		force, tls, secure bool
	}{{"local-http", false, false, false}, {"proxy-https", true, false, true}, {"direct-tls", false, true, true}} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := New(Options{Repository: JSONRepository{Dir: t.TempDir()}, CookieSecure: tc.force})
			if err != nil {
				t.Fatal(err)
			}
			h := s.Handler()
			var session *http.Cookie
			for _, method := range []string{"POST", "DELETE"} {
				r := httptest.NewRequest(method, "http://example.test/api/session", strings.NewReader(`{"name":"transport-test"}`))
				if tc.tls {
					r.TLS = &tls.ConnectionState{}
				}
				if session != nil {
					r.AddCookie(session)
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != http.StatusOK {
					t.Fatalf("%s: status %d", method, w.Code)
				}
				cookies := w.Result().Cookies()
				if len(cookies) != 1 {
					t.Fatalf("%s: missing cookie", method)
				}
				session = cookies[0]
				if session.Secure != tc.secure || !session.HttpOnly || session.SameSite != http.SameSiteLaxMode {
					t.Fatalf("%s: wrong transport attributes", method)
				}
				if method == "DELETE" && session.MaxAge >= 0 {
					t.Fatal("logout must expire cookie")
				}
			}
		})
	}
}

func TestHTTPWebsocketPrivacyReplayRestart(t *testing.T) {
	repo := JSONRepository{Dir: t.TempDir()}
	s, e := New(Options{Repository: repo})
	if e != nil {
		t.Fatal(e)
	}
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	a, b, outsider := client(), client(), client()
	auth := request(t, a, "POST", h.URL+"/api/session", map[string]any{"name": "甲"}, 200)
	request(t, b, "POST", h.URL+"/api/session", map[string]any{"name": "乙"}, 200)
	request(t, outsider, "POST", h.URL+"/api/session", map[string]any{"name": "外部"}, 200)
	create := map[string]any{"op": "create", "requestId": "create-1", "config": map[string]any{"game": "hongzhong"}}
	r := request(t, a, "POST", h.URL+"/api/command", create, 200)
	// API command replies wrap the room in a named field.
	if wrapped, ok := r["room"].(map[string]any); ok {
		r = wrapped
	}
	rid := r["id"].(string)
	cmd := func(c *http.Client, op, req string, version, want int) map[string]any {
		v := request(t, c, "POST", h.URL+"/api/command", map[string]any{"op": op, "roomId": rid, "requestId": req, "version": version}, want)
		if w, ok := v["room"].(map[string]any); ok {
			return w
		}
		return v
	}
	replayed := request(t, a, "POST", h.URL+"/api/command", create, 200)
	if w, ok := replayed["room"].(map[string]any); ok {
		replayed = w
	}
	if replayed["id"] != rid || number(replayed, "version") != 1 {
		t.Fatal("create replay")
	}
	request(t, outsider, "GET", h.URL+"/api/room?id="+rid, nil, 403)
	r = cmd(b, "join", "join", 0, 200)
	cmd(b, "ready", "stale", 1, 409)
	r = cmd(a, "ready", "ready-a", number(r, "version"), 200)
	r = cmd(b, "ready", "ready-b", number(r, "version"), 200)
	r = cmd(a, "start", "start", number(r, "version"), 200)
	version := number(r, "version")
	r = cmd(a, "start", "start", version-1, 200)
	if number(r, "version") != version {
		t.Fatal("start replay mutated version")
	}
	state := r["state"].(map[string]any)
	// The browser renders these piles as arrays even before the first discard.
	piles, ok := state["discards"].([]any)
	if !ok || len(piles) != 2 {
		t.Fatal("discard piles must be arrays for every seat")
	}
	for _, pile := range piles {
		if cards, ok := pile.([]any); !ok || len(cards) != 0 {
			t.Fatal("empty discard pile must serialize as [], not null")
		}
	}
	if _, ok := state["table"].([]any); !ok {
		t.Fatal("empty table must serialize as an array")
	}
	seats := state["seats"].([]any)
	expected := 10
	if number(state, "dealer") == 0 {
		expected++
	}
	if len(seats[0].(map[string]any)["hand"].([]any)) != expected {
		t.Fatal("own hand missing")
	}
	if hand, ok := seats[1].(map[string]any)["hand"].([]any); ok && len(hand) != 0 {
		t.Fatal("opponent hand exposed")
	}
	if _, ok := state["wall"]; ok {
		t.Fatal("wall exposed")
	}
	head := http.Header{}
	baseReq, _ := http.NewRequest("GET", h.URL, nil)
	for _, c := range a.Jar.Cookies(baseReq.URL) {
		head.Add("Cookie", c.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(h.URL, "http")+"/ws?room="+rid, &websocket.DialOptions{HTTPHeader: head})
	if e != nil {
		t.Fatal(e)
	}
	defer ws.CloseNow()
	_, snapshot, e := ws.Read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(snapshot, []byte(`"type":"room"`)) {
		t.Fatal("no reconnect snapshot")
	}
	badHeader := head.Clone()
	badHeader.Set("Origin", "https://outside.invalid")
	other, resp, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(h.URL, "http")+"/ws?room="+rid, &websocket.DialOptions{HTTPHeader: badHeader})
	if other != nil {
		other.CloseNow()
	}
	if e == nil || resp == nil || resp.StatusCode != 403 {
		t.Fatal("cross-origin socket accepted")
	}
	// Actual disk restart preserves the random deal, turn, session and replay IDs.
	restored, e := New(Options{Repository: repo})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(mustJSON(s.rooms[rid].State), mustJSON(restored.rooms[rid].State)) || restored.rooms[rid].Version != version {
		t.Fatal("restart changed state")
	}
	if len(auth["recoveryCode"].(string)) < 20 {
		t.Fatal("weak recovery code")
	}
	// Revoked sessions must stop receiving room broadcasts.
	request(t, a, "DELETE", h.URL+"/api/session", nil, 200)
	cmd(b, "auto", "toggle-auto", version, 200)
	if _, _, e := ws.Read(ctx); e == nil {
		t.Fatal("revoked websocket received private snapshot")
	}
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

type failingRepo struct {
	JSONRepository
	fail bool
}

func (r *failingRepo) SaveRoom(v *Room) error {
	if r.fail {
		return errors.New("simulated disk full")
	}
	return r.JSONRepository.SaveRoom(v)
}
func TestSaveFailureRollsBackAndLeavingRevokesReplay(t *testing.T) {
	repo := &failingRepo{JSONRepository: JSONRepository{Dir: t.TempDir()}}
	s, e := New(Options{Repository: repo})
	if e != nil {
		t.Fatal(e)
	}
	u := &User{ID: "u", Name: "tester"}
	v, e := s.command(u, Command{Op: "create", RequestID: "create", Config: game.Config{Game: "runfast"}})
	if e != nil {
		t.Fatal(e)
	}
	rid := v.(map[string]any)["id"].(string)
	before := mustJSON(s.rooms[rid])
	repo.fail = true
	if _, e = s.command(u, Command{Op: "ready", RoomID: rid, RequestID: "ready", Version: 1}); e == nil {
		t.Fatal("disk error hidden")
	}
	if !bytes.Equal(before, mustJSON(s.rooms[rid])) {
		t.Fatal("memory advanced despite save failure")
	}
	repo.fail = false
	if _, e = s.command(u, Command{Op: "leave", RoomID: rid, RequestID: "leave", Version: 1}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.command(u, Command{Op: "create", RequestID: "create", Config: game.Config{Game: "runfast"}}); e == nil {
		t.Fatal("departed member may replay room snapshot")
	}
}
func TestAllGamesRoomLifecycle(t *testing.T) {
	for _, d := range game.Definitions {
		t.Run(d.Key, func(t *testing.T) {
			s, e := New(Options{Repository: JSONRepository{Dir: t.TempDir()}, ActionTimeout: time.Second, BotDelay: time.Millisecond})
			if e != nil {
				t.Fatal(e)
			}
			u := &User{ID: "owner", Name: "玩家"}
			v, e := s.command(u, Command{Op: "create", RequestID: "create", Config: game.Config{Game: d.Key}})
			if e != nil {
				t.Fatal(e)
			}
			rid := v.(map[string]any)["id"].(string)
			apply := func(op string) {
				t.Helper()
				_, err := s.command(u, Command{Op: op, RoomID: rid, RequestID: id(8), Version: s.rooms[rid].Version})
				if err != nil {
					t.Fatal(err)
				}
			}
			for i := 1; i < d.Players; i++ {
				apply("bot")
			}
			apply("ready")
			apply("start")
			apply("auto")
			clock := time.Now().Add(time.Hour)
			for step := 0; s.rooms[rid].Status == "playing"; step++ {
				if step > 2000 {
					t.Fatal("room stuck")
				}
				s.tick(clock.Add(time.Duration(step) * time.Second))
			}
			r := s.rooms[rid]
			if r.Status != "roundEnd" || len(r.History) != 1 {
				t.Fatalf("completion not persisted: %+v", r)
			}
			total := 0
			for _, p := range r.Players {
				total += p.Total
			}
			if total != 0 {
				t.Fatal("room scores not zero-sum")
			}
			apply("dissolve")
			if r.Status != "closed" {
				t.Fatal("owner plus bots should dissolve")
			}
		})
	}
}

// Opt-in bounded connection fan-out benchmark, not a unique-player or WAN test.
func TestConnectionFanout(t *testing.T) {
	if os.Getenv("HJY_LOAD") != "1" {
		t.Skip("set HJY_LOAD=1")
	}
	for _, n := range []int{100, 300, 500} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s, e := New(Options{Repository: JSONRepository{Dir: t.TempDir()}})
			if e != nil {
				t.Fatal(e)
			}
			s.accounts.Users["load"] = &User{ID: "load", Name: "load"}
			s.accounts.Sessions[hash("load-token")] = Session{UserID: "load", Expires: time.Now().Add(time.Hour).Unix()}
			v, e := s.command(s.accounts.Users["load"], Command{Op: "create", RequestID: "create", Config: game.Config{Game: "sichuan"}})
			if e != nil {
				t.Fatal(e)
			}
			rid := v.(map[string]any)["id"].(string)
			h := httptest.NewServer(s.Handler())
			defer h.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			connections := make([]*websocket.Conn, 0, n)
			defer func() {
				for _, c := range connections {
					c.CloseNow()
				}
			}()
			ready := make(chan error, n)
			received := make(chan error, n*2)
			began := time.Now()
			for i := 0; i < n; i++ {
				c, _, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(h.URL, "http")+"/ws?room="+rid, &websocket.DialOptions{HTTPHeader: http.Header{"Cookie": []string{"hjy_session=load-token"}}})
				if e != nil {
					t.Fatal(e)
				}
				connections = append(connections, c)
				go func() {
					_, _, e := c.Read(ctx)
					ready <- e
					for {
						_, _, e := c.Read(ctx)
						received <- e
						if e != nil {
							return
						}
					}
				}()
			}
			for i := 0; i < n; i++ {
				if e := <-ready; e != nil {
					t.Fatal(e)
				}
			}
			connectTime := time.Since(began)
			var latencies []float64
			for j := 0; j < 10; j++ {
				start := time.Now()
				s.mu.Lock()
				r := s.rooms[rid]
				r.Version++
				e := s.repo.SaveRoom(r)
				if e == nil {
					s.broadcast(r)
				}
				s.mu.Unlock()
				if e != nil {
					t.Fatal(e)
				}
				for i := 0; i < n; i++ {
					select {
					case e := <-received:
						if e != nil {
							t.Fatal(e)
						}
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				}
				latencies = append(latencies, float64(time.Since(start).Microseconds())/1000)
			}
			sort.Float64s(latencies)
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			t.Logf("connections=%d snapshots=%d errors=0 setup_ms=%d fanout_p50_ms=%.2f fanout_p90_ms=%.2f heap_mib=%.2f goroutines=%d", n, n*11, connectTime.Milliseconds(), latencies[4], latencies[8], float64(m.HeapAlloc)/1048576, runtime.NumGoroutine())
		})
	}
}
