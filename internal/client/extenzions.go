package client

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Extenzion represents a HEY email extension.
type Extenzion struct {
	ID      int64    `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Members []string `json:"members"`
}

// ListExtenzions fetches and parses the extensions list from HTML.
func (c *Client) ListExtenzions(accountID int64) ([]Extenzion, error) {
	path := fmt.Sprintf("/accounts/%d/domains/extenzions", accountID)
	data, err := c.GetHTML(path)
	if err != nil {
		return nil, err
	}
	return parseExtenzionsHTML(string(data))
}

// CreateExtenzion creates a new email extension.
func (c *Client) CreateExtenzion(accountID int64, name string, members []string) (*RedirectResponse, error) {
	values := url.Values{}
	values.Set("extenzion[name]", name)
	values.Set("extenzion[membership]", "internal")
	values.Set("commit", "Add this extension")
	for _, m := range members {
		values.Add("extenzion[members][]", m)
	}
	return c.PostForm(fmt.Sprintf("/accounts/%d/domains/extenzions", accountID), values)
}

// UpdateExtenzion updates an existing extension using Rails _method override.
func (c *Client) UpdateExtenzion(accountID, extID int64, name string, members []string) error {
	values := url.Values{}
	values.Set("_method", "patch")
	values.Set("extenzion[name]", name)
	values.Set("commit", "Save changes")
	for _, m := range members {
		values.Add("extenzion[members][]", m)
	}
	_, err := c.PostForm(fmt.Sprintf("/accounts/%d/domains/extenzions/%d", accountID, extID), values)
	return err
}

// DeleteExtenzion deletes an extension using Rails _method override.
func (c *Client) DeleteExtenzion(accountID, extID int64) error {
	values := url.Values{
		"_method": {"delete"},
	}
	_, err := c.PostForm(fmt.Sprintf("/accounts/%d/domains/extenzions/%d", accountID, extID), values)
	return err
}

var extenzionIDRegex = regexp.MustCompile(`/extenzions/(\d+)/edit`)

// parseExtenzionsHTML parses the extensions list HTML page.
func parseExtenzionsHTML(body string) ([]Extenzion, error) {
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse extenzions HTML: %w", err)
	}

	var exts []Extenzion
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "section" && hasClass(n, "extenzion") {
			if ext, ok := parseExtenzionSection(n); ok {
				exts = append(exts, ext)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return exts, nil
}

func parseExtenzionSection(section *html.Node) (Extenzion, bool) {
	var ext Extenzion

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Extract ID from edit link
			if n.Data == "a" {
				href := getAttr(n, "href")
				if m := extenzionIDRegex.FindStringSubmatch(href); m != nil {
					ext.ID, _ = strconv.ParseInt(m[1], 10, 64)
				}
			}

			// Extract name and email from h2.extenzion__name
			if n.Data == "h2" && hasClass(n, "extenzion__name") {
				ext.Email = strings.TrimSpace(textContent(n))
				// Name is inside <strong>
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && c.Data == "strong" {
						ext.Name = strings.TrimSpace(textContent(c))
					}
				}
			}

			// Extract members from extenzion-contacts div
			if n.Data == "div" && hasClass(n, "extenzion-contacts") {
				ext.Members = parseMembers(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(section)

	return ext, ext.ID != 0
}

func parseMembers(contacts *html.Node) []string {
	var members []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "txt--x-small") {
			name := strings.TrimSpace(textContent(n))
			if name != "" {
				members = append(members, name)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(contacts)
	return members
}

func hasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
