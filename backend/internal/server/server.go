package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/coder/websocket"
	"haojiayuan/internal/game"
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	UsernameKey  string `json:"usernameKey"`
	PasswordHash string `json:"passwordHash"`
	Name         string `json:"name"`
	Avatar       int    `json:"avatar"`
	Created      int64  `json:"created"`
}
type Session struct {
	UserID  string `json:"userId"`
	Expires int64  `json:"expires"`
}
type ClubTemplate struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Config  game.Config `json:"config"`
	Enabled bool        `json:"enabled"`
}
type Club struct {
	ID                       string           `json:"id"`
	Name                     string           `json:"name"`
	Owner                    string           `json:"owner"`
	Members                  map[string]bool  `json:"members"`
	Managers                 map[string]bool  `json:"managers,omitempty"`
	JoinClosed               bool             `json:"joinClosed"`
	RequireApproval          bool             `json:"requireApproval"`
	AutoCreate               bool             `json:"autoCreate"`
	Announcement             string           `json:"announcement"`
	Applications             map[string]int64 `json:"applications,omitempty"`
	Disabled                 map[string]bool  `json:"disabled,omitempty"`
	Templates                []ClubTemplate   `json:"templates"`
	ExternalRecordsTokenHash string           `json:"externalRecordsTokenHash,omitempty"`
	ExternalRecordsEnabled   bool             `json:"externalRecordsEnabled,omitempty"`
}
type Report struct {
	ID      string `json:"id"`
	UserID  string `json:"userId"`
	RoomID  string `json:"roomId"`
	Text    string `json:"text"`
	Created int64  `json:"created"`
}
type Accounts struct {
	Users    map[string]*User   `json:"users"`
	Sessions map[string]Session `json:"sessions"`
	Clubs    map[string]*Club   `json:"clubs"`
	Reports  []Report           `json:"reports"`
}
type Player struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Avatar int    `json:"avatar"`
	Ready  bool   `json:"ready"`
	Total  int    `json:"total"`
}
type Round struct {
	Number int             `json:"number"`
	End    string          `json:"end"`
	Scores []int           `json:"scores"`
	Wins   []game.Win      `json:"wins"`
	Ledger []game.Transfer `json:"ledger"`
	At     int64           `json:"at"`
}
type Room struct {
	ID           string          `json:"id"`
	Owner        string          `json:"owner"`
	ClubID       string          `json:"clubId,omitempty"`
	TemplateID   string          `json:"templateId,omitempty"`
	Config       game.Config     `json:"config"`
	Players      []*Player       `json:"players"`
	Status       string          `json:"status"`
	Version      int             `json:"version"`
	Round        int             `json:"round"`
	State        *game.State     `json:"state,omitempty"`
	History      []Round         `json:"history"`
	Created      int64           `json:"created"`
	Updated      int64           `json:"updated"`
	Votes        map[string]bool `json:"votes,omitempty"`
	VoteDeadline int64           `json:"voteDeadline,omitempty"`
	Used         []string        `json:"used,omitempty"`
}
type Options struct {
	Repository   Repository
	StaticDir    string
	AdminKey     string
	CookieSecure bool
}
type peer struct {
	user, room string
	session    string
	ch         chan []byte
	conn       *websocket.Conn
}
type Server struct {
	mu                           sync.Mutex
	repo                         Repository
	accounts                     Accounts
	rooms                        map[string]*Room
	peers                        map[*peer]bool
	static, admin                string
	cookieSecure                 bool
	requests, commands, failures uint64
	lastError                    string
	limits                       map[string]rate
	started                      time.Time
	lastClubHistoryPurge         time.Time
	apiStats, commandStats       durationStats
}
type rate struct {
	at    time.Time
	count int
}
type Command struct {
	Op         string      `json:"op"`
	RoomID     string      `json:"roomId"`
	Version    int         `json:"version"`
	RequestID  string      `json:"requestId"`
	Config     game.Config `json:"config"`
	Action     game.Action `json:"action"`
	ClubID     string      `json:"clubId"`
	TemplateID string      `json:"templateId"`
	Value      int         `json:"value"`
}
type apiError struct {
	status  int
	message string
}

func (e apiError) Error() string { return e.message }
func bad(s string) error         { return apiError{400, s} }
func conflict(s string) error    { return apiError{409, s} }
func forbidden() error           { return apiError{403, "无权执行此操作"} }
func id(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func clone[T any](v T) T   { b, _ := json.Marshal(v); var out T; _ = json.Unmarshal(b, &out); return out }
func New(o Options) (*Server, error) {
	if o.Repository == nil {
		return nil, errors.New("repository required")
	}
	a, rooms, err := o.Repository.Load()
	if err != nil {
		return nil, err
	}
	s := &Server{repo: o.Repository, accounts: a, rooms: map[string]*Room{}, peers: map[*peer]bool{}, static: o.StaticDir, admin: o.AdminKey, cookieSecure: o.CookieSecure, limits: map[string]rate{}}
	s.started = time.Now()
	for _, r := range rooms {
		legacyBot := false
		for i, p := range r.Players {
			if p != nil && (strings.HasPrefix(p.UserID, "bot-") || strings.HasPrefix(p.Name, "练习伙伴")) {
				legacyBot = true
				if r.Status == "waiting" {
					r.Players[i] = nil
				}
			}
		}
		if legacyBot {
			if r.Status == "waiting" {
				for _, p := range r.Players {
					if p != nil {
						p.Ready = false
					}
				}
			} else if r.Status == "playing" || r.Status == "roundEnd" {
				r.Status = "closed"
			}
			r.Version++
			r.Updated = time.Now().Unix()
			if err := o.Repository.SaveRoom(r); err != nil {
				return nil, err
			}
		}
		s.rooms[r.ID] = r
	}
	if err := s.pruneExpiredClubHistory(time.Now()); err != nil {
		return nil, err
	}
	s.lastClubHistoryPurge = time.Now()
	return s, nil
}
func (s *Server) Run(ctx context.Context) {
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			for p := range s.peers {
				_ = p.conn.CloseNow()
			}
			s.mu.Unlock()
			return
		case now := <-t.C:
			s.tick(now)
		}
	}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", s.observeAPI)
	mux.HandleFunc("/metrics", s.metrics)
	mux.HandleFunc("/ws", s.ws)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { jsonOut(w, 200, map[string]any{"ok": true}) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		jsonOut(w, 200, map[string]any{"ok": true, "storage": "json-single-writer"})
	})
	mux.HandleFunc("/", s.files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		s.mu.Lock()
		s.requests++
		s.mu.Unlock()
		defer func() {
			if v := recover(); v != nil {
				log.Printf("request panic: %v", v)
				jsonOut(w, 500, map[string]string{"error": "服务器内部错误"})
			}
		}()
		mux.ServeHTTP(w, r)
	})
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	d := json.NewDecoder(io.LimitReader(r.Body, 65537))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return bad("请求格式错误")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return bad("请求内容过长或包含多份JSON")
	}
	return nil
}
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, e := url.Parse(origin)
	return e == nil && u.Host == r.Host && (u.Scheme == "http" || u.Scheme == "https")
}
func (s *Server) user(r *http.Request) *User {
	c, e := r.Cookie("hjy_session")
	if e != nil {
		return nil
	}
	v, ok := s.accounts.Sessions[hash(c.Value)]
	if !ok || v.Expires < time.Now().Unix() {
		return nil
	}
	return s.accounts.Users[v.UserID]
}
func publicUser(u *User) any {
	return map[string]any{"id": u.ID, "username": u.Username, "name": u.Name, "avatar": u.Avatar}
}
func (s *Server) saveAccounts(before Accounts) error {
	if e := s.repo.SaveAccounts(s.accounts); e != nil {
		s.accounts = before
		s.storageError(e)
		return apiError{503, "保存失败，请稍后重试"}
	}
	return nil
}
func (s *Server) storageError(e error) {
	s.failures++
	s.lastError = e.Error()
	log.Printf("storage: %v", e)
}
func (s *Server) issue(u *User) string {
	token := id(32)
	s.accounts.Sessions[hash(token)] = Session{u.ID, time.Now().Add(30 * 24 * time.Hour).Unix()}
	return token
}
func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{Name: "hjy_session", Value: token, Path: "/", HttpOnly: true, Secure: s.cookieSecure || r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 3600})
}

const passwordRounds = 120000

func pbkdf2SHA256(password, salt []byte, rounds, length int) []byte {
	result := make([]byte, 0, length)
	for block := uint32(1); len(result) < length; block++ {
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		mixed := append([]byte(nil), u...)
		for i := 1; i < rounds; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range mixed {
				mixed[j] ^= u[j]
			}
		}
		result = append(result, mixed...)
	}
	return result[:length]
}

func passwordHash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := pbkdf2SHA256([]byte(password), salt, passwordRounds, 32)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", passwordRounds, hex.EncodeToString(salt), hex.EncodeToString(derived)), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	rounds, err := strconv.Atoi(parts[1])
	if err != nil || rounds < 100000 || rounds > 1000000 {
		return false
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil || len(salt) < 16 || len(salt) > 64 {
		return false
	}
	expected, err := hex.DecodeString(parts[3])
	if err != nil || len(expected) != 32 {
		return false
	}
	actual := pbkdf2SHA256([]byte(password), salt, rounds, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func validUsername(username string) bool {
	runes := []rune(username)
	if len(runes) < 3 || len(runes) > 24 {
		return false
	}
	for _, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func (s *Server) usernameTaken(key string) bool {
	for _, user := range s.accounts.Users {
		userKey := user.UsernameKey
		if userKey == "" {
			userKey = strings.ToLower(user.Username)
		}
		if userKey == key {
			return true
		}
	}
	return false
}
func (s *Server) authRate(r *http.Request) bool {
	k, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		k = r.RemoteAddr
	}
	v := s.limits[k]
	if time.Since(v.at) > time.Minute {
		v = rate{at: time.Now()}
	}
	v.count++
	s.limits[k] = v
	return v.count <= 15
}
func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	externalRecords := strings.HasPrefix(r.URL.Path, "/api/v1/teahouses/")
	if externalRecords {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	} else if r.Method != "GET" && !sameOrigin(r) {
		jsonOut(w, 403, map[string]string{"error": "来源校验失败"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var result any
	var err error
	u := s.user(r)
	switch r.URL.Path {
	case "/api/session":
		switch r.Method {
		case "GET":
			if u == nil {
				err = apiError{401, "请先登录"}
			} else {
				result = map[string]any{"user": publicUser(u), "roomId": s.activeRoom(u.ID)}
			}
		case "POST":
			err = apiError{405, "请使用用户名和密码注册或登录"}
		case "DELETE":
			before := clone(s.accounts)
			if c, e := r.Cookie("hjy_session"); e == nil {
				delete(s.accounts.Sessions, hash(c.Value))
			}
			err = s.saveAccounts(before)
			http.SetCookie(w, &http.Cookie{Name: "hjy_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cookieSecure || r.TLS != nil, SameSite: http.SameSiteLaxMode})
			result = map[string]bool{"ok": true}
		default:
			err = apiError{405, "请求方法不支持"}
		}
	case "/api/register", "/api/login":
		if r.Method != "POST" {
			err = apiError{405, "请求方法不支持"}
			break
		}
		if !s.authRate(r) {
			err = apiError{429, "操作频繁"}
			break
		}
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err = decode(r, &input); err != nil {
			break
		}
		input.Username = strings.TrimSpace(input.Username)
		if r.URL.Path == "/api/register" {
			if !validUsername(input.Username) {
				err = bad("用户名需为3到24位字母、数字、中文、下划线或连字符")
				break
			}
			if len([]rune(input.Password)) < 8 || len([]rune(input.Password)) > 64 {
				err = bad("密码需为8到64个字符")
				break
			}
			key := strings.ToLower(input.Username)
			if s.usernameTaken(key) {
				err = conflict("用户名已被使用")
				break
			}
			encoded, hashErr := passwordHash(input.Password)
			if hashErr != nil {
				err = apiError{500, "账号创建失败，请稍后重试"}
				break
			}
			before := clone(s.accounts)
			u = &User{ID: id(6), Username: input.Username, UsernameKey: key, PasswordHash: encoded, Name: input.Username, Avatar: game.Random(8), Created: time.Now().Unix()}
			s.accounts.Users[u.ID] = u
			token := s.issue(u)
			err = s.saveAccounts(before)
			if err != nil {
				break
			}
			s.setSessionCookie(w, r, token)
			result = map[string]any{"user": publicUser(u), "roomId": s.activeRoom(u.ID)}
		} else {
			if input.Username == "" || input.Password == "" {
				err = apiError{401, "用户名或密码错误"}
				break
			}
			key := strings.ToLower(input.Username)
			for _, candidate := range s.accounts.Users {
				candidateKey := candidate.UsernameKey
				if candidateKey == "" {
					candidateKey = strings.ToLower(candidate.Username)
				}
				if candidateKey == key {
					u = candidate
					break
				}
			}
			if u == nil || !verifyPassword(u.PasswordHash, input.Password) {
				// Spend a comparable amount of work for unknown usernames to reduce account enumeration by timing.
				_, _ = passwordHash(input.Password)
				err = apiError{401, "用户名或密码错误"}
				break
			}
			before := clone(s.accounts)
			token := s.issue(u)
			err = s.saveAccounts(before)
			if err != nil {
				break
			}
			s.setSessionCookie(w, r, token)
			result = map[string]any{"user": publicUser(u), "roomId": s.activeRoom(u.ID)}
		}
	case "/api/recover":
		err = apiError{404, "接口不存在"}
		break
	case "/api/games":
		result = game.Definitions
	case "/api/admin":
		key := r.Header.Get("X-Admin-Key")
		if s.admin == "" || subtle.ConstantTimeCompare([]byte(key), []byte(s.admin)) != 1 {
			err = forbidden()
			break
		}
		if r.Method != "GET" {
			err = apiError{405, "请求方法不支持"}
			break
		}
		result = s.adminView()
	default:
		if strings.HasPrefix(r.URL.Path, "/api/v1/teahouses/") {
			result, err = s.externalClubRecords(r)
			break
		}
		if u == nil {
			err = apiError{401, "请先登录"}
			break
		}
		switch r.URL.Path {
		case "/api/rooms":
			var list []any
			for _, room := range s.rooms {
				if room.Status == "waiting" && (room.ClubID == "" || s.isMember(room.ClubID, u.ID)) {
					list = append(list, s.summary(room))
				}
			}
			result = list
		case "/api/room":
			room := s.rooms[r.URL.Query().Get("id")]
			if room == nil {
				err = apiError{404, "房间不存在"}
			} else if seatOf(room, u.ID) < 0 {
				err = forbidden()
			} else {
				result = s.view(room, u.ID)
			}
		case "/api/command":
			if r.Method != "POST" {
				err = apiError{405, "请求方法不支持"}
				break
			}
			var cmd Command
			if err = decode(r, &cmd); err == nil {
				result, err = s.command(u, cmd)
			}
		case "/api/history":
			var list []any
			for _, room := range s.rooms {
				if seatOf(room, u.ID) >= 0 && len(room.History) > 0 {
					list = append(list, map[string]any{"room": s.summary(room), "players": room.Players, "rounds": room.History, "updated": room.Updated})
				}
			}
			sort.Slice(list, func(i, j int) bool {
				return list[i].(map[string]any)["updated"].(int64) > list[j].(map[string]any)["updated"].(int64)
			})
			result = list
		case "/api/clubs":
			result, err = s.clubsAPI(r, u)
		case "/api/club":
			if r.Method != "GET" {
				err = apiError{405, "请求方法不支持"}
				break
			}
			c := s.accounts.Clubs[r.URL.Query().Get("id")]
			if c == nil || !c.Members[u.ID] {
				err = forbidden()
				break
			}
			result = s.clubView(c, u.ID)
		case "/api/profile":
			if r.Method != "POST" {
				err = apiError{405, "请求方法不支持"}
				break
			}
			var input struct {
				Name   string `json:"name"`
				Avatar int    `json:"avatar"`
			}
			if err = decode(r, &input); err != nil {
				break
			}
			input.Name = strings.TrimSpace(input.Name)
			if len([]rune(input.Name)) < 1 || len([]rune(input.Name)) > 16 || input.Avatar < 0 || input.Avatar > 7 {
				err = bad("昵称或头像无效")
				break
			}
			before := clone(s.accounts)
			u.Name = input.Name
			u.Avatar = input.Avatar
			err = s.saveAccounts(before)
			result = publicUser(u)
		case "/api/reports":
			if r.Method != "POST" {
				err = apiError{405, "请求方法不支持"}
				break
			}
			var input struct {
				RoomID string `json:"roomId"`
				Text   string `json:"text"`
			}
			if err = decode(r, &input); err != nil {
				break
			}
			if len([]rune(strings.TrimSpace(input.Text))) < 3 || len([]rune(input.Text)) > 1000 {
				err = bad("反馈需要3到1000字")
				break
			}
			if input.RoomID != "" && (s.rooms[input.RoomID] == nil || seatOf(s.rooms[input.RoomID], u.ID) < 0) {
				err = forbidden()
				break
			}
			before := clone(s.accounts)
			report := Report{id(6), u.ID, input.RoomID, input.Text, time.Now().Unix()}
			s.accounts.Reports = append(s.accounts.Reports, report)
			err = s.saveAccounts(before)
			result = map[string]string{"id": report.ID}
		default:
			err = apiError{404, "接口不存在"}
		}
	}
	if err != nil {
		status := 400
		var ae apiError
		if errors.As(err, &ae) {
			status = ae.status
		}
		failure := map[string]any{"error": err.Error()}
		if result != nil {
			failure["room"] = result
		}
		jsonOut(w, status, failure)
		return
	}
	jsonOut(w, 200, result)
}
func (s *Server) isMember(club, user string) bool {
	c := s.accounts.Clubs[club]
	return c != nil && c.Members[user] && !c.Disabled[user]
}
func seatOf(r *Room, user string) int {
	for i, p := range r.Players {
		if p != nil && p.UserID == user {
			return i
		}
	}
	return -1
}
func (s *Server) activeRoom(user string) string {
	for _, r := range s.rooms {
		if r.Status != "finished" && r.Status != "closed" && seatOf(r, user) >= 0 {
			return r.ID
		}
	}
	return ""
}
func (s *Server) summary(r *Room) any {
	count := 0
	for _, p := range r.Players {
		if p != nil {
			count++
		}
	}
	return map[string]any{"id": r.ID, "config": r.Config, "status": r.Status, "players": count, "capacity": len(r.Players), "round": r.Round, "clubId": r.ClubID, "templateId": r.TemplateID}
}
func (s *Server) command(u *User, c Command) (result any, err error) {
	start := time.Now()
	defer func() { s.commandStats.observe(time.Since(start).Seconds(), err != nil) }()
	s.commands++
	if c.RequestID == "" || len(c.RequestID) > 96 {
		return nil, bad("缺少有效请求编号")
	}
	if c.Op == "create" {
		// A retried create returns the already-created room, including after restart.
		for _, r := range s.rooms {
			for _, used := range r.Used {
				if used == u.ID+":"+c.RequestID {
					if seatOf(r, u.ID) < 0 {
						return nil, forbidden()
					}
					return s.view(r, u.ID), nil
				}
			}
		}
		if active := s.activeRoom(u.ID); active != "" {
			return nil, conflict("已有未结束房间：" + active)
		}
		if c.ClubID != "" {
			club := s.accounts.Clubs[c.ClubID]
			if !s.isMember(c.ClubID, u.ID) {
				return nil, forbidden()
			}
			if c.TemplateID == "" {
				return nil, bad("请从茶馆已启用的玩法开桌")
			}
			found := false
			for _, template := range club.Templates {
				if template.ID == c.TemplateID && template.Enabled {
					c.Config = clone(template.Config)
					found = true
					break
				}
			}
			if !found {
				return nil, bad("茶馆玩法不存在或已停用")
			}
		} else if c.TemplateID != "" {
			return nil, bad("玩法必须属于指定茶馆")
		}
		cfg, e := game.Normalize(c.Config)
		if e != nil {
			return nil, e
		}

		d, _ := game.Def(cfg.Game)
		roomID := ""
		for roomID == "" || s.rooms[roomID] != nil {
			roomID = strconv.Itoa(100000 + game.Random(900000))
		}
		r := &Room{ID: roomID, Owner: u.ID, ClubID: c.ClubID, TemplateID: c.TemplateID, Config: cfg, Players: make([]*Player, d.Players), Status: "waiting", Version: 1, Created: time.Now().Unix(), Updated: time.Now().Unix(), Used: []string{u.ID + ":" + c.RequestID}}
		r.Players[0] = &Player{UserID: u.ID, Name: u.Name, Avatar: u.Avatar}
		if e = s.repo.SaveRoom(r); e != nil {
			s.storageError(e)
			return nil, apiError{503, "保存失败"}
		}
		s.rooms[r.ID] = r
		return s.view(r, u.ID), nil
	}
	r := s.rooms[c.RoomID]
	if r == nil {
		return nil, apiError{404, "房间不存在"}
	}
	key := u.ID + ":" + c.RequestID
	p := seatOf(r, u.ID)
	if c.Op != "join" && p < 0 {
		return nil, forbidden()
	}
	for _, v := range r.Used {
		if v == key {
			if p < 0 {
				return nil, forbidden()
			}
			return s.view(r, u.ID), nil
		}
	}
	if c.Op != "join" && c.Version != r.Version {
		return s.view(r, u.ID), conflict("房间已更新，请按最新状态操作")
	}
	before := clone(r)
	switch c.Op {
	case "join":
		if p >= 0 {
			return s.view(r, u.ID), nil
		}
		if s.activeRoom(u.ID) != "" {
			return nil, conflict("请先离开当前房间")
		}
		if r.Status != "waiting" {
			return nil, conflict("房间已经开始")
		}
		if r.ClubID != "" && !s.isMember(r.ClubID, u.ID) {
			return nil, forbidden()
		}
		slot := -1
		for i, v := range r.Players {
			if v == nil {
				slot = i
				break
			}
		}
		if slot < 0 {
			return nil, conflict("房间已满")
		}
		r.Players[slot] = &Player{UserID: u.ID, Name: u.Name, Avatar: u.Avatar}
		if r.Owner == "" {
			r.Owner = u.ID
		}
	case "ready":
		if r.Status != "waiting" && r.Status != "roundEnd" {
			err = bad("当前不能准备")
			break
		}
		r.Players[p].Ready = !r.Players[p].Ready
	case "start", "next":
		if r.Owner != u.ID || (r.Status != "waiting" && r.Status != "roundEnd") {
			err = forbidden()
			break
		}
		for _, v := range r.Players {
			if v == nil || !v.Ready {
				err = bad("请等待所有玩家准备")
				break
			}
		}
		if err != nil {
			break
		}
		dealer := -1
		if r.State != nil {
			dealer = r.State.Dealer
			if len(r.State.Wins) > 0 {
				dealer = r.State.Wins[0].Seat
			}
		}
		r.State, err = game.New(r.Config, dealer)
		if err == nil {
			r.Round++
			r.Status = "playing"
			s.complete(r)
		}
	case "action":
		if r.Status != "playing" {
			err = bad("当前不在对局中")
			break
		}
		err = game.Apply(r.State, p, c.Action)
		if err == nil {
			s.complete(r)
		}
	case "leave":
		if r.Status == "finished" || r.Status == "closed" {
			break // Preserve participants in completed history.
		}
		if r.Status == "playing" || r.Status == "roundEnd" {
			err = bad("对局已开始，请发起解散投票")
			break
		}
		r.Players[p] = nil
		if r.Owner == u.ID {
			r.Owner = ""
			for _, v := range r.Players {
				if v != nil {
					r.Owner = v.UserID
					break
				}
			}
		}
		if r.Owner == "" {
			r.Status = "closed"
		}
	case "dissolve":
		if r.Status == "closed" || r.Status == "finished" {
			err = bad("房间已结束")
			break
		}
		if r.Votes == nil {
			r.Votes = map[string]bool{}
			r.VoteDeadline = time.Now().Add(30 * time.Second).UnixMilli()
		}
		r.Votes[u.ID] = true
		all := true
		for _, v := range r.Players {
			if v != nil && !r.Votes[v.UserID] {
				all = false
			}
		}
		if all {
			r.Status = "closed"
			r.Votes = nil
			r.VoteDeadline = 0
		}
	case "rejectDissolve":
		r.Votes = nil
		r.VoteDeadline = 0
	default:
		err = bad("操作不支持")
	}
	if err != nil {
		s.rooms[r.ID] = before
		return nil, err
	}
	r.Version++
	r.Updated = time.Now().Unix()
	r.Used = append(r.Used, key)
	if len(r.Used) > 256 {
		r.Used = r.Used[len(r.Used)-256:]
	}
	if err = s.repo.SaveRoom(r); err != nil {
		s.rooms[r.ID] = before
		s.storageError(err)
		return nil, apiError{503, "保存失败，操作未生效"}
	}
	s.broadcast(r)
	return s.view(r, u.ID), nil
}
func (s *Server) complete(r *Room) {
	if r.State == nil || r.State.End == "" || len(r.History) >= r.Round {
		return
	}
	rec := Round{Number: r.Round, End: r.State.End, Wins: r.State.Wins, Ledger: r.State.Ledger, At: time.Now().Unix()}
	for i, v := range r.State.Seats {
		r.Players[i].Total += v.Score
		rec.Scores = append(rec.Scores, v.Score)
	}
	r.History = append(r.History, rec)
	r.Status = "roundEnd"
	if r.Round >= r.Config.Rounds {
		r.Status = "finished"
	}
}
func (s *Server) tick(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.rooms {
		voteDue := r.Votes != nil && r.VoteDeadline <= now.UnixMilli()
		if !voteDue {
			continue
		}
		before := clone(r)
		changed := false
		if r.Votes != nil && r.VoteDeadline <= now.UnixMilli() {
			r.Votes = nil
			r.VoteDeadline = 0
			changed = true
		}
		if changed {
			r.Version++
			r.Updated = now.Unix()
			if err := s.repo.SaveRoom(r); err != nil {
				s.rooms[r.ID] = before
				s.storageError(err)
			} else {
				s.broadcast(r)
			}
		}
	}
	if now.Sub(s.lastClubHistoryPurge) >= time.Hour {
		if err := s.pruneExpiredClubHistory(now); err != nil {
			s.storageError(err)
		}
		s.lastClubHistoryPurge = now
	}
	s.ensureClubTables(now)
}

func newClubRecordsToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "hjy_" + hex.EncodeToString(raw), nil
}

// view deliberately builds an allow-listed projection. Never serialize State to a client.
func (s *Server) view(r *Room, user string) any {
	me := seatOf(r, user)
	v := map[string]any{"id": r.ID, "owner": r.Owner, "clubId": r.ClubID, "config": r.Config, "players": r.Players, "status": r.Status, "version": r.Version, "round": r.Round, "me": me, "history": r.History, "votes": r.Votes, "voteDeadline": r.VoteDeadline}
	if r.State == nil {
		return v
	}
	st := r.State
	hidden := st.Phase == "float"
	var seats []any
	for i, p := range st.Seats {
		hand := []int{}
		if !hidden && (i == me || st.End != "") {
			hand = append(hand, p.Hand...)
		}
		melds := []game.Meld{}
		if !hidden {
			for _, m := range p.Melds {
				if m.Hidden && i != me && st.End == "" {
					m.Cards = make([]int, len(m.Cards))
				}
				melds = append(melds, m)
			}
		}
		missing := p.Missing
		if st.Phase == "missing" && i != me {
			missing = -1
		}
		f := p.Float
		if hidden && i != me {
			f = -1
		}
		seats = append(seats, map[string]any{"hand": hand, "count": len(p.Hand), "melds": melds, "score": p.Score, "won": p.Won, "report": p.Report, "missing": missing, "float": f})
	}
	// Events can carry private draw cards or concealed sets. Their public text is
	// useful; only explicit public discard/play/claim events expose card values.
	events := []game.Event{}
	if !hidden {
		for _, e := range st.Events {
			if e.Seat != me && st.End == "" {
				e.Cards = nil
			}
			events = append(events, e)
		}
	}
	wins := append([]game.Win{}, st.Wins...)
	for i := range wins {
		if wins[i].Seat != me && st.End == "" {
			wins[i].Hand = nil
		}
	}
	// Empty piles are arrays in the public protocol, including immediately after
	// dealing. Nil slices would become null and break array-based clients.
	discards := make([][]int, len(st.Discards))
	for i, pile := range st.Discards {
		discards[i] = append([]int{}, pile...)
	}
	state := map[string]any{"seats": seats, "phase": st.Phase, "turn": st.Turn, "dealer": st.Dealer, "wallCount": len(st.Wall), "discards": discards, "table": append([]int{}, st.Table...), "tableSeat": st.TableSeat, "events": events, "end": st.End, "ledger": st.Ledger, "wins": wins}
	if st.Offer != nil && !hidden {
		state["offer"] = map[string]any{"card": st.Offer.Card, "from": st.Offer.From, "source": st.Offer.Source}
	}
	v["state"] = state
	v["legal"] = game.Legal(st, me)
	return v
}
func (s *Server) broadcast(r *Room) {
	for p := range s.peers {
		if p.room != r.ID {
			continue
		}
		session, valid := s.accounts.Sessions[p.session]
		if seatOf(r, p.user) < 0 || !valid || session.UserID != p.user || session.Expires <= time.Now().Unix() {
			_ = p.conn.CloseNow()
			delete(s.peers, p)
			continue
		}
		b, _ := json.Marshal(map[string]any{"type": "room", "room": s.view(r, p.user)})
		select {
		case p.ch <- b:
		default:
			_ = p.conn.CloseNow()
			delete(s.peers, p)
		}
	}
}
func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "origin rejected", 403)
		return
	}
	s.mu.Lock()
	u := s.user(r)
	room := s.rooms[r.URL.Query().Get("room")]
	if u == nil || room == nil || seatOf(room, u.ID) < 0 {
		s.mu.Unlock()
		http.Error(w, "forbidden", 403)
		return
	}
	userID := u.ID
	s.mu.Unlock()
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled})
	if err != nil {
		return
	}
	conn.SetReadLimit(65536)
	cookie, _ := r.Cookie("hjy_session")
	p := &peer{user: userID, room: room.ID, session: hash(cookie.Value), ch: make(chan []byte, 16), conn: conn}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer conn.CloseNow()
	s.mu.Lock()
	if s.user(r) == nil || seatOf(s.rooms[p.room], p.user) < 0 {
		s.mu.Unlock()
		return
	}
	s.peers[p] = true
	b, _ := json.Marshal(map[string]any{"type": "room", "room": s.view(s.rooms[p.room], p.user)})
	p.ch <- b
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.peers, p); s.mu.Unlock() }()
	go func() {
		defer cancel()
		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case b := <-p.ch:
				c, cancelWrite := context.WithTimeout(ctx, 8*time.Second)
				e := conn.Write(c, websocket.MessageText, b)
				cancelWrite()
				if e != nil {
					return
				}
			case <-heartbeat.C:
				c, cancelPing := context.WithTimeout(ctx, 8*time.Second)
				e := conn.Ping(c)
				cancelPing()
				if e != nil {
					return
				}
			}
		}
	}()
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var cmd Command
		if json.Unmarshal(b, &cmd) != nil {
			continue
		}
		if cmd.RoomID != p.room {
			continue
		}
		s.mu.Lock()
		current := s.user(r)
		var response any
		var cmdErr error
		if current == nil {
			cmdErr = forbidden()
		} else {
			response, cmdErr = s.command(current, cmd)
		}
		if cmdErr != nil {
			msg, _ := json.Marshal(map[string]any{"type": "error", "error": cmdErr.Error(), "requestId": cmd.RequestID, "room": s.view(s.rooms[p.room], p.user)})
			select {
			case p.ch <- msg:
			default:
				cancel()
			}
		} else {
			msg, _ := json.Marshal(map[string]any{"type": "ack", "requestId": cmd.RequestID, "room": response})
			select {
			case p.ch <- msg:
			default:
				cancel()
			}
		}
		s.mu.Unlock()
	}
}
func (s *Server) files(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "method", 405)
		return
	}
	if s.static == "" {
		http.Error(w, "frontend not built", 503)
		return
	}
	clean := filepath.Clean(filepath.FromSlash("/" + r.URL.Path))
	path := filepath.Join(s.static, strings.TrimLeft(clean, "\\/"))
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		if filepath.Ext(r.URL.Path) != "" {
			http.NotFound(w, r)
			return
		}
		path = filepath.Join(s.static, "index.html")
	}
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' ws: wss:; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; frame-ancestors 'none'")
	http.ServeFile(w, r, path)
}
