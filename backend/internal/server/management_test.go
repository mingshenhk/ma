package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
		v := request(t, c, "POST", h.URL+"/api/session", map[string]any{"name": []string{"圈主", "管理", "成员", "外部"}[i]}, 200)
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
	room := request(t, member, "POST", h.URL+"/api/command", map[string]any{"op": "create", "requestId": "club-room", "clubId": cid, "config": map[string]any{"game": "hongzhong"}}, 200)
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
		for _, secret := range []string{"recoveryHash", "sessions", "\"wall\"", "\"state\""} {
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
