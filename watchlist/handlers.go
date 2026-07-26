package watchlist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"earapi/compare"
	"earapi/imdb"
	"earapi/jellyfin"
)

// RegisterRoutes mounts IMDb, watchlist, compare, Jellyfin, and jobs endpoints.
func RegisterRoutes(r *gin.Engine, svc *Service) {
	imdbG := r.Group("/imdb/v1")
	{
		imdbG.POST("/resolve", svc.handleResolve)
		imdbG.POST("/fetch", svc.handleFetch)
		imdbG.POST("/import-csv", svc.handleImportCSV)
		imdbG.POST("/titles/hydrate", svc.handleHydrate)
	}

	wlG := r.Group("/watchlist/v1")
	{
		wlG.GET("/:id", svc.handleGetWatchlist)
		wlG.GET("/:id/export", svc.handleExportWatchlist)
	}

	cmpG := r.Group("/compare/v1")
	{
		cmpG.POST("", svc.handleCompare)
		cmpG.POST("/", svc.handleCompare)
		cmpG.GET("/:id", svc.handleGetCompare)
		cmpG.GET("/:id/export", svc.handleExportCompare)
	}

	jfG := r.Group("/jellyfin/v1")
	{
		jfG.POST("/connect", svc.handleJFConnect)
		jfG.POST("/disconnect", svc.handleJFDisconnect)
		jfG.GET("/status", svc.handleJFStatus)
		jfG.POST("/scan", svc.handleJFScan)
		jfG.POST("/match", svc.handleJFMatch)
		jfG.POST("/sync", svc.handleJFSync)
		jfG.GET("/playlists", svc.handleJFPlaylists)
		jfG.GET("/playlists/items", svc.handleJFPlaylistItems)
	}

	jobsG := r.Group("/jobs/v1")
	{
		jobsG.GET("/:id", svc.handleJobStatus)
		jobsG.GET("/:id/events", svc.handleJobEvents)
	}

	r.GET("/watchlistsync/v1/capabilities", svc.handleCapabilities)
}

func (s *Service) handleCapabilities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"supports_alias": s.BrowserName != "",
		"browser":        s.BrowserName,
		"groups": []string{
			"/imdb/v1", "/watchlist/v1", "/compare/v1", "/jellyfin/v1", "/jobs/v1",
		},
	})
}

// --- error helpers -----------------------------------------------------------

type apiError struct {
	Error struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
		Hint    string `json:"hint,omitempty"`
	} `json:"error"`
}

func writeErr(c *gin.Context, code int, kind, msg, hint string) {
	var e apiError
	e.Error.Kind, e.Error.Message, e.Error.Hint = kind, msg, hint
	c.JSON(code, e)
}

func writeDomainErr(c *gin.Context, err error) {
	var ie *imdb.Error
	if errors.As(err, &ie) {
		code := http.StatusBadGateway
		switch ie.Kind {
		case imdb.ErrKindInvalidInput, imdb.ErrKindProfileAlias:
			code = http.StatusBadRequest
		case imdb.ErrKindNotFound:
			code = http.StatusNotFound
		case imdb.ErrKindPrivate:
			code = http.StatusForbidden
		}
		writeErr(c, code, string(ie.Kind), ie.Message, ie.Hint)
		return
	}
	var je *jellyfin.Error
	if errors.As(err, &je) {
		code := http.StatusBadGateway
		switch je.Kind {
		case "config":
			code = http.StatusBadRequest
		case "auth":
			code = http.StatusUnauthorized
		}
		writeErr(c, code, je.Kind, je.Message, je.Hint)
		return
	}
	writeErr(c, http.StatusInternalServerError, "internal", err.Error(), "")
}

func jobErrOf(err error) (kind, msg, hint string) {
	var ie *imdb.Error
	if errors.As(err, &ie) {
		return string(ie.Kind), ie.Message, ie.Hint
	}
	var je *jellyfin.Error
	if errors.As(err, &je) {
		return je.Kind, je.Message, je.Hint
	}
	return "internal", err.Error(), ""
}

// --- imdb --------------------------------------------------------------------

func (s *Service) handleResolve(c *gin.Context) {
	var req struct {
		Input string `json:"input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", "malformed request", "")
		return
	}
	ref, err := imdb.ParseRef(req.Input)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	c.JSON(http.StatusOK, ref)
}

func (s *Service) handleFetch(c *gin.Context) {
	var req struct {
		Input   string `json:"input"`
		Owner   string `json:"owner"`
		Refresh bool   `json:"refresh"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", "malformed request", "")
		return
	}
	ref, err := imdb.ParseRef(req.Input)
	if err != nil {
		writeDomainErr(c, err)
		return
	}

	phase := "Fetching from IMDb"
	if ref.Kind == imdb.KindAlias {
		phase = "Opening IMDb in " + firstNonEmpty(s.BrowserName, "a browser")
	}
	job := s.Jobs.Create(phase)
	owner := strings.TrimSpace(req.Owner)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if !req.Refresh {
			if wl, ok := s.Store.CachedList(ref); ok {
				wl.Owner = owner
				id := s.Store.PutWatchlist(wl)
				job.Done(map[string]any{"watchlist_id": id, "cached": true, "watchlist": wl})
				return
			}
		}

		wl, err := s.IMDb.FetchList(ctx, ref, func(fetched, total int) {
			job.Progress("Fetching from IMDb", fetched, total)
		})
		if err != nil {
			job.Fail(jobErrOf(err))
			return
		}
		wl.Owner = owner
		s.Store.CacheList(ref, wl)
		id := s.Store.PutWatchlist(wl)
		job.Done(map[string]any{"watchlist_id": id, "cached": false, "watchlist": wl})
	}()

	c.JSON(http.StatusAccepted, gin.H{"job_id": job.ID, "source": ref})
}

func (s *Service) handleImportCSV(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", "no file was uploaded", "")
		return
	}
	defer file.Close()

	owner := strings.TrimSpace(c.PostForm("owner"))
	hydrate := c.DefaultPostForm("hydrate", "true") != "false"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	wl, err := imdb.ImportCSV(ctx, file, owner, s.IMDb, hydrate)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	if header != nil && header.Filename != "" {
		wl.Name = header.Filename
		wl.Source.Label = header.Filename
	}
	id := s.Store.PutWatchlist(wl)
	c.JSON(http.StatusOK, gin.H{"watchlist_id": id, "watchlist": wl})
}

func (s *Service) handleHydrate(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", "malformed request", "")
		return
	}
	if len(req.IDs) == 0 {
		writeErr(c, http.StatusBadRequest, "invalid_input", "no title ids given", "")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	titles, err := s.IMDb.FetchTitles(ctx, req.IDs, nil)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(titles), "titles": titles})
}

func (s *Service) handleGetWatchlist(c *gin.Context) {
	wl, ok := s.Store.Watchlist(c.Param("id"))
	if !ok {
		writeErr(c, http.StatusNotFound, "not_found", "that watchlist is no longer loaded",
			"Fetch it again — results are kept only while the API process is running.")
		return
	}
	c.JSON(http.StatusOK, wl)
}

func (s *Service) handleExportWatchlist(c *gin.Context) {
	wl, ok := s.Store.Watchlist(c.Param("id"))
	if !ok {
		writeErr(c, http.StatusNotFound, "not_found", "that watchlist is no longer loaded", "")
		return
	}
	base := safeFilename(firstNonEmpty(wl.Owner, wl.Name, "watchlist"))

	if c.Query("format") == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="`+base+`.csv"`)
		_ = imdb.ExportCSV(c.Writer, wl.Titles)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+base+`.json"`)
	c.JSON(http.StatusOK, gin.H{
		"schema_version": 1,
		"exported_at":    time.Now().UTC(),
		"source":         wl.Source,
		"owner":          wl.Owner,
		"name":           wl.Name,
		"fetched_at":     wl.FetchedAt,
		"count":          wl.Count,
		"titles":         wl.Titles,
	})
}

// --- compare -----------------------------------------------------------------

func (s *Service) handleCompare(c *gin.Context) {
	var req struct {
		WatchlistIDs []string `json:"watchlist_ids"`
		Lists        []struct {
			Owner  string       `json:"owner"`
			Titles []imdb.Title `json:"titles"`
		} `json:"lists"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", "malformed request", "")
		return
	}

	var inputs []compare.Input
	switch {
	case len(req.Lists) >= 2:
		inputs = make([]compare.Input, 0, len(req.Lists))
		for i, l := range req.Lists {
			owner := strings.TrimSpace(l.Owner)
			if owner == "" {
				owner = fmt.Sprintf("List %d", i+1)
			}
			inputs = append(inputs, compare.Input{Owner: owner, Titles: l.Titles})
		}
	case len(req.WatchlistIDs) >= 2:
		inputs = make([]compare.Input, 0, len(req.WatchlistIDs))
		for i, id := range req.WatchlistIDs {
			wl, ok := s.Store.Watchlist(id)
			if !ok {
				writeErr(c, http.StatusNotFound, "not_found",
					"one of those lists is no longer loaded", "Fetch it again, then compare.")
				return
			}
			owner := wl.Owner
			if owner == "" {
				owner = firstNonEmpty(wl.Name, fmt.Sprintf("List %d", i+1))
			}
			inputs = append(inputs, compare.Input{Owner: owner, Titles: wl.Titles})
		}
	default:
		writeErr(c, http.StatusBadRequest, "invalid_input",
			"comparing needs at least two lists",
			"Pass watchlist_ids from prior fetches, or inline lists with titles.")
		return
	}

	res := compare.Compare(inputs)
	id := s.Store.PutCompare(res)
	c.JSON(http.StatusOK, gin.H{"compare_id": id, "result": res})
}

func (s *Service) handleGetCompare(c *gin.Context) {
	res, ok := s.Store.Compare(c.Param("id"))
	if !ok {
		writeErr(c, http.StatusNotFound, "not_found", "that comparison is no longer loaded", "")
		return
	}
	c.JSON(http.StatusOK, res)
}

func (s *Service) handleExportCompare(c *gin.Context) {
	res, ok := s.Store.Compare(c.Param("id"))
	if !ok {
		writeErr(c, http.StatusNotFound, "not_found", "that comparison is no longer loaded", "")
		return
	}
	view := c.Query("view")
	titles, ok := res.View(view)
	if !ok {
		writeErr(c, http.StatusBadRequest, "invalid_input", "unknown view "+view, "")
		return
	}
	base := safeFilename("watchlist-" + firstNonEmpty(view, "all"))

	if c.Query("format") == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="`+base+`.csv"`)
		_ = imdb.ExportCSV(c.Writer, titles)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+base+`.json"`)
	c.JSON(http.StatusOK, gin.H{
		"schema_version": 1,
		"exported_at":    time.Now().UTC(),
		"view":           firstNonEmpty(view, "all"),
		"owners":         res.Owners,
		"stats":          res.Stats,
		"count":          len(titles),
		"titles":         titles,
	})
}

// --- jellyfin ----------------------------------------------------------------

func (s *Service) handleJFConnect(c *gin.Context) {
	var req struct {
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", "malformed request", "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	conn, err := jellyfin.Connect(ctx, jellyfin.Credentials{URL: req.URL, APIKey: req.APIKey})
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	s.setConnection(conn)
	c.JSON(http.StatusOK, gin.H{"server": conn.Server, "user": conn.User, "url": conn.Client.BaseURL})
}

func (s *Service) handleJFDisconnect(c *gin.Context) {
	s.clearConnection()
	c.JSON(http.StatusOK, gin.H{"connected": false})
}

func (s *Service) handleJFStatus(c *gin.Context) {
	conn := s.connection()
	idx := s.index()
	out := gin.H{"connected": conn != nil}
	if conn != nil {
		out["server"] = conn.Server
		out["user"] = conn.User
		out["url"] = conn.Client.BaseURL
	}
	if idx != nil {
		out["library"] = gin.H{
			"scanned_at": idx.ScannedAt, "total": idx.Total, "with_imdb": idx.WithIMDb,
		}
	}
	c.JSON(http.StatusOK, out)
}

func (s *Service) requireConn(c *gin.Context) (*jellyfin.Connection, bool) {
	conn := s.connection()
	if conn == nil {
		writeErr(c, http.StatusConflict, "config", "not connected to Jellyfin",
			"POST /jellyfin/v1/connect with url and api_key first, or pass them on this request.")
		return nil, false
	}
	return conn, true
}

// resolveJFConn returns an ephemeral connection from request credentials, or
// the session connection when url/api_key are omitted.
func (s *Service) resolveJFConn(c *gin.Context, url, apiKey string) (*jellyfin.Connection, bool, error) {
	if strings.TrimSpace(url) != "" || strings.TrimSpace(apiKey) != "" {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		conn, err := jellyfin.Connect(ctx, jellyfin.Credentials{URL: url, APIKey: apiKey})
		if err != nil {
			return nil, false, err
		}
		return conn, false, nil // ephemeral — do not cache index on session
	}
	conn, ok := s.requireConn(c)
	return conn, ok, nil
}

func (s *Service) handleJFScan(c *gin.Context) {
	var req struct {
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
	}
	_ = c.ShouldBindJSON(&req)

	conn, session, err := s.resolveJFConn(c, req.URL, req.APIKey)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	if conn == nil {
		return
	}

	job := s.Jobs.Create("Scanning Jellyfin library")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		idx, err := conn.Client.ScanLibrary(ctx, conn.User.ID, func(fetched, total int) {
			job.Progress("Scanning Jellyfin library", fetched, total)
		})
		if err != nil {
			job.Fail(jobErrOf(err))
			return
		}
		if session {
			s.setIndex(idx)
		}
		job.Done(map[string]any{
			"total": idx.Total, "with_imdb": idx.WithIMDb, "scanned_at": idx.ScannedAt,
		})
	}()

	c.JSON(http.StatusAccepted, gin.H{"job_id": job.ID})
}

func (s *Service) resolveTitles(watchlistID, compareID, view string, inline []imdb.Title) ([]imdb.Title, error) {
	switch {
	case len(inline) > 0:
		return inline, nil
	case watchlistID != "":
		wl, ok := s.Store.Watchlist(watchlistID)
		if !ok {
			return nil, errors.New("that watchlist is no longer loaded")
		}
		return wl.Titles, nil
	case compareID != "":
		res, ok := s.Store.Compare(compareID)
		if !ok {
			return nil, errors.New("that comparison is no longer loaded")
		}
		titles, ok := res.View(view)
		if !ok {
			return nil, errors.New("unknown view " + view)
		}
		return titles, nil
	}
	return nil, errors.New("nothing to match — pass titles, watchlist_id, or compare_id")
}

func (s *Service) libraryIndex(ctx context.Context, conn *jellyfin.Connection, useSession bool) (*jellyfin.LibraryIndex, error) {
	if useSession {
		if idx := s.index(); idx != nil {
			return idx, nil
		}
	}
	idx, err := conn.Client.ScanLibrary(ctx, conn.User.ID, nil)
	if err != nil {
		return nil, err
	}
	if useSession {
		s.setIndex(idx)
	}
	return idx, nil
}

func (s *Service) handleJFMatch(c *gin.Context) {
	var req struct {
		URL         string       `json:"url"`
		APIKey      string       `json:"api_key"`
		WatchlistID string       `json:"watchlist_id"`
		CompareID   string       `json:"compare_id"`
		View        string       `json:"view"`
		Titles      []imdb.Title `json:"titles"`
		MoviesOnly  bool         `json:"movies_only"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", "malformed request", "")
		return
	}

	conn, session, err := s.resolveJFConn(c, req.URL, req.APIKey)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	if conn == nil {
		return
	}

	titles, err := s.resolveTitles(req.WatchlistID, req.CompareID, req.View, req.Titles)
	if err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", err.Error(), "")
		return
	}
	if req.MoviesOnly {
		titles = onlyMovies(titles)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute)
	defer cancel()

	idx, err := s.libraryIndex(ctx, conn, session)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	c.JSON(http.StatusOK, idx.Match(titles))
}

func (s *Service) handleJFSync(c *gin.Context) {
	var req struct {
		URL          string       `json:"url"`
		APIKey       string       `json:"api_key"`
		WatchlistID  string       `json:"watchlist_id"`
		CompareID    string       `json:"compare_id"`
		View         string       `json:"view"`
		Titles       []imdb.Title `json:"titles"`
		MoviesOnly   bool         `json:"movies_only"`
		PlaylistName string       `json:"playlist_name"`
		Mode         string       `json:"mode"`
		Public       bool         `json:"public"`
		Confirm      bool         `json:"confirm"`
		ItemIDs      []string     `json:"item_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeErr(c, http.StatusBadRequest, "invalid_input", "malformed request", "")
		return
	}

	conn, session, err := s.resolveJFConn(c, req.URL, req.APIKey)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	if conn == nil {
		return
	}
	if strings.TrimSpace(req.PlaylistName) == "" {
		writeErr(c, http.StatusBadRequest, "invalid_input", "give the playlist a name", "")
		return
	}

	mode := jellyfin.SyncMode(req.Mode)
	switch mode {
	case jellyfin.ModeCreate, jellyfin.ModeAppend, jellyfin.ModeReplace:
	case "":
		mode = jellyfin.ModeAppend
	default:
		writeErr(c, http.StatusBadRequest, "invalid_input", "unknown sync mode "+req.Mode, "")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute)
	defer cancel()

	itemIDs := req.ItemIDs
	var match *jellyfin.MatchResult
	if len(itemIDs) == 0 {
		titles, err := s.resolveTitles(req.WatchlistID, req.CompareID, req.View, req.Titles)
		if err != nil {
			writeErr(c, http.StatusBadRequest, "invalid_input", err.Error(), "")
			return
		}
		if req.MoviesOnly {
			titles = onlyMovies(titles)
		}
		idx, err := s.libraryIndex(ctx, conn, session)
		if err != nil {
			writeDomainErr(c, err)
			return
		}
		match = idx.Match(titles)
		for _, m := range match.Matched {
			itemIDs = append(itemIDs, m.ItemID)
		}
	}

	syncReq := jellyfin.SyncRequest{
		Name: req.PlaylistName, Mode: mode, Public: req.Public, ItemIDs: itemIDs,
	}

	if !req.Confirm {
		plan, err := conn.Client.PlanSync(ctx, conn.User.ID, syncReq)
		if err != nil {
			writeDomainErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"dry_run": true, "plan": plan, "match": match})
		return
	}

	res, err := conn.Client.Sync(ctx, conn.User.ID, syncReq)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"dry_run": false, "result": res, "match": match})
}

func (s *Service) handleJFPlaylists(c *gin.Context) {
	url := c.Query("url")
	apiKey := c.Query("api_key")
	name := c.Query("name")

	conn, _, err := s.resolveJFConn(c, url, apiKey)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	if conn == nil {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	if name != "" {
		pl, err := conn.Client.FindPlaylist(ctx, conn.User.ID, name)
		if err != nil {
			writeDomainErr(c, err)
			return
		}
		if pl == nil {
			writeErr(c, http.StatusNotFound, "not_found", "no playlist named "+name, "")
			return
		}
		c.JSON(http.StatusOK, pl)
		return
	}

	items, err := conn.Client.ListPlaylists(ctx, conn.User.ID)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(items), "playlists": items})
}

func (s *Service) handleJFPlaylistItems(c *gin.Context) {
	url := c.Query("url")
	apiKey := c.Query("api_key")
	playlistID := c.Query("id")
	if playlistID == "" {
		writeErr(c, http.StatusBadRequest, "invalid_input", "playlist id is required", "")
		return
	}

	conn, _, err := s.resolveJFConn(c, url, apiKey)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	if conn == nil {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	ids, err := conn.Client.PlaylistItemIDs(ctx, playlistID, conn.User.ID)
	if err != nil {
		writeDomainErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"playlist_id": playlistID, "count": len(ids), "item_ids": ids})
}

// --- jobs --------------------------------------------------------------------

func (s *Service) handleJobStatus(c *gin.Context) {
	job, ok := s.Jobs.Get(c.Param("id"))
	if !ok {
		writeErr(c, http.StatusNotFound, "not_found", "unknown job", "")
		return
	}
	c.JSON(http.StatusOK, job.Snapshot())
}

func (s *Service) handleJobEvents(c *gin.Context) {
	job, ok := s.Jobs.Get(c.Param("id"))
	if !ok {
		writeErr(c, http.StatusNotFound, "not_found", "unknown job", "")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	updates, unsubscribe := job.Subscribe()
	defer unsubscribe()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case u, open := <-updates:
			if !open {
				return
			}
			data, err := json.Marshal(u)
			if err != nil {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
			if u.Finished {
				return
			}
		case <-ticker.C:
			fmt.Fprint(c.Writer, ": keepalive\n\n")
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

// --- helpers -----------------------------------------------------------------

func onlyMovies(in []imdb.Title) []imdb.Title {
	out := make([]imdb.Title, 0, len(in))
	for _, t := range in {
		switch t.Type {
		case "movie", "tvMovie", "video", "short":
			out = append(out, t)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func safeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "watchlist"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "watchlist"
	}
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}
