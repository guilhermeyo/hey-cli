package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/basecamp/hey-cli/internal/output"
)

func eventServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/calendars.json":
			resp := map[string]any{
				"calendars": []map[string]any{
					{"calendar": map[string]any{"id": 1, "name": "Personal", "personal": true}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case r.Method == "GET" && r.URL.Path == "/calendars/1/recordings":
			resp := map[string]any{
				"Calendar::Event": []map[string]any{
					{
						"id":        100,
						"title":     "Test Event",
						"all_day":   false,
						"starts_at": "2026-04-06T10:00:00Z",
						"ends_at":   "2026-04-06T11:00:00Z",
						"type":      "Calendar::Event",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		default:
			w.WriteHeader(200)
		}
	}))
}

func runEventList(t *testing.T, server *httptest.Server, args ...string) (output.Response, error) {
	t.Helper()
	t.Setenv("HEY_TOKEN", "test-token")
	t.Setenv("HEY_NO_KEYRING", "1")
	t.Setenv("HEY_BASE_URL", "")
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"event", "list", "--json", "--base-url", server.URL}, args...))

	err := root.Execute()
	var resp output.Response
	if buf.Len() > 0 {
		_ = json.Unmarshal(buf.Bytes(), &resp)
	}
	return resp, err
}

func TestEventList(t *testing.T) {
	server := eventServer(t)
	defer server.Close()

	resp, err := runEventList(t, server)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.OK {
		t.Fatal("expected ok=true")
	}
}
