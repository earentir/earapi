package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	yt "google.golang.org/api/youtube/v3"
)

// BuildAuthURL returns the user-consent URL for headless OAuth flow.
func BuildAuthURL(clientID, clientSecret string) string {
    cfg := &oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        Endpoint:     google.Endpoint,
        Scopes:       []string{yt.YoutubeScope},
        RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
    }
    return cfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode exchanges the auth code for a token and returns the refresh token (if any).
func ExchangeCode(ctx context.Context, clientID, clientSecret, code string) (refreshToken string, accessToken string, err error) {
    cfg := &oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        Endpoint:     google.Endpoint,
        Scopes:       []string{yt.YoutubeScope, yt.YoutubeForceSslScope},
        RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
    }
    tok, err := cfg.Exchange(ctx, code)
    if err != nil {
        return "", "", err
    }
    return tok.RefreshToken, tok.AccessToken, nil
}


// Device authorization flow (for headless environments)

type DeviceStartResponse struct {
    DeviceCode              string `json:"device_code"`
    UserCode                string `json:"user_code"`
    VerificationURL         string `json:"verification_url"`
    VerificationURLComplete string `json:"verification_url_complete"`
    ExpiresIn               int    `json:"expires_in"`
    Interval                int    `json:"interval"`
}

type deviceTokenError struct {
    Error            string `json:"error"`
    ErrorDescription string `json:"error_description"`
}

// StartDeviceFlow starts OAuth 2.0 Device Authorization flow for YouTube scopes.
func StartDeviceFlow(ctx context.Context, clientID string) (DeviceStartResponse, error) {
    form := url.Values{}
    form.Set("client_id", clientID)
    // Some TV/Device clients reject youtube.force-ssl in device flow; youtube scope is sufficient for playlist operations
    form.Set("scope", yt.YoutubeScope)

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/device/code", io.NopCloser(stringsNewReader(form.Encode())))
    if err != nil {
        return DeviceStartResponse{}, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return DeviceStartResponse{}, err
    }
    defer func() { _ = resp.Body.Close() }()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return DeviceStartResponse{}, errors.New(string(b))
    }
    var out DeviceStartResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return DeviceStartResponse{}, err
    }
    if out.Interval <= 0 {
        out.Interval = 5
    }
    return out, nil
}

// PollDeviceToken polls token endpoint until user authorizes or an error occurs.
// Returns refresh and access tokens on success.
func PollDeviceToken(ctx context.Context, clientID, clientSecret, deviceCode string, intervalSec int) (string, string, error) {
    if intervalSec <= 0 {
        intervalSec = 5
    }
    ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
    defer ticker.Stop()
    tokenURL := "https://oauth2.googleapis.com/token"

    for {
        select {
        case <-ctx.Done():
            return "", "", ctx.Err()
        case <-ticker.C:
            form := url.Values{}
            form.Set("client_id", clientID)
            if clientSecret != "" {
                form.Set("client_secret", clientSecret)
            }
            form.Set("device_code", deviceCode)
            form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

            req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, io.NopCloser(stringsNewReader(form.Encode())))
            if err != nil {
                return "", "", err
            }
            req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

            resp, err := http.DefaultClient.Do(req)
            if err != nil {
                return "", "", err
            }
            body, _ := io.ReadAll(resp.Body)
            _ = resp.Body.Close()
            if resp.StatusCode == 200 {
                var tok struct {
                    AccessToken  string `json:"access_token"`
                    RefreshToken string `json:"refresh_token"`
                }
                if err := json.Unmarshal(body, &tok); err != nil {
                    return "", "", err
                }
                return tok.RefreshToken, tok.AccessToken, nil
            }
            var derr deviceTokenError
            _ = json.Unmarshal(body, &derr)
            switch derr.Error {
            case "authorization_pending":
                // keep polling
            case "slow_down":
                intervalSec += 5
                ticker.Reset(time.Duration(intervalSec) * time.Second)
            default:
                if derr.Error != "" {
                    return "", "", errors.New(derr.Error + ": " + derr.ErrorDescription)
                }
                return "", "", errors.New(string(body))
            }
        }
    }
}

// small helper to avoid importing strings in multiple files
func stringsNewReader(s string) io.Reader { return stringsReader{s: s} }
type stringsReader struct{ s string }
func (r stringsReader) Read(p []byte) (n int, err error) {
    if len(r.s) == 0 { return 0, io.EOF }
    n = copy(p, r.s)
    r.s = r.s[n:]
    if len(r.s) == 0 { err = io.EOF }
    return
}


