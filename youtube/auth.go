package youtube

import (
	"context"

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
        Scopes:       []string{yt.YoutubeScope, yt.YoutubeForceSslScope},
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


