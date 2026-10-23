package server

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"haojiayuan/internal/game"
)

var durationBounds = [...]float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 5}

type durationStats struct {
	count, errors uint64
	seconds       float64
	buckets       [len(durationBounds)]uint64
}

func (d *durationStats) observe(seconds float64, failed bool) {
	d.count++
	d.seconds += seconds
	if failed {
		d.errors++
	}
	for i, bound := range durationBounds {
		if seconds <= bound {
			d.buckets[i]++
		}
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(b)
}

// Only API responses are wrapped; WebSocket upgrade retains the original writer.
func (s *Server) observeAPI(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	out := &statusWriter{ResponseWriter: w}
	defer func() {
		s.mu.Lock()
		s.apiStats.observe(time.Since(start).Seconds(), out.status >= 400 || out.status == 0)
		s.mu.Unlock()
	}()
	s.api(out, r)
}

func writeDuration(b *strings.Builder, name string, d durationStats) {
	fmt.Fprintf(b, "# TYPE %s histogram\n", name)
	for i, bound := range durationBounds {
		fmt.Fprintf(b, "%s_bucket{le=\"%g\"} %d\n", name, bound, d.buckets[i])
	}
	fmt.Fprintf(b, "%s_bucket{le=\"+Inf\"} %d\n%s_sum %g\n%s_count %d\n", name, d.count, name, d.seconds, name, d.count)
	fmt.Fprintf(b, "# TYPE %s_errors_total counter\n%s_errors_total %d\n", name, name, d.errors)
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		jsonOut(w, 405, map[string]any{"error": "仅支持 GET"})
		return
	}
	if s.admin == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Admin-Key")), []byte(s.admin)) != 1 {
		jsonOut(w, 403, map[string]any{"error": "无权查看指标"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var b strings.Builder
	gauge := func(name string, v any) { fmt.Fprintf(&b, "# TYPE %s gauge\n%s %v\n", name, name, v) }
	counter := func(name string, v uint64) { fmt.Fprintf(&b, "# TYPE %s counter\n%s %d\n", name, name, v) }
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	gauge("hjy_uptime_seconds", time.Since(s.started).Seconds())
	gauge("hjy_heap_alloc_bytes", mem.HeapAlloc)
	gauge("hjy_go_sys_bytes", mem.Sys)
	gauge("hjy_goroutines", runtime.NumGoroutine())
	if cpu, err := processCPUSeconds(); err == nil {
		fmt.Fprintf(&b, "# TYPE process_cpu_seconds_total counter\nprocess_cpu_seconds_total %g\n", cpu)
	}
	gauge("hjy_websocket_connections", len(s.peers))
	users := map[string]bool{}
	for p := range s.peers {
		users[p.user] = true
	}
	gauge("hjy_online_accounts", len(users))
	active := 0
	for _, r := range s.rooms {
		if r.Status != "closed" && r.Status != "finished" {
			active++
		}
	}
	gauge("hjy_rooms_active", active)
	counter("hjy_http_requests_total", s.requests)
	counter("hjy_commands_total", s.commands)
	counter("hjy_storage_errors_total", s.failures)
	for _, def := range game.Definitions {
		online := map[string]bool{}
		rooms := 0
		for p := range s.peers {
			if r := s.rooms[p.room]; r != nil && r.Config.Game == def.Key {
				online[p.user] = true
			}
		}
		for _, r := range s.rooms {
			if r.Config.Game == def.Key && r.Status != "closed" && r.Status != "finished" {
				rooms++
			}
		}
		fmt.Fprintf(&b, "hjy_online_by_game{game=\"%s\"} %d\nhjy_rooms_by_game{game=\"%s\"} %d\n", def.Key, len(online), def.Key, rooms)
	}
	writeDuration(&b, "hjy_api_duration_seconds", s.apiStats)
	writeDuration(&b, "hjy_command_duration_seconds", s.commandStats)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}
