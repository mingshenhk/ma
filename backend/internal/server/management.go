package server

import (
	"net/http"
	"runtime"
	"sort"
	"strings"
)

func clubManager(c *Club, user string) bool {
	return c.Members[user] && (c.Owner == user || c.Managers[user])
}

func (s *Server) clubsAPI(r *http.Request, u *User) (any, error) {
	if r.Method == "GET" {
		list := []*Club{}
		for _, c := range s.accounts.Clubs {
			if c.Members[u.ID] {
				list = append(list, c)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
		return list, nil
	}
	if r.Method != "POST" {
		return nil, apiError{405, "请求方法不支持"}
	}
	var input struct {
		Op      string `json:"op"`
		ID      string `json:"id"`
		Name    string `json:"name"`
		UserID  string `json:"userId"`
		Enabled bool   `json:"enabled"`
	}
	if err := decode(r, &input); err != nil {
		return nil, err
	}
	before := clone(s.accounts)
	c := s.accounts.Clubs[input.ID]
	if input.Op == "create" {
		name := strings.TrimSpace(input.Name)
		if len([]rune(name)) < 1 || len([]rune(name)) > 24 {
			return nil, bad("名称需要1到24字")
		}
		c = &Club{ID: id(4), Name: name, Owner: u.ID, Members: map[string]bool{u.ID: true}, Managers: map[string]bool{}}
		s.accounts.Clubs[c.ID] = c
	} else {
		if c == nil {
			return nil, bad("亲友圈不存在")
		}
		if input.Op == "join" {
			if c.JoinClosed && !c.Members[u.ID] {
				return nil, forbidden()
			}
			c.Members[u.ID] = true
		} else {
			if !c.Members[u.ID] {
				return nil, forbidden()
			}
			switch input.Op {
			case "settings":
				if !clubManager(c, u.ID) {
					return nil, forbidden()
				}
				name := strings.TrimSpace(input.Name)
				if len([]rune(name)) < 1 || len([]rune(name)) > 24 {
					return nil, bad("名称需要1到24字")
				}
				c.Name, c.JoinClosed = name, input.Enabled
			case "manager":
				if c.Owner != u.ID {
					return nil, forbidden()
				}
				if !c.Members[input.UserID] || input.UserID == c.Owner {
					return nil, bad("请选择普通成员")
				}
				if c.Managers == nil {
					c.Managers = map[string]bool{}
				}
				if input.Enabled {
					c.Managers[input.UserID] = true
				} else {
					delete(c.Managers, input.UserID)
				}
			case "leave", "remove":
				target := u.ID
				if input.Op == "remove" {
					if !clubManager(c, u.ID) {
						return nil, forbidden()
					}
					target = input.UserID
					if c.Owner != u.ID && c.Managers[target] {
						return nil, forbidden()
					}
				}
				if target == c.Owner {
					return nil, bad("圈主不能退出，请保留圈主管理账号")
				}
				if !c.Members[target] {
					return nil, bad("成员不存在")
				}
				for _, room := range s.rooms {
					if room.ClubID == c.ID && room.Status != "finished" && room.Status != "closed" && seatOf(room, target) >= 0 {
						return nil, conflict("该成员仍在圈内房间，请先离桌或解散房间")
					}
				}
				delete(c.Members, target)
				delete(c.Managers, target)
			default:
				return nil, bad("亲友圈操作不支持")
			}
		}
	}
	if err := s.saveAccounts(before); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Server) clubView(c *Club) any {
	members, rooms, history := []any{}, []any{}, []any{}
	ids := []string{}
	for uid := range c.Members {
		ids = append(ids, uid)
	}
	sort.Strings(ids)
	for _, uid := range ids {
		if u := s.accounts.Users[uid]; u != nil {
			members = append(members, publicUser(u))
		}
	}
	for _, r := range s.rooms {
		if r.ClubID != c.ID {
			continue
		}
		rooms = append(rooms, s.summary(r))
		if len(r.History) > 0 {
			history = append(history, map[string]any{"room": s.summary(r), "players": r.Players, "rounds": r.History, "updated": r.Updated})
		}
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].(map[string]any)["updated"].(int64) > history[j].(map[string]any)["updated"].(int64)
	})
	return map[string]any{"club": c, "members": members, "rooms": rooms, "history": history}
}

// Administration exposes account profiles and completed results, never session or live game secrets.
func (s *Server) adminView() any {
	users, rooms, clubs, records := []any{}, []any{}, []any{}, []any{}
	online := map[string]map[string]bool{}
	for _, u := range s.accounts.Users {
		users = append(users, publicUser(u))
	}
	for _, r := range s.rooms {
		rooms = append(rooms, map[string]any{"room": s.summary(r), "owner": r.Owner, "players": r.Players, "updated": r.Updated})
		if len(r.History) > 0 {
			records = append(records, map[string]any{"room": s.summary(r), "players": r.Players, "rounds": r.History, "updated": r.Updated})
		}
	}
	for _, c := range s.accounts.Clubs {
		clubs = append(clubs, c)
	}
	for p := range s.peers {
		if r := s.rooms[p.room]; r != nil {
			if online[r.Config.Game] == nil {
				online[r.Config.Game] = map[string]bool{}
			}
			online[r.Config.Game][p.user] = true
		}
	}
	counts := map[string]int{}
	for key, users := range online {
		counts[key] = len(users)
	}
	return map[string]any{"users": len(users), "rooms": len(rooms), "connections": len(s.peers), "requests": s.requests, "commands": s.commands, "storageErrors": s.failures, "lastStorageError": s.lastError, "reports": s.accounts.Reports, "userList": users, "roomList": rooms, "clubList": clubs, "records": records, "onlineByGame": counts, "service": map[string]any{"storage": "json-single-writer", "go": runtime.Version(), "goroutines": runtime.NumGoroutine()}}
}
