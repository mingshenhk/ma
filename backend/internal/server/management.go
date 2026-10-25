package server

import (
	"haojiayuan/internal/game"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const clubHistoryRetention = 7 * 24 * time.Hour

func recentClubRounds(rounds []Round, now time.Time) []Round {
	cutoff := now.Add(-clubHistoryRetention).Unix()
	out := make([]Round, 0, len(rounds))
	for _, round := range rounds {
		if round.At >= cutoff && round.At <= now.Unix() {
			out = append(out, round)
		}
	}
	return out
}

func (s *Server) pruneExpiredClubHistory(now time.Time) error {
	for _, room := range s.rooms {
		if room.ClubID == "" {
			continue
		}
		recent := recentClubRounds(room.History, now)
		if len(recent) == len(room.History) {
			continue
		}
		previous := clone(room)
		room.History = recent
		if err := s.repo.SaveRoom(room); err != nil {
			s.rooms[room.ID] = previous
			return err
		}
	}
	return nil
}

func clubManager(c *Club, user string) bool {
	return c.Members[user] && (c.Owner == user || c.Managers[user])
}

func visibleClub(c *Club, user string) *Club {
	out := clone(c)
	if !clubManager(c, user) {
		out.Applications = nil
	}
	return out
}
func (s *Server) clubMemberBusy(c *Club, user string) bool {
	for _, r := range s.rooms {
		if r.ClubID == c.ID && r.Status != "finished" && r.Status != "closed" && seatOf(r, user) >= 0 {
			return true
		}
	}
	return false
}
func (s *Server) clubsAPI(r *http.Request, u *User) (any, error) {
	if r.Method == "GET" {
		list := []*Club{}
		for _, c := range s.accounts.Clubs {
			if c.Members[u.ID] {
				list = append(list, visibleClub(c, u.ID))
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
		return list, nil
	}
	if r.Method != "POST" {
		return nil, apiError{405, "请求方法不支持"}
	}
	var input struct {
		Op              string       `json:"op"`
		ID              string       `json:"id"`
		Name            string       `json:"name"`
		UserID          string       `json:"userId"`
		Enabled         bool         `json:"enabled"`
		RequireApproval *bool        `json:"requireApproval"`
		AutoCreate      *bool        `json:"autoCreate"`
		RoomID          string       `json:"roomId"`
		Announcement    string       `json:"announcement"`
		Template        ClubTemplate `json:"template"`
		TemplateID      string       `json:"templateId"`
	}
	if err := decode(r, &input); err != nil {
		return nil, err
	}
	before := clone(s.accounts)
	c := s.accounts.Clubs[input.ID]
	pending := false
	if input.Op == "create" {
		name := strings.TrimSpace(input.Name)
		if len([]rune(name)) < 1 || len([]rune(name)) > 24 {
			return nil, bad("名称需要1到24字")
		}
		c = &Club{ID: id(4), Name: name, Owner: u.ID, Members: map[string]bool{u.ID: true}, Managers: map[string]bool{}, Applications: map[string]int64{}, Disabled: map[string]bool{}, Templates: []ClubTemplate{}}
		if input.RequireApproval != nil {
			c.RequireApproval = *input.RequireApproval
		}
		s.accounts.Clubs[c.ID] = c
	} else {
		if c == nil {
			return nil, bad("茶馆不存在")
		}
		if input.Op == "join" {
			if !c.Members[u.ID] {
				if c.JoinClosed {
					return nil, forbidden()
				}
				if c.RequireApproval {
					if c.Applications == nil {
						c.Applications = map[string]int64{}
					}
					if c.Applications[u.ID] == 0 {
						c.Applications[u.ID] = time.Now().Unix()
					}
					pending = true
				} else {
					c.Members[u.ID] = true
					delete(c.Applications, u.ID)
				}
			}
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
				if len([]rune(input.Announcement)) > 300 {
					return nil, bad("公告最多300字")
				}
				if input.AutoCreate != nil && *input.AutoCreate {
					enabled := false
					for _, template := range c.Templates {
						enabled = enabled || template.Enabled
					}
					if !enabled {
						return nil, bad("请先添加并启用茶馆玩法")
					}
				}
				c.Name, c.JoinClosed, c.Announcement = name, input.Enabled, strings.TrimSpace(input.Announcement)
				if input.RequireApproval != nil {
					c.RequireApproval = *input.RequireApproval
				}
				if input.AutoCreate != nil {
					c.AutoCreate = *input.AutoCreate
				}
			case "dissolveRoom":
				if !clubManager(c, u.ID) {
					return nil, forbidden()
				}
				r := s.rooms[input.RoomID]
				if r == nil || r.ClubID != c.ID {
					return nil, bad("馆内房间不存在")
				}
				if r.Status != "closed" && r.Status != "finished" {
					previous := clone(r)
					r.Status, r.VoteDeadline, r.Votes = "closed", 0, nil
					r.Version++
					r.Updated = time.Now().Unix()
					if err := s.repo.SaveRoom(r); err != nil {
						s.rooms[r.ID] = previous
						s.storageError(err)
						return nil, apiError{503, "保存失败，操作未生效"}
					}
					s.broadcast(r)
				}
				return visibleClub(c, u.ID), nil
			case "approve", "reject":
				if !clubManager(c, u.ID) {
					return nil, forbidden()
				}
				if c.Applications[input.UserID] == 0 {
					return nil, bad("申请不存在或已处理")
				}
				delete(c.Applications, input.UserID)
				if input.Op == "approve" {
					c.Members[input.UserID] = true
				}
			case "template":
				if !clubManager(c, u.ID) {
					return nil, forbidden()
				}
				template := input.Template
				template.Name = strings.TrimSpace(template.Name)
				if len([]rune(template.Name)) < 1 || len([]rune(template.Name)) > 24 {
					return nil, bad("玩法名称需要1到24字")
				}
				cfg, err := game.Normalize(template.Config)
				if err != nil {
					return nil, bad(err.Error())
				}
				template.Config = cfg
				if template.ID == "" {
					if len(c.Templates) >= 30 {
						return nil, bad("最多设置30个玩法")
					}
					template.ID = id(4)
					c.Templates = append(c.Templates, template)
				} else {
					found := false
					for i, t := range c.Templates {
						if t.ID == template.ID {
							c.Templates[i] = template
							found = true
							break
						}
					}
					if !found {
						return nil, bad("玩法不存在")
					}
				}
			case "deleteTemplate":
				if !clubManager(c, u.ID) {
					return nil, forbidden()
				}
				found := false
				for i, t := range c.Templates {
					if t.ID == input.TemplateID {
						c.Templates = append(c.Templates[:i], c.Templates[i+1:]...)
						found = true
						break
					}
				}
				if !found {
					return nil, bad("玩法不存在")
				}
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
					delete(c.Disabled, input.UserID)
				} else {
					delete(c.Managers, input.UserID)
				}
			case "disable":
				if !clubManager(c, u.ID) || input.UserID == c.Owner || (c.Owner != u.ID && c.Managers[input.UserID]) {
					return nil, forbidden()
				}
				if !c.Members[input.UserID] {
					return nil, bad("成员不存在")
				}
				if c.Managers[input.UserID] {
					return nil, bad("请先取消该成员的管理员身份")
				}
				if c.Disabled == nil {
					c.Disabled = map[string]bool{}
				}
				if input.Enabled {
					c.Disabled[input.UserID] = true
				} else {
					delete(c.Disabled, input.UserID)
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
					return nil, bad("馆主不能退出，请使用解散茶馆")
				}
				if !c.Members[target] {
					return nil, bad("成员不存在")
				}
				if s.clubMemberBusy(c, target) {
					return nil, conflict("该成员仍在茶馆房间，请先离桌或解散房间")
				}
				delete(c.Members, target)
				delete(c.Managers, target)
				delete(c.Disabled, target)
			case "delete":
				if c.Owner != u.ID {
					return nil, forbidden()
				}
				for _, room := range s.rooms {
					if room.ClubID == c.ID && room.Status != "finished" && room.Status != "closed" {
						return nil, conflict("请先结束茶馆内所有房间")
					}
				}
				delete(s.accounts.Clubs, c.ID)
			default:
				return nil, bad("茶馆操作不支持")
			}
		}
	}
	if err := s.saveAccounts(before); err != nil {
		return nil, err
	}
	if pending {
		return map[string]any{"id": c.ID, "pending": true}, nil
	}
	return visibleClub(c, u.ID), nil
}

// Each enabled template keeps one completely empty waiting room. Occupied
// waiting rooms do not count; restarting reads the persisted tables first.
func (s *Server) ensureClubTables(now time.Time) {
	for _, c := range s.accounts.Clubs {
		if !c.AutoCreate {
			continue
		}
		for _, template := range c.Templates {
			if !template.Enabled {
				continue
			}
			available := false
			for _, r := range s.rooms {
				if r.ClubID != c.ID || r.TemplateID != template.ID || r.Status != "waiting" {
					continue
				}
				empty := true
				for _, p := range r.Players {
					empty = empty && p == nil
				}
				available = available || empty
			}
			if available {
				continue
			}
			cfg, err := game.Normalize(template.Config)
			if err != nil {
				s.storageError(err)
				continue
			}
			d, _ := game.Def(cfg.Game)
			roomID := ""
			for roomID == "" || s.rooms[roomID] != nil {
				roomID = strconv.Itoa(100000 + game.Random(900000))
			}
			r := &Room{ID: roomID, ClubID: c.ID, TemplateID: template.ID, Config: cfg, Players: make([]*Player, d.Players), Status: "waiting", Version: 1, Created: now.Unix(), Updated: now.Unix()}
			if err := s.repo.SaveRoom(r); err != nil {
				s.storageError(err)
				continue
			}
			s.rooms[r.ID] = r
		}
	}
}

func (s *Server) clubView(c *Club, user string) any {
	members, rooms, history, applications := []any{}, []any{}, []any{}, []any{}
	type stat struct {
		Rounds  int `json:"rounds"`
		Total   int `json:"total"`
		BigWins int `json:"bigWins"`
	}
	stats := map[string]*stat{}
	ids := []string{}
	for uid := range c.Members {
		ids = append(ids, uid)
		stats[uid] = &stat{}
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
		summary := s.summary(r).(map[string]any)
		summary["seated"] = seatOf(r, user) >= 0
		rooms = append(rooms, summary)
		recentRounds := recentClubRounds(r.History, time.Now())
		if len(recentRounds) > 0 {
			history = append(history, map[string]any{"room": s.summary(r), "players": r.Players, "rounds": recentRounds, "updated": r.Updated})
			totals := make([]int, len(r.Players))
			for _, round := range recentRounds {
				for i, score := range round.Scores {
					if i >= len(r.Players) || r.Players[i] == nil {
						continue
					}
					uid := r.Players[i].UserID
					if stats[uid] == nil {
						stats[uid] = &stat{}
					}
					stats[uid].Rounds++
					stats[uid].Total += score
					totals[i] += score
				}
			}
			if r.Status == "finished" || r.Status == "closed" {
				best := 0
				for _, v := range totals {
					if v > best {
						best = v
					}
				}
				for i, v := range totals {
					if v == best && best > 0 && r.Players[i] != nil {
						stats[r.Players[i].UserID].BigWins++
					}
				}
			}
		}
	}
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].(map[string]any)["id"].(string) < rooms[j].(map[string]any)["id"].(string)
	})
	sort.Slice(history, func(i, j int) bool {
		return history[i].(map[string]any)["updated"].(int64) > history[j].(map[string]any)["updated"].(int64)
	})
	if clubManager(c, user) {
		for uid, at := range c.Applications {
			if u := s.accounts.Users[uid]; u != nil {
				applications = append(applications, map[string]any{"user": publicUser(u), "at": at})
			}
		}
	}
	sort.Slice(applications, func(i, j int) bool {
		return applications[i].(map[string]any)["at"].(int64) < applications[j].(map[string]any)["at"].(int64)
	})
	return map[string]any{"club": visibleClub(c, user), "members": members, "rooms": rooms, "history": history, "applications": applications, "stats": stats}
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
		recent := r.History
		if r.ClubID != "" {
			recent = recentClubRounds(r.History, time.Now())
		}
		if len(recent) > 0 {
			records = append(records, map[string]any{"room": s.summary(r), "players": r.Players, "rounds": recent, "updated": r.Updated})
		}
	}
	for _, c := range s.accounts.Clubs {
		clubs = append(clubs, visibleClub(c, ""))
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

func (s *Server) externalClubRecords(r *http.Request) (any, error) {
	if r.Method != http.MethodGet {
		return nil, apiError{405, "请求方法不支持"}
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "teahouses" || parts[4] != "records" || parts[3] == "" {
		return nil, apiError{404, "接口不存在"}
	}
	club := s.accounts.Clubs[parts[3]]
	if club == nil {
		return nil, apiError{404, "茶馆不存在"}
	}
	limit, offset := 50, 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return nil, bad("limit需为1到100")
		}
		limit = parsed
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > 10000 {
			return nil, bad("offset需为0到10000")
		}
		offset = parsed
	}
	now := time.Now()
	gameFilter, memberFilter := r.URL.Query().Get("game"), r.URL.Query().Get("memberId")
	dateFilter := r.URL.Query().Get("date")
	dayStart, dayEnd := int64(0), int64(0)
	if dateFilter != "" {
		chinaTime := now.In(time.FixedZone("Asia/Shanghai", 8*60*60))
		var selectedDay time.Time
		if dateFilter == "today" {
			selectedDay = chinaTime
		} else {
			parsed, err := time.ParseInLocation("2006-01-02", dateFilter, chinaTime.Location())
			if err != nil || parsed.Format("2006-01-02") != dateFilter {
				return nil, bad("date需为today或YYYY-MM-DD")
			}
			selectedDay = parsed
		}
		selectedDay = time.Date(selectedDay.Year(), selectedDay.Month(), selectedDay.Day(), 0, 0, 0, 0, selectedDay.Location())
		today := time.Date(chinaTime.Year(), chinaTime.Month(), chinaTime.Day(), 0, 0, 0, 0, chinaTime.Location())
		if selectedDay.After(today) {
			return nil, bad("date不能晚于今天")
		}
		dayStart, dayEnd = selectedDay.Unix(), selectedDay.AddDate(0, 0, 1).Unix()
	}
	type record struct {
		value   map[string]any
		updated int64
		roomID  string
	}
	all := []record{}
	for _, room := range s.rooms {
		if room.ClubID != club.ID || (gameFilter != "" && room.Config.Game != gameFilter) {
			continue
		}
		if memberFilter != "" {
			found := false
			for _, player := range room.Players {
				found = found || (player != nil && player.UserID == memberFilter)
			}
			if !found {
				continue
			}
		}
		rounds := recentClubRounds(room.History, now)
		if dayStart != 0 {
			dayRounds := make([]Round, 0, len(rounds))
			for _, round := range rounds {
				if round.At >= dayStart && round.At < dayEnd {
					dayRounds = append(dayRounds, round)
				}
			}
			rounds = dayRounds
		}
		if len(rounds) == 0 {
			continue
		}
		latest := int64(0)
		for _, round := range rounds {
			if round.At > latest {
				latest = round.At
			}
		}
		all = append(all, record{map[string]any{"room": s.summary(room), "players": room.Players, "rounds": rounds, "updated": latest}, latest, room.ID})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].updated == all[j].updated {
			return all[i].roomID > all[j].roomID
		}
		return all[i].updated > all[j].updated
	})
	total := len(all)
	if offset >= total {
		all = []record{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		all = all[offset:end]
	}
	results := make([]map[string]any, 0, len(all))
	for _, item := range all {
		results = append(results, item.value)
	}
	return map[string]any{"teaHouse": map[string]any{"id": club.ID, "name": club.Name}, "retentionDays": 7, "generatedAt": now.Unix(), "total": total, "limit": limit, "offset": offset, "records": results}, nil
}
