// Package imdb reads IMDb watchlists and custom lists.
//
// IMDb's HTML is served behind an AWS WAF challenge and cannot be scraped, but
// the public GraphQL endpoint at api.graphql.imdb.com answers unauthenticated
// requests. All list reads therefore go through GraphQL; see queries.go.
//
// IMDb data obtained this way is licensed for limited, non-commercial, private
// use only. See https://help.imdb.com/article/imdb/general-information/G5JTRESSHJBBHTGX
package imdb

import (
	"fmt"
	"strings"
	"time"
)

// ListKind distinguishes the three supported ways of naming a list.
type ListKind string

const (
	KindWatchlist ListKind = "watchlist" // ur… profile watchlist
	KindCustom    ListKind = "list"      // ls… custom/shared list
	KindCSV       ListKind = "csv"       // uploaded IMDb export
	KindAlias     ListKind = "alias"     // p.… profile link, resolved via a browser
)

// ListRef is a normalized reference to something we can fetch.
type ListRef struct {
	Kind  ListKind `json:"kind"`
	ID    string   `json:"id"`    // "ur12345678" | "ls123456789" | "" for CSV
	Label string   `json:"label"` // human-facing description
}

func (r ListRef) String() string {
	if r.ID == "" {
		return string(r.Kind)
	}
	return string(r.Kind) + ":" + r.ID
}

// Title is one entry on a list, flattened from the GraphQL response.
type Title struct {
	IMDbID     string   `json:"imdb_id"`
	Title      string   `json:"title"`
	OrigTitle  string   `json:"original_title,omitempty"`
	Year       int      `json:"year,omitempty"`
	Type       string   `json:"type"` // movie, tvSeries, tvMiniSeries, …
	RuntimeSec int      `json:"runtime_seconds,omitempty"`
	Rating     float64  `json:"rating,omitempty"`
	Votes      int      `json:"votes,omitempty"`
	Genres     []string `json:"genres,omitempty"`
	Plot       string   `json:"plot,omitempty"`
	PosterURL  string   `json:"poster_url,omitempty"`
	IMDbURL    string   `json:"imdb_url"`
}

// Watchlist is a fetched list plus provenance.
type Watchlist struct {
	ID        string    `json:"id"` // internal handle used by the HTTP API
	Source    ListRef   `json:"source"`
	Owner     string    `json:"owner"`
	Name      string    `json:"name,omitempty"` // list title, when IMDb exposes one
	FetchedAt time.Time `json:"fetched_at"`
	Count     int       `json:"count"`
	Titles    []Title   `json:"titles"`
}

// ErrKind classifies failures so the UI can respond differently to each.
type ErrKind string

const (
	ErrKindInvalidInput ErrKind = "invalid_input"
	ErrKindProfileAlias ErrKind = "profile_alias" // p.* link — needs CSV or ur id
	ErrKindNotFound     ErrKind = "not_found"
	ErrKindPrivate      ErrKind = "private"
	ErrKindUpstream     ErrKind = "upstream"
	ErrKindTransport    ErrKind = "transport"
)

// Error is a classified failure carrying UI-ready guidance.
type Error struct {
	Kind    ErrKind `json:"kind"`
	Message string  `json:"message"`
	Hint    string  `json:"hint,omitempty"`
	err     error
}

func (e *Error) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.err)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error { return e.err }

func newErr(kind ErrKind, msg, hint string, wrapped error) *Error {
	return &Error{Kind: kind, Message: msg, Hint: hint, err: wrapped}
}

// TitleURL builds the canonical IMDb page URL for a title id.
func TitleURL(id string) string { return "https://www.imdb.com/title/" + id + "/" }

// Runtime renders seconds as "1h 32m" for display.
func (t Title) Runtime() string {
	if t.RuntimeSec <= 0 {
		return ""
	}
	d := time.Duration(t.RuntimeSec) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// GenreString joins genres for CSV export.
func (t Title) GenreString() string { return strings.Join(t.Genres, ", ") }
