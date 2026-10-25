package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"haojiayuan/internal/game"
)

func TestClubPermissionsPersistenceAndAdminPrivacy(t *testing.T) {
	repo := JSONRepository{Dir: t.TempDir()}
	s, err := New(Options{Repository: repo, AdminKey: "test-admin-key"})
	if err != nil {
		t.Fatal(err)
	}
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	owner, manager, member, outsider := client(), client(), client(), client()
	ids := map[*http.Client]string{}
	for i, c := range []*http.Client{owner, manager, member, outsider} {
		v := register(t, c, h.URL, []string{"圈主账号", "管理账号", "成员账号", "外部账号"}[i])
		ids[c] = v["user"].(map[string]any)["id"].(string)
	}
	club := request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "create", "name": "亲友测试"}, 200)
	cid := club["id"].(string)
	change := func(c *http.Client, op, target, name string, enabled bool, want int) {
		request(t, c, "POST", h.URL+"/api/clubs", map[string]any{"op": op, "id": cid, "userId": target, "name": name, "enabled": enabled}, want)
	}
	change(manager, "join", "", "", false, 200)
	change(member, "join", "", "", false, 200)
	request(t, outsider, "GET", h.URL+"/api/club?id="+cid, nil, 403)
	change(member, "settings", "", "越权改名", false, 403)
	change(manager, "manager", ids[member], "", true, 403)
	change(owner, "manager", ids[manager], "", true, 200)
	change(manager, "settings", "", "好友茶桌", true, 200)
	change(outsider, "join", "", "", false, 403)
	change(manager, "remove", ids[owner], "", false, 400)
	change(owner, "manager", ids[member], "", true, 200)
	change(manager, "remove", ids[member], "", false, 403)
	change(owner, "manager", ids[member], "", false, 200)
	change(manager, "settings", "", "好友茶桌", false, 200)
	change(outsider, "join", "", "", false, 200)
	template := request(t, manager, "POST", h.URL+"/api/clubs", map[string]any{"op": "template", "id": cid, "template": map[string]any{"name": "红中二人", "enabled": true, "config": map[string]any{"game": "hongzhong", "base": 1, "rounds": 8}}}, 200)
	tid := template["templates"].([]any)[0].(map[string]any)["id"].(string)
	room := request(t, member, "POST", h.URL+"/api/command", map[string]any{"op": "create", "requestId": "club-room", "clubId": cid, "templateId": tid, "config": map[string]any{"game": "hongzhong"}}, 200)
	if r, ok := room["room"].(map[string]any); ok {
		room = r
	}
	change(manager, "remove", ids[member], "", false, 409)
	request(t, member, "POST", h.URL+"/api/command", map[string]any{"op": "leave", "roomId": room["id"], "version": room["version"], "requestId": "club-leave"}, 200)
	change(manager, "remove", ids[member], "", false, 200)
	view := request(t, manager, "GET", h.URL+"/api/club?id="+cid, nil, 200)
	if len(view["members"].([]any)) != 3 {
		t.Fatal("member removal missing")
	}
	request(t, member, "GET", h.URL+"/api/club?id="+cid, nil, 403)
	admin := func(key, method string, want int) map[string]any {
		r, _ := http.NewRequest(method, h.URL+"/api/admin", nil)
		r.Header.Set("X-Admin-Key", key)
		resp, e := owner.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		data := map[string]any{}
		_ = json.NewDecoder(resp.Body).Decode(&data)
		if resp.StatusCode != want {
			t.Fatalf("admin %s: %d want %d", method, resp.StatusCode, want)
		}
		return data
	}
	admin("wrong", "GET", 403)
	admin("test-admin-key", "POST", 405)
	data := admin("test-admin-key", "GET", 200)
	if number(data, "users") != 4 || len(data["clubList"].([]any)) != 1 {
		t.Fatal("admin counts")
	}
	for _, v := range []any{data, view} {
		b, _ := json.Marshal(v)
		for _, secret := range []string{"passwordHash", "usernameKey", "recoveryHash", "sessions", "\"wall\"", "\"state\""} {
			if strings.Contains(string(b), secret) {
				t.Fatalf("management leaks %s", secret)
			}
		}
	}
	reloaded, e := New(Options{Repository: repo})
	if e != nil {
		t.Fatal(e)
	}
	if c := reloaded.accounts.Clubs[cid]; c == nil || c.Name != "好友茶桌" || !c.Managers[ids[manager]] || c.Members[ids[member]] {
		t.Fatal("club mutation did not survive restart")
	}
}

func TestTeaHouseApprovalTemplatesAndPlayingRestrictions(t *testing.T) {
	repo := JSONRepository{Dir: t.TempDir()}
	s, err := New(Options{Repository: repo})
	if err != nil {
		t.Fatal(err)
	}
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	owner, member, applicant := client(), client(), client()
	ids := map[*http.Client]string{}
	for i, c := range []*http.Client{owner, member, applicant} {
		v := register(t, c, h.URL, []string{"馆主账号", "成员账号", "申请账号"}[i])
		ids[c] = v["user"].(map[string]any)["id"].(string)
	}
	club := request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "create", "name": "审核茶馆", "requireApproval": true}, 200)
	cid := club["id"].(string)
	change := func(c *http.Client, op, uid string, enabled bool, want int) map[string]any {
		return request(t, c, "POST", h.URL+"/api/clubs", map[string]any{"op": op, "id": cid, "userId": uid, "enabled": enabled}, want)
	}
	if v := change(member, "join", "", false, 200); v["pending"] != true {
		t.Fatal("join bypassed approval")
	}
	change(member, "join", "", false, 200)
	request(t, member, "GET", h.URL+"/api/club?id="+cid, nil, 403)
	v := request(t, owner, "GET", h.URL+"/api/club?id="+cid, nil, 200)
	if len(v["applications"].([]any)) != 1 {
		t.Fatal("duplicate application")
	}
	change(owner, "approve", ids[member], false, 200)
	change(owner, "approve", ids[member], false, 400)
	change(applicant, "join", "", false, 200)
	change(member, "approve", ids[applicant], false, 403)
	v = request(t, member, "GET", h.URL+"/api/club?id="+cid, nil, 200)
	if len(v["applications"].([]any)) != 0 || v["club"].(map[string]any)["applications"] != nil {
		t.Fatal("ordinary member saw pending applicants")
	}
	change(owner, "reject", ids[applicant], false, 200)
	request(t, applicant, "GET", h.URL+"/api/club?id="+cid, nil, 403)
	change(applicant, "join", "", false, 200)
	change(owner, "approve", ids[applicant], false, 200)
	setTemplate := func(tid string, enabled bool, base int) map[string]any {
		return request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "template", "id": cid, "template": map[string]any{"id": tid, "name": "固定红中", "enabled": enabled, "config": map[string]any{"game": "hongzhong", "base": base, "rounds": 16, "options": map[string]int{"root": 0, "ka": 1, "minFan": 2}}}}, 200)
	}
	v = setTemplate("", true, 2)
	tid := v["templates"].([]any)[0].(map[string]any)["id"].(string)
	request(t, member, "POST", h.URL+"/api/clubs", map[string]any{"op": "template", "id": cid, "template": map[string]any{"name": "越权", "config": map[string]any{"game": "runfast"}}}, 403)
	request(t, member, "POST", h.URL+"/api/command", map[string]any{"op": "create", "requestId": "missing-template", "clubId": cid, "config": map[string]any{"game": "runfast"}}, 400)
	room := request(t, member, "POST", h.URL+"/api/command", map[string]any{"op": "create", "requestId": "tea-create", "clubId": cid, "templateId": tid, "config": map[string]any{"game": "runfast", "base": 10}}, 200)
	cfg := room["config"].(map[string]any)
	if cfg["game"] != "hongzhong" || number(cfg, "base") != 2 || number(cfg, "rounds") != 16 || number(cfg["options"].(map[string]any), "root") != 0 {
		t.Fatal("client overrode tea-house configuration")
	}
	setTemplate(tid, false, 5)
	request(t, owner, "POST", h.URL+"/api/command", map[string]any{"op": "create", "requestId": "disabled-template", "clubId": cid, "templateId": tid}, 400)
	s.mu.Lock()
	roomBase := s.rooms[room["id"].(string)].Config.Base
	s.mu.Unlock()
	if roomBase != 2 {
		t.Fatal("template edit mutated existing table")
	}
	change(owner, "disable", ids[applicant], true, 200)
	request(t, applicant, "POST", h.URL+"/api/command", map[string]any{"op": "join", "requestId": "disabled-join", "roomId": room["id"]}, 403)
	change(owner, "disable", ids[member], true, 200)
	request(t, member, "POST", h.URL+"/api/command", map[string]any{"op": "join", "requestId": "seated-reconnect", "roomId": room["id"]}, 200)
	setTemplate(tid, true, 5)
	request(t, applicant, "POST", h.URL+"/api/command", map[string]any{"op": "create", "requestId": "disabled-create", "clubId": cid, "templateId": tid}, 403)
	change(owner, "disable", ids[applicant], false, 200)
	request(t, applicant, "POST", h.URL+"/api/command", map[string]any{"op": "join", "requestId": "enabled-join", "roomId": room["id"]}, 200)
	change(owner, "remove", ids[applicant], false, 409)
	request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "deleteTemplate", "id": cid, "templateId": tid}, 200)
	reloaded, err := New(Options{Repository: repo})
	if err != nil {
		t.Fatal(err)
	}
	c := reloaded.accounts.Clubs[cid]
	if !c.RequireApproval || !c.Disabled[ids[member]] || !c.Members[ids[applicant]] || len(c.Applications) != 0 || len(c.Templates) != 0 {
		t.Fatal("tea-house state did not survive restart")
	}
	if reloaded.rooms[room["id"].(string)].Config.Base != 2 {
		t.Fatal("persisted table changed with deleted template")
	}
}

func TestTeaHouseCompletedResultsAndStatistics(t *testing.T) {
	s, err := New(Options{Repository: JSONRepository{Dir: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	a, b := client(), client()
	uid := register(t, a, h.URL, "甲账号")["user"].(map[string]any)["id"].(string)
	other := register(t, b, h.URL, "乙账号")["user"].(map[string]any)["id"].(string)
	cid := request(t, a, "POST", h.URL+"/api/clubs", map[string]any{"op": "create", "name": "统计馆"}, 200)["id"].(string)
	request(t, b, "POST", h.URL+"/api/clubs", map[string]any{"op": "join", "id": cid}, 200)
	// Fixture completed rounds isolate aggregation from the separately tested game lifecycle.
	s.mu.Lock()
	now := time.Now().Unix()
	s.rooms["123456"] = &Room{ID: "123456", ClubID: cid, Config: game.Config{Game: "hongzhong", Base: 1, Rounds: 8}, Status: "finished", Players: []*Player{{UserID: uid, Name: "甲"}, {UserID: other, Name: "乙"}}, History: []Round{{Number: 1, Scores: []int{8, -8}, At: now}, {Number: 2, Scores: []int{-2, 2}, At: now}}, Updated: 20}
	s.rooms["654321"] = &Room{ID: "654321", ClubID: cid, Config: game.Config{Game: "hongzhong", Base: 1, Rounds: 8}, Status: "roundEnd", Players: []*Player{{UserID: uid, Name: "甲"}, {UserID: other, Name: "乙"}}, History: []Round{{Number: 1, Scores: []int{4, -4}, At: now}}, Updated: 30}
	s.mu.Unlock()
	v := request(t, b, "GET", h.URL+"/api/club?id="+cid, nil, 200)
	stats := v["stats"].(map[string]any)
	sa, sb := stats[uid].(map[string]any), stats[other].(map[string]any)
	if number(sa, "rounds") != 3 || number(sa, "total") != 10 || number(sa, "bigWins") != 1 || number(sb, "total") != -10 || number(sb, "bigWins") != 0 {
		t.Fatalf("wrong statistics: %+v", stats)
	}
	history := v["history"].([]any)
	if len(history) != 2 || history[0].(map[string]any)["room"].(map[string]any)["id"] != "654321" {
		t.Fatal("completed small rounds missing or unsorted")
	}
	bts, _ := json.Marshal(v)
	if strings.Contains(string(bts), `"state"`) || strings.Contains(string(bts), `"wall"`) {
		t.Fatal("tea-house history exposed live hands")
	}
}

func TestTeaHouseExternalRecordsAPIAndSevenDayRetention(t *testing.T) {
	repo := JSONRepository{Dir: t.TempDir()}
	s, err := New(Options{Repository: repo})
	if err != nil {
		t.Fatal(err)
	}
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	owner, member := client(), client()
	ownerID := register(t, owner, h.URL, "接口馆主")["user"].(map[string]any)["id"].(string)
	memberID := register(t, member, h.URL, "接口成员")["user"].(map[string]any)["id"].(string)
	cid := request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "create", "name": "接口战绩馆"}, 200)["id"].(string)
	request(t, member, "POST", h.URL+"/api/clubs", map[string]any{"op": "join", "id": cid}, 200)
	view := request(t, member, "GET", h.URL+"/api/club?id="+cid, nil, 200)
	clubJSON, _ := json.Marshal(view["club"])
	if strings.Contains(string(clubJSON), "externalRecordsTokenHash") || strings.Contains(string(clubJSON), "externalRecordsEnabled") {
		t.Fatal("club view contains removed external API credential fields")
	}

	now := time.Now()
	chinaNow := now.In(time.FixedZone("Asia/Shanghai", 8*60*60))
	todayStart := time.Date(chinaNow.Year(), chinaNow.Month(), chinaNow.Day(), 0, 0, 0, 0, chinaNow.Location())
	players := []*Player{{UserID: ownerID, Name: "馆主"}, {UserID: memberID, Name: "成员"}}
	rooms := []*Room{
		{ID: "210001", ClubID: cid, Config: game.Config{Game: "hongzhong"}, Status: "finished", Players: players, History: []Round{{Number: 1, Scores: []int{5, -5}, At: now.Add(-time.Minute).Unix()}}, Updated: now.Add(-time.Minute).Unix()},
		{ID: "210002", ClubID: cid, Config: game.Config{Game: "runfast"}, Status: "finished", Players: players, History: []Round{{Number: 1, Scores: []int{3, -3}, At: now.Add(-2 * time.Minute).Unix()}}, Updated: now.Add(-2 * time.Minute).Unix()},
		{ID: "210003", ClubID: cid, Config: game.Config{Game: "hongzhong"}, Status: "finished", Players: players, History: []Round{{Number: 1, Scores: []int{7, -7}, At: todayStart.Add(-time.Hour).Unix()}}, Updated: todayStart.Add(-time.Hour).Unix()},
		{ID: "210004", ClubID: cid, Config: game.Config{Game: "hongzhong"}, Status: "finished", Players: players, History: []Round{{Number: 1, Scores: []int{9, -9}, At: now.Add(-8 * 24 * time.Hour).Unix()}}, Updated: now.Add(-8 * 24 * time.Hour).Unix()},
	}
	for _, room := range rooms {
		if err := repo.SaveRoom(room); err != nil {
			t.Fatal(err)
		}
		s.rooms[room.ID] = room
	}
	if err := s.pruneExpiredClubHistory(now); err != nil {
		t.Fatal(err)
	}
	if len(s.rooms["210004"].History) != 0 || len(s.rooms["210001"].History) != 1 {
		t.Fatal("pruning did not remove only expired tea-house rounds")
	}

	endpoint := h.URL + "/api/v1/teahouses/" + cid + "/records"
	getExternal := func(url string, want int) map[string]any {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data := map[string]any{}
		_ = json.NewDecoder(resp.Body).Decode(&data)
		if resp.StatusCode != want {
			t.Fatalf("external records status %d want %d: %+v", resp.StatusCode, want, data)
		}
		return data
	}
	data := getExternal(endpoint+"?limit=1&memberId="+memberID, 200)
	if number(data, "total") != 3 || number(data, "retentionDays") != 7 || len(data["records"].([]any)) != 1 {
		t.Fatalf("unexpected paginated records response: %+v", data)
	}
	filtered := getExternal(endpoint+"?game=runfast", 200)
	if number(filtered, "total") != 1 {
		t.Fatalf("game filter returned unexpected total: %+v", filtered)
	}
	day := chinaNow.Format("2006-01-02")
	today := getExternal(endpoint+"?date=today", 200)
	if number(today, "total") != 2 {
		t.Fatalf("today filter returned unexpected total: %+v", today)
	}
	memberToday := getExternal(endpoint+"?date="+day+"&memberId="+memberID, 200)
	if number(memberToday, "total") != 2 {
		t.Fatalf("member and date filters returned unexpected total: %+v", memberToday)
	}
	for _, item := range memberToday["records"].([]any) {
		for _, round := range item.(map[string]any)["rounds"].([]any) {
			at := time.Unix(int64(round.(map[string]any)["at"].(float64)), 0).In(chinaNow.Location())
			if at.Format("2006-01-02") != day {
				t.Fatalf("date filter returned a round from another China calendar date: %+v", round)
			}
		}
	}
	getExternal(endpoint+"?date=2026-02-30", 400)

	preflight, err := http.NewRequest(http.MethodOptions, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	preflight.Header.Set("Origin", "https://mini.example")
	resp, err := http.DefaultClient.Do(preflight)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("CORS preflight failed: %d", resp.StatusCode)
	}

}

func TestTeaHouseAutomaticTablesAndManagerDissolution(t *testing.T) {
	repo := &failingRepo{JSONRepository: JSONRepository{Dir: t.TempDir()}}
	s, err := New(Options{Repository: repo})
	if err != nil {
		t.Fatal(err)
	}
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	owner, member := client(), client()
	register(t, owner, h.URL, "馆主账号")
	uid := register(t, member, h.URL, "成员账号")["user"].(map[string]any)["id"].(string)
	cid := request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "create", "name": "自动馆"}, 200)["id"].(string)
	request(t, member, "POST", h.URL+"/api/clubs", map[string]any{"op": "join", "id": cid}, 200)
	settings := map[string]any{"op": "settings", "id": cid, "name": "自动馆", "autoCreate": true}
	request(t, owner, "POST", h.URL+"/api/clubs", settings, 400)
	for _, g := range []string{"hongzhong", "sichuan"} {
		request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "template", "id": cid, "template": map[string]any{"name": g, "enabled": true, "config": map[string]any{"game": g}}}, 200)
	}
	request(t, member, "POST", h.URL+"/api/clubs", settings, 403)
	request(t, owner, "POST", h.URL+"/api/clubs", settings, 200)
	// Failed persistence must not publish an in-memory-only table.
	repo.fail = true
	s.tick(time.Now())
	if len(s.rooms) != 0 {
		t.Fatal("failed automatic table was published")
	}
	repo.fail = false
	s.tick(time.Now())
	if len(s.rooms) != 2 {
		t.Fatalf("expected one table per enabled template: %d", len(s.rooms))
	}
	var rid string
	for _, r := range s.rooms {
		want := 2
		if r.Config.Game == "sichuan" {
			want = 4
		} else {
			rid = r.ID
		}
		if len(r.Players) != want || r.Owner != "" || r.Status != "waiting" {
			t.Fatal("bad automatic table")
		}
	}
	s.tick(time.Now())
	if len(s.rooms) != 2 {
		t.Fatal("tick duplicated empty tables")
	}
	request(t, member, "POST", h.URL+"/api/command", map[string]any{"op": "join", "requestId": "auto-seat", "roomId": rid}, 200)
	if s.rooms[rid].Owner != uid {
		t.Fatal("first human did not become table owner")
	}
	s.tick(time.Now())
	if len(s.rooms) != 3 {
		t.Fatal("occupied table did not trigger a replacement empty table")
	}
	reloaded, err := New(Options{Repository: repo})
	if err != nil {
		t.Fatal(err)
	}
	reloaded.tick(time.Now())
	if len(reloaded.rooms) != 3 || !reloaded.accounts.Clubs[cid].AutoCreate {
		t.Fatal("restart duplicated tables or lost auto setting")
	}
	// Closing an active room retains completed rounds and never settles the unfinished round.
	r := s.rooms[rid]
	r.Status = "playing"
	r.History = []Round{{Number: 1, Scores: []int{4, -4}, At: time.Now().Unix()}}
	r.Players[0].Total = 4
	request(t, member, "POST", h.URL+"/api/clubs", map[string]any{"op": "dissolveRoom", "id": cid, "roomId": rid}, 403)
	otherClub := request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "create", "name": "另一馆"}, 200)["id"].(string)
	request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "dissolveRoom", "id": otherClub, "roomId": rid}, 400)
	repo.fail = true
	request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "dissolveRoom", "id": cid, "roomId": rid}, 503)
	if s.rooms[rid].Status != "playing" {
		t.Fatal("failed closure did not roll back")
	}
	repo.fail = false
	request(t, owner, "POST", h.URL+"/api/clubs", map[string]any{"op": "dissolveRoom", "id": cid, "roomId": rid}, 200)
	r = s.rooms[rid]
	if r.Status != "closed" || len(r.History) != 1 || r.Players[0].Total != 4 {
		t.Fatal("closure changed completed results or left room active")
	}
	request(t, member, "POST", h.URL+"/api/command", map[string]any{"op": "join", "requestId": "closed-join", "roomId": rid}, 200)
	reloaded, err = New(Options{Repository: repo})
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.rooms[rid].Status != "closed" || len(reloaded.rooms[rid].History) != 1 {
		t.Fatal("closure not persisted")
	}
	settings["autoCreate"] = false
	request(t, owner, "POST", h.URL+"/api/clubs", settings, 200)
	before := len(s.rooms)
	for _, r := range s.rooms {
		if r.ClubID == cid {
			r.Status = "closed"
		}
	}
	s.tick(time.Now())
	if len(s.rooms) != before {
		t.Fatal("automatic creation continued after disabling")
	}
}
