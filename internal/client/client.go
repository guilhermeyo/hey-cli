package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/basecamp/hey-cli/internal/apierr"
	"github.com/basecamp/hey-cli/internal/auth"
	"github.com/basecamp/hey-cli/internal/version"
)

const maxRetries = 3

type Client struct {
	BaseURL    string
	AuthMgr    *auth.Manager
	HTTPClient *http.Client
	Logger     io.Writer
	SleepFunc  func(time.Duration)

	requestCount atomic.Int64
	totalLatency atomic.Int64 // nanoseconds
}

func New(baseURL string, authMgr *auth.Manager) *Client {
	return &Client{
		BaseURL: baseURL,
		AuthMgr: authMgr,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		SleepFunc: time.Sleep,
	}
}

func (c *Client) RequestCount() int           { return int(c.requestCount.Load()) }
func (c *Client) TotalLatency() time.Duration { return time.Duration(c.totalLatency.Load()) }

func (c *Client) doRequestAccept(method, path string, body io.Reader, contentType, accept string) (*http.Response, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	reqURL := base + path

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, apierr.ErrAPI(0, fmt.Sprintf("could not create request: %v", err))
	}

	if err = c.AuthMgr.AuthenticateRequest(ctx, req); err != nil {
		return nil, apierr.ErrAuth(fmt.Sprintf("authentication failed: %v", err))
	}

	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", version.UserAgent())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if c.Logger != nil {
		fmt.Fprintf(c.Logger, "> %s %s\n", method, reqURL)
	}

	start := time.Now()
	resp, err := c.HTTPClient.Do(req)
	elapsed := time.Since(start)

	c.requestCount.Add(1)
	c.totalLatency.Add(int64(elapsed))

	if err != nil {
		return nil, apierr.ErrNetwork(err)
	}

	if c.Logger != nil {
		fmt.Fprintf(c.Logger, "< %d %s (%dms)\n", resp.StatusCode, http.StatusText(resp.StatusCode), elapsed.Milliseconds())
	}

	return resp, nil
}

func (c *Client) doOnce(method, path string, body []byte, contentType, accept string) ([]byte, error) { //nolint:unparam // accept varies by caller intent

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	resp, err := c.doRequestAccept(method, path, bodyReader, contentType, accept)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return c.readBody(resp)
}

func (c *Client) readBody(resp *http.Response) ([]byte, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apierr.ErrNetwork(err)
	}
	if resp.StatusCode >= 400 {
		return nil, responseError(resp, data)
	}
	return data, nil
}

// doWithRetry performs an HTTP request with exponential backoff retry.
// Only safe for idempotent operations (GET, HEAD).
func (c *Client) doWithRetry(method, path string, body []byte, contentType, accept string) ([]byte, error) {
	var lastErr error
	for attempt := range maxRetries {
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		resp, err := c.doRequestAccept(method, path, bodyReader, contentType, accept)
		if err != nil {
			oerr := apierr.AsError(err)
			if !oerr.Retryable || attempt == maxRetries-1 {
				return nil, err
			}
			lastErr = err
			c.backoff(attempt)
			continue
		}

		data, err := c.readBody(resp)
		resp.Body.Close() //nolint:gosec // G104: error from Close is non-actionable in retry loop
		if err != nil {
			oerr := apierr.AsError(err)
			if oerr.Code == "rate_limit" && attempt < maxRetries-1 {
				retryAfter := 0
				if v := resp.Header.Get("Retry-After"); v != "" {
					_, _ = fmt.Sscanf(v, "%d", &retryAfter)
				}
				if retryAfter > 0 {
					c.SleepFunc(time.Duration(retryAfter) * time.Second)
				} else {
					c.backoff(attempt)
				}
				lastErr = err
				continue
			}
			if !oerr.Retryable || attempt == maxRetries-1 {
				return nil, err
			}
			lastErr = err
			c.backoff(attempt)
			continue
		}
		return data, nil
	}
	return nil, lastErr
}

func (c *Client) backoff(attempt int) {
	base := time.Duration(1<<uint(min(attempt, 30))) * time.Second //nolint:gosec // G115: attempt is bounded by maxRetries
	jitter := time.Duration(rand.Int64N(int64(base / 2)))          //nolint:gosec // G404: jitter does not need crypto-grade randomness
	wait := base + jitter
	if c.Logger != nil {
		fmt.Fprintf(c.Logger, "> retry #%d after %dms\n", attempt+1, wait.Milliseconds())
	}
	c.SleepFunc(wait)
}

func responseError(resp *http.Response, data []byte) *apierr.Error {
	switch resp.StatusCode {
	case 401:
		return apierr.ErrAuth("unauthorized — run `hey auth login` to authenticate")
	case 403:
		return apierr.ErrForbidden("forbidden (403)")
	case 404:
		return apierr.ErrNotFound("resource", strings.TrimSuffix(resp.Request.URL.Path, ".json"))
	case 429:
		retryAfter := 0
		if v := resp.Header.Get("Retry-After"); v != "" {
			_, _ = fmt.Sscanf(v, "%d", &retryAfter)
		}
		return apierr.ErrRateLimit(retryAfter)
	default:
		if resp.StatusCode >= 500 {
			return apierr.ErrAPI(resp.StatusCode, fmt.Sprintf("server error %d: %s", resp.StatusCode, string(data)))
		}
		return apierr.ErrAPI(resp.StatusCode, fmt.Sprintf("API error %d: %s", resp.StatusCode, string(data)))
	}
}

func (c *Client) Get(path string) ([]byte, error) {
	return c.doWithRetry("GET", path, nil, "", "application/json")
}

func (c *Client) GetHTML(path string) ([]byte, error) {
	return c.doWithRetry("GET", path, nil, "", "text/html")
}

func (c *Client) PostJSON(path string, body any) ([]byte, error) {
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			return nil, apierr.ErrAPI(0, fmt.Sprintf("could not encode body: %v", err))
		}
	}
	return c.doOnce("POST", path, encoded, "application/json", "application/json")
}

// RedirectResponse holds the Location header from a 302 redirect.
type RedirectResponse struct {
	Location string
}

// ExtractID extracts the last numeric path segment from the Location URL.
func (r *RedirectResponse) ExtractID() (int64, error) {
	parts := strings.Split(strings.TrimRight(r.Location, "/"), "/")
	if len(parts) == 0 {
		return 0, fmt.Errorf("no path segments in location: %s", r.Location)
	}
	last := parts[len(parts)-1]
	id, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse ID from location %q: %w", r.Location, err)
	}
	return id, nil
}

// doRedirect performs a request expecting a 302 redirect.
// Returns the response with Location header on 302, or an error for other statuses.
// Temporarily disables redirect-following on the shared HTTPClient for the duration of the call.
func (c *Client) doRedirect(method, path string, body io.Reader, contentType string) (*RedirectResponse, error) {
	// Temporarily disable redirect following
	orig := c.HTTPClient.CheckRedirect
	c.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() { c.HTTPClient.CheckRedirect = orig }()

	resp, err := c.doRequestAccept(method, path, body, contentType, "text/html")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther {
		return &RedirectResponse{Location: resp.Header.Get("Location")}, nil
	}

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, responseError(resp, data)
	}

	return nil, apierr.ErrAPI(resp.StatusCode, fmt.Sprintf("expected redirect, got %d", resp.StatusCode))
}

// PostForm sends a POST request with form-encoded body, expecting a 302 redirect.
func (c *Client) PostForm(path string, values url.Values) (*RedirectResponse, error) {
	return c.doRedirect("POST", path, strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
}

// PatchForm sends a PATCH request with form-encoded body, expecting a 302 redirect.
func (c *Client) PatchForm(path string, values url.Values) error {
	_, err := c.doRedirect("PATCH", path, strings.NewReader(values.Encode()), "application/x-www-form-urlencoded")
	return err
}

// Delete sends a DELETE request, expecting a 302 redirect.
func (c *Client) Delete(path string) error {
	_, err := c.doRedirect("DELETE", path, nil, "")
	return err
}
