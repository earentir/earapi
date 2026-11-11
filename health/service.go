package health

import (
	"database/sql"
	"sync"
)

// Config for the health API service (never includes auth user list).
type Config struct {
	DataDir   string
	MDDataDir string
}

// Service holds the shared DB and per-user stores.
type Service struct {
	Config    Config
	authUsers []UserCred
	db        *sql.DB

	mu     sync.Mutex
	stores map[string]*Store
}

// New creates a health service and opens the shared SQLite database.
func New(cfg Config, authUsers []UserCred) (*Service, error) {
	if cfg.DataDir == "" {
		cfg.DataDir = "healthdata"
	}
	if cfg.MDDataDir == "" {
		cfg.MDDataDir = "mddata"
	}
	db, err := OpenSharedDB(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	return &Service{
		Config:    cfg,
		authUsers: authUsers,
		db:        db,
		stores:    make(map[string]*Store),
	}, nil
}

// StoreForUser returns the authenticated user's store.
func (s *Service) StoreForUser(username string) (*Store, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if st, ok := s.stores[username]; ok {
		return st, nil
	}
	st, err := NewStore(s.db, s.Config.DataDir, username)
	if err != nil {
		return nil, err
	}
	s.stores[username] = st
	return st, nil
}
