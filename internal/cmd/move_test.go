package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/basecamp/hey-cli/internal/output"
)

func moveServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/boxes.json":
			resp := []map[string]any{
				{"id": 1, "kind": "imbox", "name": "Imbox"},
				{"id": 2, "kind": "feedbox", "name": "The Feed"},
				{"id": 3, "kind": "trailbox", "name": "Paper Trail"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case r.Method == "GET" && r.URL.Path == "/topics/999":
			resp := map[string]any{
				"id": 999,
				"creator": map[string]any{
					"id":            555,
					"name":          "Test Sender",
					"email_address": "test@example.com",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case r.Method == "POST" && r.URL.Path == "/boxes/2/designations":
			w.Header().Set("Location", "https://app.hey.com/contacts/555")
			w.WriteHeader(302)

		default:
			w.WriteHeader(200)
		}
	}))
}

func runMove(t *testing.T, server *httptest.Server, args ...string) (output.Response, error) {
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
	root.SetArgs(append([]string{"move", "--json", "--base-url", server.URL}, args...))

	err := root.Execute()
	var resp output.Response
	if buf.Len() > 0 {
		_ = json.Unmarshal(buf.Bytes(), &resp)
	}
	return resp, err
}

func TestMove_ByTopicID(t *testing.T) {
	server := moveServer(t)
	defer server.Close()

	resp, err := runMove(t, server, "999", "--box", "feedbox", "--yes")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.OK {
		t.Fatal("expected ok=true")
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("data type = %T, want map[string]any", resp.Data)
	}
	if id, ok := data["contact_id"].(float64); !ok || int64(id) != 555 {
		t.Errorf("contact_id = %v, want 555", data["contact_id"])
	}
}

func TestMove_ByContactID(t *testing.T) {
	server := moveServer(t)
	defer server.Close()

	resp, err := runMove(t, server, "--contact", "555", "--box", "feedbox", "--yes")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.OK {
		t.Fatal("expected ok=true")
	}
}

func TestMove_InvalidBox(t *testing.T) {
	server := moveServer(t)
	defer server.Close()

	_, err := runMove(t, server, "--contact", "555", "--box", "invalid", "--yes")
	if err == nil {
		t.Fatal("expected error for invalid box")
	}
}

func TestMove_MutuallyExclusive(t *testing.T) {
	server := moveServer(t)
	defer server.Close()

	_, err := runMove(t, server, "999", "--contact", "555", "--box", "feedbox", "--yes")
	if err == nil {
		t.Fatal("expected error for topic ID + --contact together")
	}
}
