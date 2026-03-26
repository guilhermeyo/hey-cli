package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testExtenzionsHTML = `
<section class="extenzion step step--showing step--full-width push_half--bottom">
    <div class="extenzion__actions">
      <a class="btn btn--x-small btn--subtle" href="/accounts/999999/domains/extenzions/300001/edit">
        <span aria-hidden="true">Edit</span><span class="u-for-screen-reader">Edit sales@example.com</span>
      </a>
    </div>
  <h2 class="extenzion__name hdg hdg--large pull_quarter--top flush--bottom txt--ellipsis txt--normal">
    <strong>sales</strong>@example.com
  </h2>
  <div class="extenzion-contacts">
    <div class="flex flex--wrap">
      <span class="txt--x-small push_quarter--top push_half--right flex">
        <img src="/avatar.png" alt="Alice" title="Alice &lt;alice@example.com&gt;" width="50" height="50">
        Alice
      </span>
      <span class="txt--x-small push_quarter--top push_half--right flex">
        <img src="/avatar2.png" alt="Bob" title="Bob &lt;bob@example.com&gt;" width="50" height="50">
        Bob
      </span>
    </div>
  </div>
</section>
<section class="extenzion step step--showing step--full-width push_half--bottom">
    <div class="extenzion__actions">
      <a class="btn btn--x-small btn--subtle" href="/accounts/999999/domains/extenzions/300002/edit">
        <span aria-hidden="true">Edit</span><span class="u-for-screen-reader">Edit support@example.com</span>
      </a>
    </div>
  <h2 class="extenzion__name hdg hdg--large pull_quarter--top flush--bottom txt--ellipsis txt--normal">
    <strong>support</strong>@example.com
  </h2>
  <div class="extenzion-contacts">
    <div class="flex flex--wrap">
      <span class="txt--x-small push_quarter--top push_half--right flex">
        <img src="/avatar.png" alt="Alice" title="Alice &lt;alice@example.com&gt;" width="50" height="50">
        Alice
      </span>
    </div>
  </div>
</section>
`

func TestListExtenzions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/accounts/999999/domains/extenzions":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, testExtenzionsHTML)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	c := testClient(t, server)
	exts, err := c.ListExtenzions(999999)
	if err != nil {
		t.Fatalf("ListExtenzions: %v", err)
	}
	if len(exts) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(exts))
	}

	if exts[0].ID != 300001 {
		t.Errorf("ext[0].ID = %d, want 300001", exts[0].ID)
	}
	if exts[0].Name != "sales" {
		t.Errorf("ext[0].Name = %q, want %q", exts[0].Name, "sales")
	}
	if exts[0].Email != "sales@example.com" {
		t.Errorf("ext[0].Email = %q, want %q", exts[0].Email, "sales@example.com")
	}
	if len(exts[0].Members) != 2 {
		t.Fatalf("ext[0].Members = %d, want 2", len(exts[0].Members))
	}
	if exts[0].Members[0] != "Alice" {
		t.Errorf("ext[0].Members[0] = %q, want %q", exts[0].Members[0], "Alice")
	}

	if exts[1].ID != 300002 {
		t.Errorf("ext[1].ID = %d, want 300002", exts[1].ID)
	}
	if exts[1].Name != "support" {
		t.Errorf("ext[1].Name = %q, want %q", exts[1].Name, "support")
	}
}

func TestListExtenzions_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><div class="sheet"></div></body></html>`)
	}))
	defer server.Close()

	c := testClient(t, server)
	exts, err := c.ListExtenzions(999999)
	if err != nil {
		t.Fatalf("ListExtenzions: %v", err)
	}
	if len(exts) != 0 {
		t.Errorf("expected 0 extensions, got %d", len(exts))
	}
}

func TestCreateExtenzion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.PostFormValue("extenzion[name]"); got != "sales" {
			t.Errorf("name = %q, want %q", got, "sales")
		}
		if got := r.PostFormValue("extenzion[membership]"); got != "internal" {
			t.Errorf("membership = %q, want %q", got, "internal")
		}
		members := r.PostForm["extenzion[members][]"]
		if len(members) != 1 || members[0] != "alice@example.com" {
			t.Errorf("members = %v, want [alice@example.com]", members)
		}
		w.Header().Set("Location", "https://app.hey.com/accounts/999999/domains/extenzions")
		w.WriteHeader(302)
	}))
	defer server.Close()

	c := testClient(t, server)
	_, err := c.CreateExtenzion(999999, "sales", []string{"alice@example.com"})
	if err != nil {
		t.Fatalf("CreateExtenzion: %v", err)
	}
}

func TestUpdateExtenzion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/accounts/999999/domains/extenzions/12345" {
			t.Errorf("path = %s, want /accounts/999999/domains/extenzions/12345", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.PostFormValue("_method"); got != "patch" {
			t.Errorf("_method = %q, want %q", got, "patch")
		}
		if got := r.PostFormValue("extenzion[name]"); got != "new-name" {
			t.Errorf("name = %q, want %q", got, "new-name")
		}
		w.Header().Set("Location", "https://app.hey.com/accounts/999999/domains/extenzions")
		w.WriteHeader(302)
	}))
	defer server.Close()

	c := testClient(t, server)
	err := c.UpdateExtenzion(999999, 12345, "new-name", []string{"alice@example.com"})
	if err != nil {
		t.Fatalf("UpdateExtenzion: %v", err)
	}
}

func TestDeleteExtenzion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.PostFormValue("_method"); got != "delete" {
			t.Errorf("_method = %q, want %q", got, "delete")
		}
		w.Header().Set("Location", "https://app.hey.com/accounts/999999/domains/extenzions")
		w.WriteHeader(302)
	}))
	defer server.Close()

	c := testClient(t, server)
	err := c.DeleteExtenzion(999999, 12345)
	if err != nil {
		t.Fatalf("DeleteExtenzion: %v", err)
	}
}
