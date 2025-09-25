package youtube

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	yt "google.golang.org/api/youtube/v3"
)

type Config struct {
    ClientID       string
    ClientSecret   string
    RefreshToken   string
    DefaultChannel string
    CacheMinutes   int
    OnRefresh      func(newRefreshToken string) error
}

type Service struct {
    yt       *yt.Service
    cfg      Config
    cache    *cacheStore
    httpClient *http.Client
    oauthCfg  *oauth2.Config
}

type cacheStore struct {
    mu              sync.RWMutex
    playlists       []playlistInfo
    playlistVideos  map[string]map[string]videoInfo // playlistID -> videoID -> info
    lastRefreshed   time.Time
    ttl             time.Duration
}

type playlistInfo struct {
    ID    string
    Title string
}

type videoInfo struct {
    ID    string
    Title string
}

func New(ctx context.Context, cfg Config) (*Service, error) {
    if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RefreshToken == "" {
        return nil, errors.New("youtube oauth config is incomplete")
    }

    oauthCfg := &oauth2.Config{
        ClientID:     cfg.ClientID,
        ClientSecret: cfg.ClientSecret,
        Endpoint:     google.Endpoint,
        Scopes:       []string{yt.YoutubeScope, yt.YoutubeForceSslScope},
        RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
    }
    token := &oauth2.Token{RefreshToken: cfg.RefreshToken}
    httpClient := oauthCfg.Client(ctx, token)

    svc, err := yt.NewService(ctx, option.WithHTTPClient(httpClient), option.WithUserAgent("earapi-youtube/1.0"))
    if err != nil {
        return nil, err
    }

    ttl := time.Duration(cfg.CacheMinutes)
    if ttl <= 0 {
        ttl = 10
    }
    ttl = ttl * time.Minute

    s := &Service{
        yt:        svc,
        cfg:       cfg,
        httpClient: httpClient,
        oauthCfg:  oauthCfg,
        cache: &cacheStore{
            playlists:      nil,
            playlistVideos: make(map[string]map[string]videoInfo),
            ttl:            ttl,
        },
    }

    // background weekly refresh to keep refresh token alive if provider rotates
    go s.startWeeklyRefresh(context.Background())

    return s, nil
}

func (s *Service) ensureCache(ctx context.Context) error {
    s.cache.mu.RLock()
    fresh := time.Since(s.cache.lastRefreshed) < s.cache.ttl && len(s.cache.playlists) > 0
    s.cache.mu.RUnlock()
    if fresh {
        return nil
    }
    return s.refreshCache(ctx)
}

func (s *Service) refreshCache(ctx context.Context) error {
    // Load playlists for the authenticated channel
    playlists, err := s.fetchAllPlaylists(ctx)
    if err != nil {
        return err
    }
    // For each playlist, load its items (video IDs and titles)
    playlistVideos := make(map[string]map[string]videoInfo)
    for _, pl := range playlists {
        vids, err := s.fetchAllPlaylistVideos(ctx, pl.ID)
        if err != nil {
            return err
        }
        inner := make(map[string]videoInfo)
        for _, v := range vids {
            inner[v.ID] = v
        }
        playlistVideos[pl.ID] = inner
    }
    s.cache.mu.Lock()
    s.cache.playlists = playlists
    s.cache.playlistVideos = playlistVideos
    s.cache.lastRefreshed = time.Now()
    s.cache.mu.Unlock()
    return nil
}

func (s *Service) fetchAllPlaylists(ctx context.Context) ([]playlistInfo, error) {
    var results []playlistInfo
    pageToken := ""
    for {
        call := s.yt.Playlists.List([]string{"id", "snippet"}).Mine(true).MaxResults(50)
        if pageToken != "" {
            call = call.PageToken(pageToken)
        }
        resp, err := call.Context(ctx).Do()
        if err != nil {
            return nil, err
        }
        for _, item := range resp.Items {
            results = append(results, playlistInfo{ID: item.Id, Title: item.Snippet.Title})
        }
        if resp.NextPageToken == "" {
            break
        }
        pageToken = resp.NextPageToken
    }
    return results, nil
}

func (s *Service) fetchAllPlaylistVideos(ctx context.Context, playlistID string) ([]videoInfo, error) {
    var results []videoInfo
    pageToken := ""
    for {
        call := s.yt.PlaylistItems.List([]string{"contentDetails", "snippet"}).PlaylistId(playlistID).MaxResults(50)
        if pageToken != "" {
            call = call.PageToken(pageToken)
        }
        resp, err := call.Context(ctx).Do()
        if err != nil {
            return nil, err
        }
        for _, item := range resp.Items {
            vid := videoInfo{ID: item.ContentDetails.VideoId, Title: item.Snippet.Title}
            results = append(results, vid)
        }
        if resp.NextPageToken == "" {
            break
        }
        pageToken = resp.NextPageToken
    }
    return results, nil
}

var (
    reShort = regexp.MustCompile(`(?i)^(?:youtu\.be/)([\w-]{11})`)
    reWatch = regexp.MustCompile(`(?i)(?:v=)([\w-]{11})`)
    reEmbed = regexp.MustCompile(`(?i)(?:embed/)([\w-]{11})`)
)

func extractVideoID(input string) string {
    s := strings.TrimSpace(input)
    if len(s) == 11 && !strings.ContainsAny(s, "/?=& ") {
        return s
    }
    if m := reShort.FindStringSubmatch(s); len(m) == 2 {
        return m[1]
    }
    if m := reWatch.FindStringSubmatch(s); len(m) == 2 {
        return m[1]
    }
    if m := reEmbed.FindStringSubmatch(s); len(m) == 2 {
        return m[1]
    }
    return ""
}

func (s *Service) findBestPlaylistByName(ctx context.Context, name string) (playlistInfo, bool, error) {
    if err := s.ensureCache(ctx); err != nil {
        return playlistInfo{}, false, err
    }
    s.cache.mu.RLock()
    defer s.cache.mu.RUnlock()
    target := strings.TrimSpace(strings.ToLower(name))
    var best playlistInfo
    bestScore := -1
    for _, pl := range s.cache.playlists {
        score := -levenshteinDistance(strings.ToLower(pl.Title), target)
        if score > bestScore {
            bestScore = score
            best = pl
        }
    }
    if best.ID == "" {
        return playlistInfo{}, false, nil
    }
    return best, true, nil
}

func (s *Service) findExactPlaylistByName(ctx context.Context, name string) (playlistInfo, bool, error) {
    if err := s.ensureCache(ctx); err != nil {
        return playlistInfo{}, false, err
    }
    s.cache.mu.RLock()
    defer s.cache.mu.RUnlock()
    target := strings.TrimSpace(name)
    for _, pl := range s.cache.playlists {
        if pl.Title == target {
            return pl, true, nil
        }
    }
    return playlistInfo{}, false, nil
}

func (s *Service) getVideoTitle(ctx context.Context, videoID string) (string, error) {
    resp, err := s.yt.Videos.List([]string{"snippet"}).Id(videoID).Context(ctx).Do()
    if err != nil {
        return "", err
    }
    if len(resp.Items) == 0 {
        return "", errors.New("video not found")
    }
    return resp.Items[0].Snippet.Title, nil
}

type AddResult struct {
    Added      bool   `json:"added"`
    Reason     string `json:"reason,omitempty"`
    PlaylistID string `json:"playlistId"`
    PlaylistTitle string `json:"playlistTitle"`
    VideoID    string `json:"videoId"`
}

func (s *Service) AddVideoToPlaylist(ctx context.Context, playlistName string, video string, force bool) (AddResult, error) {
    if err := s.ensureCache(ctx); err != nil {
        return AddResult{}, err
    }
    pl, ok, err := s.findBestPlaylistByName(ctx, playlistName)
    if err != nil {
        return AddResult{}, err
    }
    if !ok {
        return AddResult{}, errors.New("playlist not found")
    }

    videoID := extractVideoID(video)
    if videoID == "" {
        return AddResult{}, errors.New("invalid video identifier")
    }

    // Duplicate by ID
    s.cache.mu.RLock()
    existingByID := s.cache.playlistVideos[pl.ID][videoID]
    s.cache.mu.RUnlock()
    if existingByID.ID != "" && !force {
        return AddResult{Added: false, Reason: "duplicate by id", PlaylistID: pl.ID, PlaylistTitle: pl.Title, VideoID: videoID}, nil
    }

    // Duplicate by title (Levenshtein)
    title, err := s.getVideoTitle(ctx, videoID)
    if err != nil {
        return AddResult{}, err
    }
    normalizedTarget := strings.ToLower(strings.TrimSpace(title))
    isDupByTitle := false
    s.cache.mu.RLock()
    for _, vi := range s.cache.playlistVideos[pl.ID] {
        existing := strings.ToLower(strings.TrimSpace(vi.Title))
        // Simple similarity: treat duplicate if distance small relative to length
        d := levenshteinDistance(existing, normalizedTarget)
        maxLen := len(existing)
        if len(normalizedTarget) > maxLen {
            maxLen = len(normalizedTarget)
        }
        if maxLen == 0 {
            continue
        }
        // Consider duplicate if similarity >= 0.9 roughly => distance <= 10% of length
        if float64(d) <= 0.1*float64(maxLen) {
            isDupByTitle = true
            break
        }
    }
    s.cache.mu.RUnlock()

    if isDupByTitle && !force {
        return AddResult{Added: false, Reason: "duplicate by title", PlaylistID: pl.ID, PlaylistTitle: pl.Title, VideoID: videoID}, nil
    }

    // Add to playlist
    _, err = s.yt.PlaylistItems.Insert([]string{"snippet"}, &yt.PlaylistItem{
        Snippet: &yt.PlaylistItemSnippet{
            PlaylistId: pl.ID,
            ResourceId: &yt.ResourceId{Kind: "youtube#video", VideoId: videoID},
        },
    }).Context(ctx).Do()
    if err != nil {
        // If force, we still attempted; return error
        return AddResult{}, err
    }

    // Update cache
    s.cache.mu.Lock()
    if s.cache.playlistVideos[pl.ID] == nil {
        s.cache.playlistVideos[pl.ID] = make(map[string]videoInfo)
    }
    s.cache.playlistVideos[pl.ID][videoID] = videoInfo{ID: videoID, Title: title}
    s.cache.mu.Unlock()

    return AddResult{Added: true, PlaylistID: pl.ID, PlaylistTitle: pl.Title, VideoID: videoID}, nil
}

type PlaylistItem struct {
    VideoID string `json:"videoId"`
    Title   string `json:"title"`
}

// ListItems returns items for a playlist by name; fuzzy controls name matching.
func (s *Service) ListItems(ctx context.Context, playlistName string, fuzzy bool) ([]PlaylistItem, playlistInfo, error) {
    if err := s.ensureCache(ctx); err != nil {
        return nil, playlistInfo{}, err
    }
    var pl playlistInfo
    var ok bool
    var err error
    if fuzzy {
        pl, ok, err = s.findBestPlaylistByName(ctx, playlistName)
    } else {
        pl, ok, err = s.findExactPlaylistByName(ctx, playlistName)
    }
    if err != nil {
        return nil, playlistInfo{}, err
    }
    if !ok {
        return nil, playlistInfo{}, errors.New("playlist not found")
    }
    // Ensure we have fresh items for this playlist in cache (already loaded in refresh)
    s.cache.mu.RLock()
    vids := s.cache.playlistVideos[pl.ID]
    s.cache.mu.RUnlock()
    // If missing, fetch directly
    if vids == nil {
        list, err := s.fetchAllPlaylistVideos(ctx, pl.ID)
        if err != nil {
            return nil, playlistInfo{}, err
        }
        inner := make(map[string]videoInfo)
        for _, v := range list {
            inner[v.ID] = v
        }
        s.cache.mu.Lock()
        s.cache.playlistVideos[pl.ID] = inner
        s.cache.mu.Unlock()
        vids = inner
    }
    out := make([]PlaylistItem, 0, len(vids))
    for _, v := range vids {
        out = append(out, PlaylistItem{VideoID: v.ID, Title: v.Title})
    }
    return out, pl, nil
}

// CreatePlaylist creates a new playlist with the given name and privacy status.
func (s *Service) CreatePlaylist(ctx context.Context, name string, privacy string) (playlistInfo, error) {
    if name == "" {
        return playlistInfo{}, errors.New("playlist name required")
    }
    if privacy == "" {
        privacy = "private"
    }
    res, err := s.yt.Playlists.Insert([]string{"snippet", "status"}, &yt.Playlist{
        Snippet: &yt.PlaylistSnippet{Title: name},
        Status:  &yt.PlaylistStatus{PrivacyStatus: privacy},
    }).Context(ctx).Do()
    if err != nil {
        return playlistInfo{}, err
    }
    pl := playlistInfo{ID: res.Id, Title: name}
    // Update cache minimally
    s.cache.mu.Lock()
    s.cache.playlists = append(s.cache.playlists, pl)
    if s.cache.playlistVideos[pl.ID] == nil {
        s.cache.playlistVideos[pl.ID] = make(map[string]videoInfo)
    }
    s.cache.mu.Unlock()
    return pl, nil
}


// levenshteinDistance computes the Levenshtein edit distance between two strings.
func levenshteinDistance(a, b string) int {
    la := len(a)
    lb := len(b)
    if la == 0 {
        return lb
    }
    if lb == 0 {
        return la
    }
    // Use two rows to reduce memory.
    prev := make([]int, lb+1)
    curr := make([]int, lb+1)
    for j := 0; j <= lb; j++ {
        prev[j] = j
    }
    for i := 1; i <= la; i++ {
        curr[0] = i
        ai := a[i-1]
        for j := 1; j <= lb; j++ {
            cost := 0
            if ai != b[j-1] {
                cost = 1
            }
            deletion := prev[j] + 1
            insertion := curr[j-1] + 1
            substitution := prev[j-1] + cost
            curr[j] = minInt(deletion, minInt(insertion, substitution))
        }
        prev, curr = curr, prev
    }
    return prev[lb]
}

func minInt(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func (s *Service) startWeeklyRefresh(ctx context.Context) {
    ticker := time.NewTicker(7 * 24 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            // Proactively refresh to surface any new refresh_token
            ts := s.oauthCfg.TokenSource(ctx, &oauth2.Token{RefreshToken: s.cfg.RefreshToken})
            tok, err := ts.Token()
            if err == nil && tok != nil {
                if tok.RefreshToken != "" && tok.RefreshToken != s.cfg.RefreshToken {
                    s.cfg.RefreshToken = tok.RefreshToken
                    if s.cfg.OnRefresh != nil {
                        _ = s.cfg.OnRefresh(tok.RefreshToken)
                    }
                }
            }
        case <-ctx.Done():
            return
        }
    }
}


