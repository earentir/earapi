package watchlist

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"earapi/compare"
	"earapi/imdb"
)

// Store holds fetched watchlists and comparisons for the life of the process,
// plus a disk cache so repeated fetches of the same list don't hammer IMDb.
type Store struct {
	mu         sync.RWMutex
	watchlists map[string]*imdb.Watchlist
	compares   map[string]*compare.Result

	cacheDir string
	cacheTTL time.Duration
}

// NewStore creates a store backed by cacheDir.
func NewStore(cacheDir string, ttl time.Duration) *Store {
	if cacheDir != "" {
		_ = os.MkdirAll(cacheDir, 0o700)
	}
	return &Store{
		watchlists: map[string]*imdb.Watchlist{},
		compares:   map[string]*compare.Result{},
		cacheDir:   cacheDir,
		cacheTTL:   ttl,
	}
}

// NewID returns a random hex identifier.
func NewID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))[:24]
	}
	return hex.EncodeToString(b[:])
}

// PutWatchlist stores a watchlist and returns its handle.
func (s *Store) PutWatchlist(wl *imdb.Watchlist) string {
	if wl.ID == "" {
		wl.ID = NewID()
	}
	s.mu.Lock()
	s.watchlists[wl.ID] = wl
	s.mu.Unlock()
	return wl.ID
}

// Watchlist returns a stored watchlist.
func (s *Store) Watchlist(id string) (*imdb.Watchlist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wl, ok := s.watchlists[id]
	return wl, ok
}

// PutCompare stores a comparison and returns its handle.
func (s *Store) PutCompare(r *compare.Result) string {
	id := NewID()
	s.mu.Lock()
	s.compares[id] = r
	s.mu.Unlock()
	return id
}

// Compare returns a stored comparison.
func (s *Store) Compare(id string) (*compare.Result, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.compares[id]
	return r, ok
}

func (s *Store) cachePath(ref imdb.ListRef) string {
	if s.cacheDir == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ref.String()))
	return filepath.Join(s.cacheDir, hex.EncodeToString(sum[:8])+".json")
}

// CachedList returns a cached fetch if it is still within the TTL.
func (s *Store) CachedList(ref imdb.ListRef) (*imdb.Watchlist, bool) {
	path := s.cachePath(ref)
	if path == "" || s.cacheTTL <= 0 {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var wl imdb.Watchlist
	if err := json.Unmarshal(data, &wl); err != nil {
		return nil, false
	}
	if time.Since(wl.FetchedAt) > s.cacheTTL {
		return nil, false
	}
	wl.ID = ""
	return &wl, true
}

// CacheList writes a fetch to the disk cache.
func (s *Store) CacheList(ref imdb.ListRef, wl *imdb.Watchlist) {
	path := s.cachePath(ref)
	if path == "" {
		return
	}
	data, err := json.Marshal(wl)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
	}
}
