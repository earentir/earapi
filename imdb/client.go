package imdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	endpoint     = "https://api.graphql.imdb.com/"
	pageSize     = 250
	maxRetries   = 5
	maxTitleBulk = 25 // aliased titles per hydrate request
)

// userAgent identifies the app to IMDb. Not required for access (the endpoint
// answers anonymous requests), but sending it is the polite thing to do.
const userAgent = "earapi/1.0 (+https://github.com/earentir/earapi)"

// Progress reports paging progress for long fetches.
type Progress func(fetched, total int)

// Client talks to IMDb's public GraphQL endpoint.
//
// A token bucket keeps us well under any rate limit: IMDb publishes none, so we
// self-limit conservatively rather than discover the ceiling the hard way.
type Client struct {
	HTTP    *http.Client
	limiter <-chan time.Time

	// Pages renders JS-protected pages. Set by the server when a browser is
	// available; nil disables p.* alias resolution.
	Pages PageFetcher
}

// NewClient returns a client rate-limited to ~4 requests/second.
func NewClient() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		limiter: time.Tick(250 * time.Millisecond),
	}
}

func (c *Client) wait(ctx context.Context) error {
	select {
	case <-c.limiter:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// post executes one GraphQL request with retry/backoff, decoding into out.
func (c *Client) post(ctx context.Context, body gqlRequest, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return newErr(ErrKindInvalidInput, "could not encode request", "", err)
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			// Exponential backoff with jitter, capped at ~8s.
			d := time.Duration(1<<attempt) * 250 * time.Millisecond
			if d > 8*time.Second {
				d = 8 * time.Second
			}
			d += time.Duration(rand.Int63n(int64(250 * time.Millisecond)))
			select {
			case <-time.After(d):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if err := c.wait(ctx); err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
		if err != nil {
			return newErr(ErrKindTransport, "could not build request", "", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			continue // transient network failure — retry
		}

		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		switch {
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = fmt.Errorf("imdb returned HTTP %d", resp.StatusCode)
			continue // retryable
		case resp.StatusCode != http.StatusOK:
			return newErr(ErrKindUpstream,
				fmt.Sprintf("IMDb returned HTTP %d", resp.StatusCode), "", nil)
		}

		if err := json.Unmarshal(data, out); err != nil {
			return newErr(ErrKindUpstream, "could not decode IMDb's response", "", err)
		}
		return nil
	}
	return newErr(ErrKindTransport, "could not reach IMDb after several attempts",
		"Check your internet connection and try again.", lastErr)
}

// classify maps GraphQL errors onto our error kinds so the UI can react.
func classify(errs []gqlError, ref ListRef) error {
	if len(errs) == 0 {
		return nil
	}
	first := errs[0]
	code := first.Extensions.Code
	msg := first.Message

	switch {
	case code == "FORBIDDEN" || strings.Contains(msg, "Permission denied"):
		return newErr(ErrKindPrivate, "that list is private",
			"Open the list on IMDb and set its visibility to public, then try again.", nil)
	case code == "RESOURCE_NOT_FOUND" || strings.Contains(msg, "Not found"):
		return newErr(ErrKindNotFound, "no list found for "+ref.ID,
			"Double-check the link — the id may be wrong or the list may have been deleted.", nil)
	case code == "BAD_USER_INPUT" && strings.Contains(msg, "Invalid userId"):
		// Reachable if a caller hand-builds a ListRef; ParseRef normally catches
		// p.* aliases before we ever get here.
		return newErr(ErrKindProfileAlias, "IMDb rejected that profile id", aliasHint, nil)
	case code == "BAD_USER_INPUT":
		return newErr(ErrKindInvalidInput, "IMDb rejected that id: "+msg, "", nil)
	}
	return newErr(ErrKindUpstream, "IMDb returned an error: "+msg, "", nil)
}

// FetchList retrieves every title on ref, following cursor pagination.
// progress may be nil.
func (c *Client) FetchList(ctx context.Context, ref ListRef, progress Progress) (*Watchlist, error) {
	switch ref.Kind {
	case KindWatchlist, KindCustom:
	case KindAlias:
		// p.* links carry no usable id. Render the page, read the real list
		// identity out of it, then continue on the normal API path.
		resolved, ids, err := c.ResolveAlias(ctx, ref.ID)
		if err != nil {
			return nil, err
		}
		if resolved.Kind == KindWatchlist || resolved.Kind == KindCustom {
			wl, err := c.FetchList(ctx, resolved, progress)
			if err != nil {
				return nil, err
			}
			wl.Source = ref // keep the link the user actually pasted
			return wl, nil
		}
		// No id on the page, but it did list titles — hydrate those instead.
		titles, err := c.FetchTitles(ctx, ids, progress)
		if err != nil {
			return nil, err
		}
		return &Watchlist{
			Source: ref, Name: resolved.Label, FetchedAt: time.Now().UTC(),
			Count: len(titles), Titles: titles,
		}, nil
	default:
		return nil, newErr(ErrKindInvalidInput,
			"that source can't be fetched from IMDb", "", nil)
	}

	wl := &Watchlist{Source: ref, FetchedAt: time.Now().UTC(), Titles: []Title{}}
	var cursor string
	seen := make(map[string]bool)

	for {
		vars := map[string]any{"id": ref.ID, "first": pageSize}
		if cursor != "" {
			vars["after"] = cursor
		}
		query := qWatchlist
		if ref.Kind == KindCustom {
			query = qList
		}

		var resp listResponse
		if err := c.post(ctx, gqlRequest{Query: query, Variables: vars}, &resp); err != nil {
			return nil, err
		}
		if err := classify(resp.Errors, ref); err != nil {
			return nil, err
		}

		var conn itemsConn
		switch ref.Kind {
		case KindWatchlist:
			if resp.Data.PredefinedList == nil {
				return nil, newErr(ErrKindNotFound, "no watchlist found for "+ref.ID,
					"Check that the id is right and the watchlist is public.", nil)
			}
			conn = resp.Data.PredefinedList.Items
		case KindCustom:
			if resp.Data.List == nil {
				return nil, newErr(ErrKindNotFound, "no list found for "+ref.ID,
					"Check that the id is right and the list is public.", nil)
			}
			conn = resp.Data.List.Items
			if wl.Name == "" {
				wl.Name = resp.Data.List.Name.OriginalText
			}
		}

		for _, e := range conn.Edges {
			raw := e.Node.Item
			// Non-Title entries (people, images) come back as empty objects from
			// the inline fragment; skip them and any duplicate ids.
			if raw.ID == "" || seen[raw.ID] {
				continue
			}
			seen[raw.ID] = true
			wl.Titles = append(wl.Titles, raw.toTitle())
		}

		if progress != nil {
			progress(len(wl.Titles), conn.Total)
		}
		if !conn.PageInfo.HasNextPage || conn.PageInfo.EndCursor == "" || len(conn.Edges) == 0 {
			break
		}
		cursor = conn.PageInfo.EndCursor
	}

	wl.Count = len(wl.Titles)
	if wl.Name == "" {
		wl.Name = ref.Label
	}
	return wl, nil
}

// buildTitlesQuery builds an aliased bulk lookup: t0: title(id:"tt…"){…} …
// Used by the CSV path, which starts with ids and no metadata.
func buildTitlesQuery(ids []string) string {
	var b strings.Builder
	b.WriteString(qTitlesHeader)
	for i, id := range ids {
		fmt.Fprintf(&b, "\n t%d: title(id: %q) {%s}", i, id, titleFields)
	}
	b.WriteString("\n}")
	return b.String()
}

// FetchTitles hydrates metadata for the given title ids, in batches.
// Ids that IMDb doesn't recognise are skipped rather than failing the batch.
func (c *Client) FetchTitles(ctx context.Context, ids []string, progress Progress) ([]Title, error) {
	out := make([]Title, 0, len(ids))
	for start := 0; start < len(ids); start += maxTitleBulk {
		end := min(start+maxTitleBulk, len(ids))
		batch := ids[start:end]

		var resp struct {
			Data   map[string]*rawTitle `json:"data"`
			Errors []gqlError           `json:"errors"`
		}
		if err := c.post(ctx, gqlRequest{Query: buildTitlesQuery(batch)}, &resp); err != nil {
			return nil, err
		}
		// Partial errors are expected when an id is bad: data still carries the
		// aliases that resolved, so only bail if nothing came back at all.
		if len(resp.Data) == 0 && len(resp.Errors) > 0 {
			return nil, classify(resp.Errors, ListRef{Kind: KindCSV})
		}

		for i := range batch {
			if rt := resp.Data[fmt.Sprintf("t%d", i)]; rt != nil && rt.ID != "" {
				out = append(out, rt.toTitle())
			}
		}
		if progress != nil {
			progress(len(out), len(ids))
		}
	}
	return out, nil
}
