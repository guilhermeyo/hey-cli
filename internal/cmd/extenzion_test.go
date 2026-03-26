package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/basecamp/hey-cli/internal/output"
)

const testExtenzionsListHTML = `
<section class="extenzion step step--showing step--full-width push_half--bottom">
    <div class="extenzion__actions">
      <a class="btn btn--x-small btn--subtle" href="/accounts/999999/domains/extenzions/100/edit">
        <span aria-hidden="true">Edit</span>
      </a>
    </div>
  <h2 class="extenzion__name hdg hdg--large pull_quarter--top flush--bottom txt--ellipsis txt--normal">
    <strong>sales</strong>@example.com
  </h2>
  <div class="extenzion-contacts">
    <div class="flex flex--wrap">
      <span class="txt--x-small push_quarter--top push_half--right flex">
        Alice
      </span>
    </div>
  </div>
</section>
`

func extenzionServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/identity.json":
			resp := map[string]any{
				"id":   111111,
				"name": "Test User",
				"primary_contact": map[string]any{
					"id":            222222,
					"account_id":    999999,
					"email_address": "test@example.com",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)

		case r.Method == "GET" && r.URL.Path == "/accounts/999999/domains/extenzions":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, testExtenzionsListHTML)

		default:
			w.WriteHeader(200)
		}
	}))
}

func runExtenzion(t *testing.T, server *httptest.Server, args ...string) (output.Response, error) {
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
	root.SetArgs(append([]string{"extenzion", "--json", "--base-url", server.URL}, args...))

	err := root.Execute()
	var resp output.Response
	if buf.Len() > 0 {
		_ = json.Unmarshal(buf.Bytes(), &resp)
	}
	return resp, err
}

func TestExtenzionList(t *testing.T) {
	server := extenzionServer(t)
	defer server.Close()

	resp, err := runExtenzion(t, server, "list")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !resp.OK {
		t.Fatal("expected ok=true")
	}
}
