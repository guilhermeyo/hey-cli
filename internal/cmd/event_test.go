package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func eventCreateServer(t *testing.T) *httptest.Server {
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

		case r.Method == "POST" && r.URL.Path == "/calendar/events":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Location", "https://app.hey.com/calendar/events/99999")
			w.WriteHeader(302)

		default:
			w.WriteHeader(200)
		}
	}))
}

func runEventCreate(t *testing.T, server *httptest.Server, args ...string) (output.Response, error) {
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
	root.SetArgs(append([]string{"event", "create", "--json", "--base-url", server.URL}, args...))

	err := root.Execute()
	var resp output.Response
	if buf.Len() > 0 {
		_ = json.Unmarshal(buf.Bytes(), &resp)
	}
	return resp, err
}

func TestEventCreate(t *testing.T) {
	server := eventCreateServer(t)
	defer server.Close()

	resp, err := runEventCreate(t, server, "Meeting", "--date", "2026-04-06", "--start", "10:00", "--end", "11:00")
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
	if id, ok := data["id"].(float64); !ok || int64(id) != 99999 {
		t.Errorf("id = %v, want 99999", data["id"])
	}
}

func TestEventCreate_AllDay(t *testing.T) {
	server := eventCreateServer(t)
	defer server.Close()

	resp, err := runEventCreate(t, server, "Holiday", "--date", "2026-04-06", "--all-day")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.OK {
		t.Fatal("expected ok=true")
	}
}

func TestEventCreate_WithReminders(t *testing.T) {
	var capturedForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/calendars.json":
			resp := map[string]any{
				"calendars": []map[string]any{
					{"calendar": map[string]any{"id": 1, "name": "Personal", "personal": true}},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		case r.Method == "POST" && r.URL.Path == "/calendar/events":
			r.ParseForm()
			capturedForm = r.PostForm
			w.Header().Set("Location", "https://app.hey.com/calendar/events/99999")
			w.WriteHeader(302)
		default:
			w.WriteHeader(200)
		}
	}))
	defer server.Close()

	_, err := runEventCreate(t, server, "Meeting", "--date", "2026-04-06", "--start", "10:00", "--end", "11:00", "--reminder", "30m", "--reminder", "1d")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	reminders := capturedForm["timed_reminder_durations[]"]
	if len(reminders) != 2 {
		t.Fatalf("expected 2 reminders, got %d", len(reminders))
	}
	if reminders[0] != "1800" {
		t.Errorf("reminder[0] = %q, want %q", reminders[0], "1800")
	}
	if reminders[1] != "86400" {
		t.Errorf("reminder[1] = %q, want %q", reminders[1], "86400")
	}
}

func TestEventCreate_MissingDate(t *testing.T) {
	server := eventCreateServer(t)
	defer server.Close()

	_, err := runEventCreate(t, server, "Meeting", "--start", "10:00", "--end", "11:00")
	if err == nil {
		t.Fatal("expected error for missing --date")
	}
}

func TestEventCreate_MissingTimeWithoutAllDay(t *testing.T) {
	server := eventCreateServer(t)
	defer server.Close()

	_, err := runEventCreate(t, server, "Meeting", "--date", "2026-04-06")
	if err == nil {
		t.Fatal("expected error for missing --start/--end without --all-day")
	}
}

func eventDeleteServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "DELETE" && r.URL.Path == "/calendar/events/100":
			w.Header().Set("Location", "https://app.hey.com/calendar/days/2026-04-06")
			w.WriteHeader(302)
		default:
			w.WriteHeader(200)
		}
	}))
}

func runEventDelete(t *testing.T, server *httptest.Server, args ...string) (output.Response, error) {
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
	root.SetArgs(append([]string{"event", "delete", "--json", "--base-url", server.URL}, args...))

	err := root.Execute()
	var resp output.Response
	if buf.Len() > 0 {
		_ = json.Unmarshal(buf.Bytes(), &resp)
	}
	return resp, err
}

func TestEventDelete(t *testing.T) {
	server := eventDeleteServer(t)
	defer server.Close()

	resp, err := runEventDelete(t, server, "100")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.OK {
		t.Fatal("expected ok=true")
	}
}
