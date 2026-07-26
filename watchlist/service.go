// Package watchlist exposes IMDb watchlist parsing, list comparison, and
// Jellyfin playlist management as HTTP API groups under earapi.
package watchlist

import (
	"sync"
	"time"

	"earapi/browser"
	"earapi/imdb"
	"earapi/jellyfin"
)

// Config controls cache and optional browser-backed alias resolution.
type Config struct {
	CacheDir     string
	CacheTTL     time.Duration // 0 disables disk cache
	BrowserPath  string        // optional override; empty = auto-discover
	BrowserHeadful bool
}

// Service wires IMDb client, in-memory store, jobs, and Jellyfin session state.
type Service struct {
	Store *Store
	Jobs  *Jobs
	IMDb  *imdb.Client

	BrowserName string // empty when p.* alias resolution is unavailable

	jfMu   sync.RWMutex
	jfConn *jellyfin.Connection
	jfIdx  *jellyfin.LibraryIndex
}

// New builds a Service. A missing browser is non-fatal (alias links need CSV).
// CacheTTL of 0 disables the on-disk IMDb list cache.
func New(cfg Config) *Service {
	svc := &Service{
		Store: NewStore(cfg.CacheDir, cfg.CacheTTL),
		Jobs:  NewJobs(),
		IMDb:  imdb.NewClient(),
	}

	r, err := browser.New("")
	if err == nil {
		if cfg.BrowserPath != "" {
			r.ExecPath = cfg.BrowserPath
		}
		r.Headless = !cfg.BrowserHeadful
		svc.IMDb.Pages = r
		svc.BrowserName = r.Name()
	}
	return svc
}

func (s *Service) setConnection(conn *jellyfin.Connection) {
	s.jfMu.Lock()
	s.jfConn = conn
	s.jfIdx = nil
	s.jfMu.Unlock()
}

func (s *Service) clearConnection() {
	s.jfMu.Lock()
	s.jfConn, s.jfIdx = nil, nil
	s.jfMu.Unlock()
}

func (s *Service) connection() *jellyfin.Connection {
	s.jfMu.RLock()
	defer s.jfMu.RUnlock()
	return s.jfConn
}

func (s *Service) setIndex(idx *jellyfin.LibraryIndex) {
	s.jfMu.Lock()
	s.jfIdx = idx
	s.jfMu.Unlock()
}

func (s *Service) index() *jellyfin.LibraryIndex {
	s.jfMu.RLock()
	defer s.jfMu.RUnlock()
	return s.jfIdx
}
