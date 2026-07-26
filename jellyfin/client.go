// Package jellyfin talks to a Jellyfin server's REST API.
//
// Matching IMDb titles to library items is done client-side against a cached
// index rather than per-title server queries: the stable API (spec 12.0.0) has
// no provider-id filter on /Items, so a single bulk scan is both the only
// option and far fewer round trips.
package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a configured connection to one Jellyfin server.
type Client struct {
	BaseURL  string // normalized, no trailing slash
	apiKey   string
	deviceID string
	HTTP     *http.Client
}

// Credentials is what the user supplies in the UI.
type Credentials struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// ServerInfo is the useful subset of /System/Info.
type ServerInfo struct {
	ServerName string `json:"ServerName"`
	Version    string `json:"Version"`
	ID         string `json:"Id"`
}

// User is the useful subset of /Users/Me.
type User struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// Error is a classified Jellyfin failure.
type Error struct {
	Kind    string `json:"kind"` // config | auth | transport | upstream
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	err     error
}

func (e *Error) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.err)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}
func (e *Error) Unwrap() error { return e.err }

func jfErr(kind, msg, hint string, err error) *Error {
	return &Error{Kind: kind, Message: msg, Hint: hint, err: err}
}

// NewClient validates and normalizes credentials into a Client.
func NewClient(creds Credentials) (*Client, error) {
	raw := strings.TrimSpace(creds.URL)
	key := strings.TrimSpace(creds.APIKey)
	if raw == "" {
		return nil, jfErr("config", "no Jellyfin server address given", "", nil)
	}
	if key == "" {
		return nil, jfErr("config", "no Jellyfin API key given",
			"Create one in Jellyfin under Dashboard → API Keys.", nil)
	}
	// Bare host/IP is the common case for a LAN server; assume http.
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, jfErr("config", "that Jellyfin address isn't a valid URL", "", err)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") // keep any subpath, e.g. /jellyfin
	u.RawQuery, u.Fragment = "", ""

	return &Client{
		BaseURL:  u.String(),
		apiKey:   key,
		deviceID: "earapi",
		HTTP:     &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// authHeader builds the CustomAuthentication value the API expects.
func (c *Client) authHeader() string {
	return fmt.Sprintf(
		`MediaBrowser Token=%q, Client=%q, Device=%q, DeviceId=%q, Version=%q`,
		c.apiKey, "earapi", "earapi", c.deviceID, "1.0")
}

// do performs a request against path with optional query and JSON body.
func (c *Client) do(ctx context.Context, method, path string, q url.Values, body, out any) error {
	full := c.BaseURL + path
	if len(q) > 0 {
		full += "?" + q.Encode()
	}

	var rdr io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return jfErr("config", "could not encode request", "", err)
		}
		rdr = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, full, rdr)
	if err != nil {
		return jfErr("config", "could not build request", "", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return jfErr("transport", "could not reach the Jellyfin server",
			"Check the address and that the server is running and reachable from here.", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return jfErr("auth", "Jellyfin rejected that API key",
			"Generate a fresh key in Dashboard → API Keys and paste it again.", nil)
	case resp.StatusCode == http.StatusNotFound:
		return jfErr("upstream", "Jellyfin returned 404 for "+path,
			"Check the server address — if Jellyfin runs under a subpath, include it.", nil)
	case resp.StatusCode >= 400:
		msg := strings.TrimSpace(string(data))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return jfErr("upstream", fmt.Sprintf("Jellyfin returned HTTP %d", resp.StatusCode), msg, nil)
	}

	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return jfErr("upstream", "could not decode Jellyfin's response", "", err)
	}
	return nil
}

// SystemInfo fetches server identity — the connectivity probe.
func (c *Client) SystemInfo(ctx context.Context) (*ServerInfo, error) {
	var info ServerInfo
	if err := c.do(ctx, http.MethodGet, "/System/Info", nil, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Me resolves the user the API key acts as. Playlist calls need this id.
func (c *Client) Me(ctx context.Context) (*User, error) {
	var u User
	err := c.do(ctx, http.MethodGet, "/Users/Me", nil, nil, &u)
	if err == nil && u.ID != "" {
		return &u, nil
	}

	// An API key created in the dashboard isn't bound to a user on some
	// versions, so /Users/Me can come back empty. Fall back to the first
	// administrator account, which is what playlist writes should belong to.
	var users []User
	if lerr := c.do(ctx, http.MethodGet, "/Users", nil, nil, &users); lerr != nil {
		if err != nil {
			return nil, err
		}
		return nil, lerr
	}
	if len(users) == 0 {
		return nil, jfErr("auth", "no Jellyfin user is available for this API key",
			"Use a key created under a user account with library access.", nil)
	}
	return &users[0], nil
}

// Connection bundles a validated client with the identity it resolved.
type Connection struct {
	Client *Client     `json:"-"`
	Server *ServerInfo `json:"server"`
	User   *User       `json:"user"`
}

// Connect validates credentials end to end.
func Connect(ctx context.Context, creds Credentials) (*Connection, error) {
	c, err := NewClient(creds)
	if err != nil {
		return nil, err
	}
	info, err := c.SystemInfo(ctx)
	if err != nil {
		return nil, err
	}
	user, err := c.Me(ctx)
	if err != nil {
		return nil, err
	}
	return &Connection{Client: c, Server: info, User: user}, nil
}

// AsJFError extracts a *Error from an error chain.
func AsJFError(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
