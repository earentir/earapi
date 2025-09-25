package youtube

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AddRequest struct {
    PlaylistName string `json:"playlistName"`
    Video        string `json:"video"`
    Force        bool   `json:"force"`
    User         string `json:"user"`
}

func RegisterRoutes(r *gin.Engine, svc *Service) {
    g := r.Group("/youtube/v1")
    g.POST("/playlist/add", func(c *gin.Context) {
        var req AddRequest
        if err := c.BindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
        defer cancel()
        res, err := svc.AddVideoToPlaylist(ctx, req.PlaylistName, req.Video, req.Force)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        // append log line on success
        go appendAddLog(req.PlaylistName, res.VideoID, req.User, req.Force)
        c.JSON(http.StatusOK, res)
    })

    // Create playlist (separate from add)
    g.POST("/playlist/create", func(c *gin.Context) {
        var req struct {
            Name    string `json:"name"`
            Privacy string `json:"privacy"`
        }
        if err := c.BindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
        defer cancel()
        pl, err := svc.CreatePlaylist(ctx, req.Name, req.Privacy)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"playlistId": pl.ID, "title": pl.Title})
    })

    // List items with optional fuzzy and metadata
    g.GET("/playlist/items", func(c *gin.Context) {
        name := c.Query("name")
        fuzzy := c.DefaultQuery("fuzzy", "false") == "true"
        withMeta := c.DefaultQuery("metadata", "false") == "true"
        ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
        defer cancel()
        items, pl, err := svc.ListItems(ctx, name, fuzzy)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        if withMeta {
            metaIdx := loadAdditionsIndex()
            type itemWithMeta struct {
                VideoID string `json:"videoId"`
                Title   string `json:"title"`
                Date    string `json:"date,omitempty"`
                User    string `json:"user,omitempty"`
                Playlist string `json:"playlist,omitempty"`
                Force   bool   `json:"force"`
            }
            out := make([]itemWithMeta, 0, len(items))
            for _, it := range items {
                m := metaIdx[it.VideoID]
                out = append(out, itemWithMeta{VideoID: it.VideoID, Title: it.Title, Date: m.Date, User: m.User, Playlist: m.Playlist, Force: m.Force})
            }
            c.JSON(http.StatusOK, gin.H{"playlistId": pl.ID, "title": pl.Title, "items": out})
            return
        }
        c.JSON(http.StatusOK, gin.H{"playlistId": pl.ID, "title": pl.Title, "items": items})
    })

    // Metadata for specific video+playlist
    g.GET("/playlist/video/meta", func(c *gin.Context) {
        name := c.Query("name")
        fuzzy := c.DefaultQuery("fuzzy", "true") == "true"
        videoID := c.Query("videoId")
        if name == "" || videoID == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "name and videoId are required"})
            return
        }
        ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
        defer cancel()
        _, pl, err := svc.ListItems(ctx, name, fuzzy)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        metaIdx := loadAdditionsIndex()
        m := metaIdx[videoID]
        c.JSON(http.StatusOK, gin.H{"playlistId": pl.ID, "title": pl.Title, "videoId": videoID, "date": m.Date, "user": m.User, "playlist": m.Playlist, "force": m.Force})
    })
}

func appendAddLog(playlistProvided string, videoID string, user string, force bool) {
    if videoID == "" {
        return
    }
    ts := time.Now().UTC().Format(time.RFC3339)
    line := ts + "," + playlistProvided + "," + videoID + "," + user + "," + boolToStr(force) + "\n"
    f, err := os.OpenFile("youtubedata/added.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return
    }
    defer func() { _ = f.Close() }()
    _, _ = f.WriteString(line)
}

func boolToStr(b bool) string {
    if b {
        return "true"
    }
    return "false"
}

// additionsIndex holds latest metadata keyed by video id.
type additionsIndex map[string]struct{
    Date string
    Playlist string
    User string
    Force bool
}

func loadAdditionsIndex() additionsIndex {
    f, err := os.Open("youtubedata/added.log")
    if err != nil {
        return additionsIndex{}
    }
    defer func(){ _ = f.Close() }()
    // naive read: small file assumption; for large files, stream scanner
    bs, err := os.ReadFile("youtubedata/added.log")
    if err != nil {
        return additionsIndex{}
    }
    lines := strings.Split(string(bs), "\n")
    idx := make(additionsIndex)
    for _, ln := range lines {
        if ln == "" { continue }
        parts := strings.Split(ln, ",")
        if len(parts) < 5 { continue }
        // ts, playlist, videoID, user, force
        idx[parts[2]] = struct{Date string; Playlist string; User string; Force bool}{
            Date: parts[0],
            Playlist: parts[1],
            User: parts[3],
            Force: parts[4] == "true",
        }
    }
    return idx
}


