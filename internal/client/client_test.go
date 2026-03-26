package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/basecamp/hey-cli/internal/apierr"
	"github.com/basecamp/hey-cli/internal/auth"
)

func testClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	t.Setenv("HEY_TOKEN", "test-token")
	t.Setenv("HEY_NO_KEYRING", "1")
	mgr := auth.NewManager(server.URL, server.Client(), t.TempDir())
	c := New(server.URL, mgr)
	c.HTTPClient = server.Client()
	c.SleepFunc = func(time.Duration) {}
	return c
}

func TestResponseError(t *testing.T) {
	tests := []struct {
		status    int
		wantCode  string
		retryable bool
	}{
		{401, "auth", false},
		{403, "forbidden", false},
		{404, "not_found", false},
		{429, "rate_limit", true},
		{500, "api", true},
		{422, "api", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status == 429 {
					w.Header().Set("Retry-After", "1")
				}
				w.WriteHeader(tt.status)
				fmt.Fprint(w, "error body")
			}))
			defer server.Close()

			c := testClient(t, server)
			_, err := c.Get("/test.json")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			oerr := apierr.AsError(err)
			if oerr.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", oerr.Code, tt.wantCode)
			}
			if oerr.Retryable != tt.retryable {
				t.Errorf("retryable = %v, want %v", oerr.Retryable, tt.retryable)
			}
		})
	}
}

func TestDoWithRetrySuccess(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(500)
			fmt.Fprint(w, "server error")
			return
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	c := testClient(t, server)
	data, err := c.Get("/test.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("body = %q, want %q", string(data), `{"ok":true}`)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 calls, got %d", got)
	}
}

func TestDoWithRetryExhausted(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(500)
		fmt.Fprint(w, "server error")
	}))
	defer server.Close()

	c := testClient(t, server)
	_, err := c.Get("/test.json")
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if got := calls.Load(); got != int32(maxRetries) {
		t.Errorf("expected %d calls, got %d", maxRetries, got)
	}
}

func TestPostForm_302(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %s, want application/x-www-form-urlencoded", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.PostFormValue("calendar_event[summary]"); got != "Test" {
			t.Errorf("summary = %q, want %q", got, "Test")
		}
		w.Header().Set("Location", "https://app.hey.com/calendar/events/12345")
		w.WriteHeader(302)
	}))
	defer server.Close()

	c := testClient(t, server)
	resp, err := c.PostForm("/calendar/events", url.Values{
		"calendar_event[summary]": {"Test"},
	})
	if err != nil {
		t.Fatalf("PostForm: %v", err)
	}
	if resp.Location != "https://app.hey.com/calendar/events/12345" {
		t.Errorf("location = %q, want %q", resp.Location, "https://app.hey.com/calendar/events/12345")
	}
	id, err := resp.ExtractID()
	if err != nil {
		t.Fatalf("ExtractID: %v", err)
	}
	if id != 12345 {
		t.Errorf("id = %d, want %d", id, 12345)
	}
}

func TestPatchForm_302(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		w.Header().Set("Location", "https://app.hey.com/calendar/days/2026-04-06")
		w.WriteHeader(302)
	}))
	defer server.Close()

	c := testClient(t, server)
	err := c.PatchForm("/calendar/events/12345", url.Values{
		"calendar_event[summary]": {"Updated"},
	})
	if err != nil {
		t.Fatalf("PatchForm: %v", err)
	}
}

func TestDelete_302(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.Header().Set("Location", "https://app.hey.com/calendar/days/2026-04-06")
		w.WriteHeader(302)
	}))
	defer server.Close()

	c := testClient(t, server)
	err := c.Delete("/calendar/events/12345")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestPostForm_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		fmt.Fprint(w, "validation error")
	}))
	defer server.Close()

	c := testClient(t, server)
	_, err := c.PostForm("/calendar/events", url.Values{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	oerr := apierr.AsError(err)
	if oerr.Code != "api" {
		t.Errorf("code = %q, want %q", oerr.Code, "api")
	}
}

func TestRateLimitRetryAfter(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(429)
			fmt.Fprint(w, "rate limited")
			return
		}
		w.WriteHeader(200)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	var sleptDuration time.Duration
	c := testClient(t, server)
	c.SleepFunc = func(d time.Duration) {
		sleptDuration = d
	}

	data, err := c.Get("/test.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("body = %q, want %q", string(data), `{"ok":true}`)
	}
	if sleptDuration != 2*time.Second {
		t.Errorf("sleep duration = %v, want %v", sleptDuration, 2*time.Second)
	}
}
