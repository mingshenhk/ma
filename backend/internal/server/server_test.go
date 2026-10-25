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
	"net/netip"
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
func register(t *testing.T, c *http.Client, baseURL, username string) map[string]any {
	t.Helper()
	return request(t, c, "POST", baseURL+"/api/register", map[string]any{"username": username, "password": "TestPassword123"}, 200)
}
func number(m map[string]any, k string) int { return int(m[k].(float64)) }

func TestUsernamePasswordRegistrationAndLogin(t *testing.T) {
	s, err := New(Options{Repository: JSONRepository{Dir: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	registered := client()
	response := register(t, registered, h.URL, "SampleUser")
	user := response["user"].(map[string]any)
	if user["username"] != "SampleUser" || user["id"] == "" {
		t.Fatalf("registration returned wrong public identity: %+v", user)
	}
	public, _ := json.Marshal(response)
	for _, secret := range []string{"password", "passwordHash", "usernameKey", "recoveryCode"} {
		if bytes.Contains(public, []byte(secret)) {
			t.Fatalf("registration response leaked %q", secret)
		}
	}
	request(t, client(), "POST", h.URL+"/api/register", map[string]any{"username": "sampleuser", "password": "TestPassword123"}, 409)
	loggedIn := request(t, client(), "POST", h.URL+"/api/login", map[string]any{"username": "sAmPlEuSeR", "password": "TestPassword123"}, 200)
	if loggedIn["user"].(map[string]any)["id"] != user["id"] {
		t.Fatal("login should accept case-insensitive username and return the same account")
	}
	request(t, client(), "POST", h.URL+"/api/login", map[string]any{"username": "SampleUser", "password": "wrong-password"}, 401)
	request(t, client(), "POST", h.URL+"/api/register", map[string]any{"username": "ab", "password": "short"}, 400)
}

func TestSessionCookieTransport(t *testing.T) {
	for _, cfg := range []struct {
		name               string
		force, tls, secure bool
	}{{"local-http", false, false, false}, {"proxy-https", true, false, true}, {"direct-tls", false, true, true}} {
		t.Run(cfg.name, func(t *testing.T) {
			s, err := New(Options{Repository: JSONRepository{Dir: t.TempDir()}, CookieSecure: cfg.force})
			if err != nil {
				t.Fatal(err)
			}
			h := s.Handler()
			var session *http.Cookie
			for _, call := range []struct{ method, path, body string }{
				{"POST", "/api/register", `{"username":"transport-test","password":"TestPassword123"}`},
				{"DELETE", "/api/session", ""},
			} {
				r := httptest.NewRequest(call.method, "http://example.test"+call.path, strings.NewReader(call.body))
				if cfg.tls {
					r.TLS = &tls.ConnectionState{}
				}
				if session != nil {
					r.AddCookie(session)
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != http.StatusOK {
					t.Fatalf("%s: status %d", call.method, w.Code)
				}
				cookies := w.Result().Cookies()
				if len(cookies) != 1 {
					t.Fatalf("%s: missing cookie", call.method)
				}
				session = cookies[0]
				if session.Secure != cfg.secure || !session.HttpOnly || session.SameSite != http.SameSiteLaxMode {
					t.Fatalf("%s: wrong transport attributes", call.method)
				}
				if call.method == "DELETE" && session.MaxAge >= 0 {
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
	auth := register(t, a, h.URL, "甲账号")
	register(t, b, h.URL, "乙账号")
	register(t, outsider, h.URL, "外部账号")
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
	publicAuth, _ := json.Marshal(auth)
	if auth["user"].(map[string]any)["username"] != "甲账号" || bytes.Contains(publicAuth, []byte("passwordHash")) || bytes.Contains(publicAuth, []byte("recoveryCode")) {
		t.Fatal("registration must return the username without credential secrets")
	}
	// Revoked sessions must stop receiving room broadcasts.
	request(t, a, "DELETE", h.URL+"/api/session", nil, 200)
	s.mu.Lock()
	s.broadcast(s.rooms[rid])
	s.mu.Unlock()
	if _, _, e := ws.Read(ctx); e == nil {
		t.Fatal("revoked websocket received private snapshot")
	}
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func TestWebSocketConnectionLimitsReleaseOnDisconnect(t *testing.T) {
	token := "ws-limit-token"
	s, err := New(Options{Repository: JSONRepository{Dir: t.TempDir()}, MaxWSPerUser: 1, MaxWSPerIP: 2, MaxWSGlobal: 2})
	if err != nil {
		t.Fatal(err)
	}
	user := &User{ID: "ws-user", Name: "WS user"}
	s.accounts.Users[user.ID] = user
	s.accounts.Sessions[hash(token)] = Session{UserID: user.ID, Expires: time.Now().Add(time.Hour).Unix()}
	created, err := s.command(user, Command{Op: "create", RequestID: "ws-limit-room", Config: game.Config{Game: "sichuan"}})
	if err != nil {
		t.Fatal(err)
	}
	rid := created.(map[string]any)["id"].(string)
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	url := "ws" + strings.TrimPrefix(h.URL, "http") + "/ws?room=" + rid
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	options := &websocket.DialOptions{HTTPHeader: http.Header{"Cookie": []string{"hjy_session=" + token}}}
	first, _, err := websocket.Dial(ctx, url, options)
	if err != nil {
		t.Fatal(err)
	}
	second, resp, err := websocket.Dial(ctx, url, options)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if second != nil {
		second.CloseNow()
	}
	if err == nil || resp == nil || resp.StatusCode != http.StatusTooManyRequests {
		first.CloseNow()
		t.Fatalf("per-user connection limit was not enforced: response=%v err=%v", resp, err)
	}
	first.CloseNow()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mu.RLock()
		remaining := s.wsGlobal
		s.mu.RUnlock()
		if remaining == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.mu.RLock()
	remaining := s.wsGlobal
	s.mu.RUnlock()
	if remaining != 0 {
		t.Fatal("websocket connection slot was not released after disconnect")
	}
	reconnected, _, err := websocket.Dial(ctx, url, options)
	if err != nil {
		t.Fatalf("connection slot remained stuck after disconnect: %v", err)
	}
	reconnected.CloseNow()
}

func TestClientIPOnlyTrustsConfiguredProxyChain(t *testing.T) {
	proxy, err := netip.ParsePrefix("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{trustedProxies: []netip.Prefix{proxy}}

	spoofed := httptest.NewRequest(http.MethodGet, "/", nil)
	spoofed.RemoteAddr = "198.51.100.20:4567"
	spoofed.Header.Set("X-Forwarded-For", "203.0.113.99")
	if got := s.clientIP(spoofed); got != "198.51.100.20" {
		t.Fatalf("untrusted peer spoofed client IP: got %q", got)
	}

	proxied := httptest.NewRequest(http.MethodGet, "/", nil)
	proxied.RemoteAddr = "10.0.0.2:4567"
	proxied.Header.Add("X-Forwarded-For", "198.51.100.99, 203.0.113.8")
	proxied.Header.Add("X-Forwarded-For", "10.0.0.3")
	if got := s.clientIP(proxied); got != "203.0.113.8" {
		t.Fatalf("trusted proxy chain resolved to %q, want closest untrusted client 203.0.113.8", got)
	}

	malformed := httptest.NewRequest(http.MethodGet, "/", nil)
	malformed.RemoteAddr = "10.0.0.2:4567"
	malformed.Header.Set("X-Forwarded-For", "attacker-controlled")
	if got := s.clientIP(malformed); got != "10.0.0.2" {
		t.Fatalf("malformed forwarded header should fall back to proxy peer, got %q", got)
	}
}

func TestClientIPBoundsForwardedHeaders(t *testing.T) {
	proxy, err := netip.ParsePrefix("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{trustedProxies: []netip.Prefix{proxy}}

	tooLong := httptest.NewRequest(http.MethodGet, "/", nil)
	tooLong.RemoteAddr = "10.0.0.2:4567"
	tooLong.Header.Set("X-Forwarded-For", strings.Repeat("1", 4097))
	if got := s.clientIP(tooLong); got != "10.0.0.2" {
		t.Fatalf("oversized forwarded header should fall back to proxy peer, got %q", got)
	}

	tooManyHops := httptest.NewRequest(http.MethodGet, "/", nil)
	tooManyHops.RemoteAddr = "10.0.0.2:4567"
	tooManyHops.Header.Set("X-Forwarded-For", strings.TrimSuffix(strings.Repeat("203.0.113.1,", 33), ","))
	if got := s.clientIP(tooManyHops); got != "10.0.0.2" {
		t.Fatalf("oversized proxy chain should fall back to proxy peer, got %q", got)
	}
}

func TestRateLimiterFailsClosedAtCapacityAndPurgesExpiredBuckets(t *testing.T) {
	s := &Server{limits: make(map[string]rate, 8192)}
	now := time.Now()
	for i := 0; i < 8192; i++ {
		s.limits[fmt.Sprintf("ip:%d", i)] = rate{at: now, count: 1}
	}
	if s.allowRate("ip:fresh", 120) {
		t.Fatal("a fresh client should be rejected while the bounded limiter is full")
	}
	if len(s.limits) != 8192 {
		t.Fatalf("rate limiter grew beyond its bound: %d", len(s.limits))
	}
	if _, ok := s.limits["ip:0"]; !ok {
		t.Fatal("active rate-limit bucket was evicted at capacity")
	}
	if !s.allowRate("ip:0", 120) {
		t.Fatal("existing client bucket should continue to work at capacity")
	}

	for key, bucket := range s.limits {
		bucket.at = now.Add(-3 * time.Minute)
		s.limits[key] = bucket
	}
	s.lastRatePurge = now.Add(-time.Minute)
	if !s.allowRate("ip:after-purge", 120) {
		t.Fatal("fresh client should be admitted after expired buckets are pruned")
	}
	if len(s.limits) != 1 {
		t.Fatalf("expected expired buckets pruned before admitting new client, got %d", len(s.limits))
	}
}

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
			s, e := New(Options{Repository: JSONRepository{Dir: t.TempDir()}})
			if e != nil {
				t.Fatal(e)
			}
			players := make([]*User, d.Players)
			players[0] = &User{ID: "owner", Name: "玩家1"}
			requestSequence := 0
			nextRequestID := func() string {
				requestSequence++
				return fmt.Sprintf("lifecycle-%d", requestSequence)
			}
			v, e := s.command(players[0], Command{Op: "create", RequestID: "create", Config: game.Config{Game: d.Key}})
			if e != nil {
				t.Fatal(e)
			}
			rid := v.(map[string]any)["id"].(string)
			for seat := 1; seat < d.Players; seat++ {
				players[seat] = &User{ID: fmt.Sprintf("player-%d", seat), Name: fmt.Sprintf("玩家%d", seat+1)}
				if _, e = s.command(players[seat], Command{Op: "join", RoomID: rid, RequestID: nextRequestID()}); e != nil {
					t.Fatal(e)
				}
			}
			apply := func(user *User, op string) {
				t.Helper()
				_, err := s.command(user, Command{Op: op, RoomID: rid, RequestID: nextRequestID(), Version: s.rooms[rid].Version})
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, user := range players {
				apply(user, "ready")
			}
			apply(players[0], "start")
			for step := 0; s.rooms[rid].Status == "playing"; step++ {
				if step > 5000 {
					t.Fatal("room stuck")
				}
				r := s.rooms[rid]
				seat := -1
				var action game.Action
				for i := range r.Players {
					if legal := game.Legal(r.State, i); len(legal) > 0 {
						seat = i
						action = game.Auto(r.State, i)
						break
					}
				}
				if seat < 0 {
					t.Fatal("no player has a legal action")
				}
				_, e = s.command(players[seat], Command{Op: "action", RoomID: rid, RequestID: nextRequestID(), Version: r.Version, Action: action})
				if e != nil {
					t.Fatalf("player %d action %s: %v", seat, action.Kind, e)
				}
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
			for _, user := range players {
				apply(user, "dissolve")
			}
			if r.Status != "closed" {
				t.Fatal("all players should dissolve the room")
			}
		})
	}
}

func TestRoomViewHidesPrivateGameInformation(t *testing.T) {
	room := &Room{
		ID: "room-1", Owner: "u0", Status: "playing",
		Players: []*Player{{UserID: "u0", Name: "甲"}, {UserID: "u1", Name: "乙"}},
		State: &game.State{
			Config: game.Config{Game: "hongzhong"}, Phase: "discard", Turn: 0, Wall: []int{999},
			Seats: []game.Seat{
				{Hand: []int{101}, Melds: []game.Meld{{Kind: "concealed-kong", Cards: []int{11, 11, 11, 11}, Hidden: true}}},
				{Hand: []int{202}, Melds: []game.Meld{{Kind: "concealed-kong", Cards: []int{22, 22, 22, 22}, Hidden: true}}},
			},
			Offer:  &game.Offer{Card: 303, From: 1, Source: "discard", Replies: map[int]game.Action{0: {Kind: "win", Cards: []int{404}}}, Eligible: []int{0}},
			Events: []game.Event{{Seat: 0, Kind: "discard", Cards: []int{505}, Text: "公开出牌"}, {Seat: 1, Kind: "draw", Cards: []int{606}, Text: "摸牌"}},
			Wins:   []game.Win{{Seat: 1, Hand: []int{707}}},
		},
	}
	s := &Server{}
	decode := func() map[string]any {
		t.Helper()
		data, err := json.Marshal(s.view(room, "u0"))
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	contains := func(value any, secret string) bool {
		t.Helper()
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return bytes.Contains(encoded, []byte(secret))
	}

	got := decode()
	state := got["state"].(map[string]any)
	seats := state["seats"].([]any)
	if len(seats[0].(map[string]any)["hand"].([]any)) != 1 {
		t.Fatal("viewer should receive their own hand")
	}
	opponent := seats[1].(map[string]any)
	if len(opponent["hand"].([]any)) != 0 || opponent["count"] != float64(1) {
		t.Fatal("opponent hand should be hidden while its count remains public")
	}
	if contains(got, "202") || contains(got, "606") || contains(got, "999") || contains(got, "replies") || contains(got, "eligible") {
		t.Fatal("live room view leaked an opponent hand, private draw, wall order, or offer response")
	}
	oppMeld := opponent["melds"].([]any)[0].(map[string]any)
	if cards := oppMeld["cards"].([]any); len(cards) != 4 || cards[0] != float64(0) {
		t.Fatal("opponent concealed meld values should be masked")
	}

	room.State.End = "win"
	got = decode()
	if !contains(got, "202") || !contains(got, "606") || !contains(got, "707") || !contains(got, "22") {
		t.Fatal("completed round should reveal its hands, events, and concealed melds")
	}

	room.State.End = ""
	room.State.Phase = "float"
	got = decode()
	state = got["state"].(map[string]any)
	for _, seat := range state["seats"].([]any) {
		if len(seat.(map[string]any)["hand"].([]any)) != 0 {
			t.Fatal("float phase must hide every hand")
		}
	}
	if _, ok := state["offer"]; ok || len(state["events"].([]any)) != 0 {
		t.Fatal("float phase must not expose offer or events")
	}
}

func TestUniqueIdentifiersRetryCollisionsAndPropagateRandomErrors(t *testing.T) {
	reader := bytes.NewReader([]byte{1, 2})
	value, err := uniqueIDFrom(reader, 1, func(value string) bool { return value == "01" })
	if err != nil || value != "02" {
		t.Fatalf("unique ID should retry a collision: value=%q err=%v", value, err)
	}
	if _, err := idWithReader(bytes.NewReader([]byte{1}), 2); err == nil {
		t.Fatal("short entropy source must fail")
	}
	if _, err := uniqueIDFrom(bytes.NewReader(make([]byte, 32)), 1, func(string) bool { return true }); err == nil {
		t.Fatal("identifier allocation must stop after its bounded collision retries")
	}
	randomCalls := 0
	roomID, err := uniqueRoomIDFrom(func(n int) (int, error) {
		randomCalls++
		if n != 900000 {
			t.Fatalf("unexpected room ID bound: %d", n)
		}
		if randomCalls == 1 {
			return 7, nil
		}
		return 8, nil
	}, func(id string) bool { return id == "100007" })
	if err != nil || roomID != "100008" || randomCalls != 2 {
		t.Fatalf("room ID should retry collision: value=%q calls=%d err=%v", roomID, randomCalls, err)
	}
	wantErr := errors.New("entropy unavailable")
	if _, err := uniqueRoomIDFrom(func(int) (int, error) { return 0, wantErr }, func(string) bool { return false }); !errors.Is(err, wantErr) {
		t.Fatalf("random source failure should propagate, got %v", err)
	}
}

// Opt-in bounded connection fan-out benchmark, not a unique-player or WAN test.
func TestConnectionFanout(t *testing.T) {
	if os.Getenv("HJY_LOAD") != "1" {
		t.Skip("set HJY_LOAD=1")
	}
	for _, n := range []int{100, 300, 500} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s, e := New(Options{Repository: JSONRepository{Dir: t.TempDir()}, MaxWSPerUser: n, MaxWSPerIP: n, MaxWSGlobal: n})
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
