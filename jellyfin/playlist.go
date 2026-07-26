package jellyfin

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

const addBatchSize = 100

// SyncMode decides what happens when a playlist of that name already exists.
type SyncMode string

const (
	ModeCreate  SyncMode = "create"  // always make a new playlist
	ModeAppend  SyncMode = "append"  // add only the items not already in it
	ModeReplace SyncMode = "replace" // clear it, then add
)

// Playlist is the subset of a playlist item we care about.
type Playlist struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// SyncRequest describes a playlist write.
type SyncRequest struct {
	Name    string   `json:"name"`
	Mode    SyncMode `json:"mode"`
	Public  bool     `json:"public"`
	ItemIDs []string `json:"item_ids"`
}

// SyncPlan is what a dry run reports: exactly what a write would do.
type SyncPlan struct {
	PlaylistName  string   `json:"playlist_name"`
	PlaylistID    string   `json:"playlist_id,omitempty"`
	Mode          SyncMode `json:"mode"`
	Exists        bool     `json:"exists"`
	ExistingCount int      `json:"existing_count"`
	ToAdd         int      `json:"to_add"`
	ToRemove      int      `json:"to_remove"`
	AlreadyThere  int      `json:"already_there"`
	FinalCount    int      `json:"final_count"`
}

// SyncResult is what actually happened.
type SyncResult struct {
	Plan       SyncPlan `json:"plan"`
	PlaylistID string   `json:"playlist_id"`
	Added      int      `json:"added"`
	Removed    int      `json:"removed"`
	Created    bool     `json:"created"`
}

// ListPlaylists returns playlists visible to the given user.
func (c *Client) ListPlaylists(ctx context.Context, userID string) ([]Playlist, error) {
	q := url.Values{}
	q.Set("userId", userID)
	q.Set("recursive", "true")
	q.Set("includeItemTypes", "Playlist")
	q.Set("limit", "2000")
	q.Set("enableImages", "false")

	var page struct {
		Items []Playlist `json:"Items"`
	}
	if err := c.do(ctx, http.MethodGet, "/Items", q, nil, &page); err != nil {
		return nil, err
	}
	if page.Items == nil {
		return []Playlist{}, nil
	}
	return page.Items, nil
}

// FindPlaylist looks up a playlist by exact (case-insensitive) name.
func (c *Client) FindPlaylist(ctx context.Context, userID, name string) (*Playlist, error) {
	items, err := c.ListPlaylists(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, p := range items {
		if strings.EqualFold(strings.TrimSpace(p.Name), strings.TrimSpace(name)) {
			return &p, nil
		}
	}
	return nil, nil
}

// playlistEntry carries both ids a playlist row has: the underlying library
// item id (used to compare against what we want to sync) and the per-entry id
// (the only thing DELETE /Playlists/{id}/Items accepts).
type playlistEntry struct {
	ID             string `json:"Id"`
	PlaylistItemID string `json:"PlaylistItemId"`
}

func (c *Client) playlistEntries(ctx context.Context, playlistID, userID string) ([]playlistEntry, error) {
	q := url.Values{}
	q.Set("userId", userID)
	q.Set("limit", "10000")
	q.Set("enableImages", "false")

	var page struct {
		Items []playlistEntry `json:"Items"`
	}
	if err := c.do(ctx, http.MethodGet,
		"/Playlists/"+url.PathEscape(playlistID)+"/Items", q, nil, &page); err != nil {
		return nil, err
	}
	return page.Items, nil
}

// PlaylistItemIDs returns the library item ids currently in a playlist.
func (c *Client) PlaylistItemIDs(ctx context.Context, playlistID, userID string) ([]string, error) {
	entries, err := c.playlistEntries(ctx, playlistID, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, it := range entries {
		if it.ID != "" {
			ids = append(ids, it.ID)
		}
	}
	return ids, nil
}

// playlistEntryIDs returns the per-entry ids needed for removal.
func (c *Client) playlistEntryIDs(ctx context.Context, playlistID, userID string) ([]string, error) {
	entries, err := c.playlistEntries(ctx, playlistID, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, it := range entries {
		// Prefer the entry id; fall back to the item id on older servers.
		if it.PlaylistItemID != "" {
			ids = append(ids, it.PlaylistItemID)
		} else if it.ID != "" {
			ids = append(ids, it.ID)
		}
	}
	return ids, nil
}

// PlanSync computes what a sync would do, without writing anything.
func (c *Client) PlanSync(ctx context.Context, userID string, req SyncRequest) (*SyncPlan, error) {
	plan := &SyncPlan{
		PlaylistName: strings.TrimSpace(req.Name),
		Mode:         req.Mode,
	}
	if plan.PlaylistName == "" {
		return nil, jfErr("config", "no playlist name given", "", nil)
	}

	wanted := dedupe(req.ItemIDs)

	if req.Mode != ModeCreate {
		existing, err := c.FindPlaylist(ctx, userID, plan.PlaylistName)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			plan.Exists = true
			plan.PlaylistID = existing.ID
			current, err := c.PlaylistItemIDs(ctx, existing.ID, userID)
			if err != nil {
				return nil, err
			}
			plan.ExistingCount = len(current)

			switch req.Mode {
			case ModeAppend:
				have := toSet(current)
				for _, id := range wanted {
					if have[id] {
						plan.AlreadyThere++
					} else {
						plan.ToAdd++
					}
				}
				plan.FinalCount = plan.ExistingCount + plan.ToAdd
			case ModeReplace:
				plan.ToRemove = len(current)
				plan.ToAdd = len(wanted)
				plan.FinalCount = len(wanted)
			}
			return plan, nil
		}
	}

	// Nothing to merge with: creating fresh.
	plan.ToAdd = len(wanted)
	plan.FinalCount = len(wanted)
	return plan, nil
}

// Sync performs the write described by req. Callers are expected to have shown
// the user a PlanSync result and taken explicit confirmation first.
func (c *Client) Sync(ctx context.Context, userID string, req SyncRequest) (*SyncResult, error) {
	plan, err := c.PlanSync(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	wanted := dedupe(req.ItemIDs)
	res := &SyncResult{Plan: *plan}

	// Existing playlist, append or replace.
	if plan.Exists && plan.PlaylistID != "" && req.Mode != ModeCreate {
		res.PlaylistID = plan.PlaylistID

		if req.Mode == ModeReplace {
			entries, err := c.playlistEntryIDs(ctx, plan.PlaylistID, userID)
			if err != nil {
				return nil, err
			}
			if len(entries) > 0 {
				for _, batch := range chunk(entries, addBatchSize) {
					q := url.Values{}
					q.Set("entryIds", strings.Join(batch, ","))
					if err := c.do(ctx, http.MethodDelete,
						"/Playlists/"+url.PathEscape(plan.PlaylistID)+"/Items", q, nil, nil); err != nil {
						return nil, err
					}
				}
				res.Removed = len(entries)
			}
		}

		toAdd := wanted
		if req.Mode == ModeAppend {
			current, err := c.PlaylistItemIDs(ctx, plan.PlaylistID, userID)
			if err != nil {
				return nil, err
			}
			have := toSet(current)
			toAdd = toAdd[:0:0]
			for _, id := range wanted {
				if !have[id] {
					toAdd = append(toAdd, id)
				}
			}
		}
		if err := c.addItems(ctx, plan.PlaylistID, userID, toAdd); err != nil {
			return nil, err
		}
		res.Added = len(toAdd)
		return res, nil
	}

	// Create a new playlist. The first batch goes in the create call; the rest
	// are appended, since a very large Ids payload is rejected by some servers.
	first := wanted
	var rest []string
	if len(first) > addBatchSize {
		first, rest = wanted[:addBatchSize], wanted[addBatchSize:]
	}

	body := map[string]any{
		"Name":      plan.PlaylistName,
		"Ids":       first,
		"UserId":    userID,
		"MediaType": "Video",
		"IsPublic":  req.Public,
	}
	var created struct {
		ID string `json:"Id"`
	}
	if err := c.do(ctx, http.MethodPost, "/Playlists", nil, body, &created); err != nil {
		return nil, err
	}
	if created.ID == "" {
		return nil, jfErr("upstream", "Jellyfin created the playlist but returned no id", "", nil)
	}
	res.PlaylistID = created.ID
	res.Created = true
	res.Added = len(first)

	if len(rest) > 0 {
		if err := c.addItems(ctx, created.ID, userID, rest); err != nil {
			return nil, err
		}
		res.Added += len(rest)
	}
	return res, nil
}

// addItems appends item ids in batches.
func (c *Client) addItems(ctx context.Context, playlistID, userID string, ids []string) error {
	for _, batch := range chunk(ids, addBatchSize) {
		q := url.Values{}
		q.Set("ids", strings.Join(batch, ","))
		q.Set("userId", userID)
		if err := c.do(ctx, http.MethodPost,
			"/Playlists/"+url.PathEscape(playlistID)+"/Items", q, nil, nil); err != nil {
			return err
		}
	}
	return nil
}

func chunk(s []string, n int) [][]string {
	if len(s) == 0 {
		return nil
	}
	var out [][]string
	for i := 0; i < len(s); i += n {
		out = append(out, s[i:min(i+n, len(s))])
	}
	return out
}

func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func toSet(in []string) map[string]bool {
	m := make(map[string]bool, len(in))
	for _, s := range in {
		m[s] = true
	}
	return m
}
