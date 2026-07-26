package jellyfin

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const scanPageSize = 500

// Item is the subset of BaseItemDto we index.
type Item struct {
	ID             string            `json:"Id"`
	Name           string            `json:"Name"`
	OriginalTitle  string            `json:"OriginalTitle"`
	ProductionYear int               `json:"ProductionYear"`
	Type           string            `json:"Type"` // Movie | Series
	ProviderIds    map[string]string `json:"ProviderIds"`
}

// imdbID returns the item's IMDb id, tolerating provider-key casing differences
// (Jellyfin has shipped both "Imdb" and "IMDB" over the years).
func (i Item) imdbID() string {
	for k, v := range i.ProviderIds {
		if strings.EqualFold(k, "imdb") {
			return strings.ToLower(strings.TrimSpace(v))
		}
	}
	return ""
}

// LibraryIndex supports fast local lookup by IMDb id, then by title+year.
type LibraryIndex struct {
	ScannedAt time.Time `json:"scanned_at"`
	Total     int       `json:"total"`
	WithIMDb  int       `json:"with_imdb"`

	byIMDb  map[string]Item
	byTitle map[string][]Item // normalized title -> items (any year)
}

// Progress reports scan progress.
type Progress func(fetched, total int)

// ScanLibrary walks the whole library once and builds the index.
//
// There is no server-side provider-id filter in the stable API, so everything
// is pulled with fields=ProviderIds and matched locally.
func (c *Client) ScanLibrary(ctx context.Context, userID string, progress Progress) (*LibraryIndex, error) {
	idx := &LibraryIndex{
		ScannedAt: time.Now().UTC(),
		byIMDb:    map[string]Item{},
		byTitle:   map[string][]Item{},
	}

	for start := 0; ; start += scanPageSize {
		q := url.Values{}
		q.Set("userId", userID)
		q.Set("recursive", "true")
		q.Set("includeItemTypes", "Movie,Series")
		q.Set("fields", "ProviderIds,ProductionYear,OriginalTitle")
		q.Set("startIndex", strconv.Itoa(start))
		q.Set("limit", strconv.Itoa(scanPageSize))
		q.Set("enableTotalRecordCount", "true")
		q.Set("enableImages", "false")
		q.Set("sortBy", "SortName")

		var page struct {
			Items            []Item `json:"Items"`
			TotalRecordCount int    `json:"TotalRecordCount"`
		}
		if err := c.do(ctx, http.MethodGet, "/Items", q, nil, &page); err != nil {
			return nil, err
		}

		for _, it := range page.Items {
			if it.ID == "" {
				continue
			}
			idx.Total++
			if id := it.imdbID(); id != "" {
				idx.WithIMDb++
				// First writer wins: duplicates (multiple editions of one film)
				// would otherwise thrash the index.
				if _, seen := idx.byIMDb[id]; !seen {
					idx.byIMDb[id] = it
				}
			}
			for _, k := range []string{normalizeTitle(it.Name), normalizeTitle(it.OriginalTitle)} {
				if k != "" {
					idx.byTitle[k] = append(idx.byTitle[k], it)
				}
			}
		}

		if progress != nil {
			progress(idx.Total, page.TotalRecordCount)
		}
		if len(page.Items) < scanPageSize || idx.Total >= page.TotalRecordCount {
			break
		}
	}
	return idx, nil
}
