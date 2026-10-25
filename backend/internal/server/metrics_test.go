package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsAuthorizationAndCounters(t *testing.T) {
	s, err := New(Options{Repository: JSONRepository{Dir: t.TempDir()}, AdminKey: "test-metrics-key"})
	if err != nil {
		t.Fatal(err)
	}
	h := httptest.NewServer(s.Handler())
	defer h.Close()
	c := client()
	register(t, c, h.URL, "metrics-user")
	request(t, c, "POST", h.URL+"/api/command", map[string]any{"op": "create", "requestId": "metrics-room", "config": map[string]any{"game": "sichuan"}}, 200)
	request(t, c, "POST", h.URL+"/api/command", map[string]any{"op": "create"}, 400)
	for _, tc := range []struct {
		method, key string
		want        int
	}{{"GET", "", 403}, {"GET", "wrong", 403}, {"POST", "test-metrics-key", 405}, {"GET", "test-metrics-key", 200}} {
		r, _ := http.NewRequest(tc.method, h.URL+"/metrics", nil)
		r.Header.Set("X-Admin-Key", tc.key)
		resp, e := c.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Fatalf("%s metrics: %d want %d", tc.method, resp.StatusCode, tc.want)
		}
		if tc.want != 200 {
			continue
		}
		if !strings.Contains(resp.Header.Get("Content-Type"), "text/plain") {
			t.Fatal("wrong exposition content type")
		}
		for _, line := range []string{"hjy_commands_total 2\n", "hjy_api_duration_seconds_count 3\n", "hjy_api_duration_seconds_errors_total 1\n", "hjy_command_duration_seconds_count 2\n", "hjy_command_duration_seconds_errors_total 1\n", "hjy_command_duration_seconds_bucket{le=\"+Inf\"} 2\n", "hjy_rooms_by_game{game=\"sichuan\"} 1\n", "hjy_online_accounts 0\n", "process_cpu_seconds_total ", "hjy_heap_alloc_bytes "} {
			if !strings.Contains(string(b), line) {
				t.Fatalf("missing metric %q: %s", line, b)
			}
		}
	}
	// Empty configured key must not accidentally open the endpoint.
	s.admin = ""
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != 403 {
		t.Fatal("unconfigured metrics must stay closed")
	}
}
