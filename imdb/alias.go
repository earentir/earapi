package imdb

import (
	"context"
	"regexp"
	"strings"
)

// Anonymised p.* profile links are rejected outright by IMDb's GraphQL API, and
// the page behind them sits behind an AWS WAF JavaScript challenge that no HTTP
// client can solve. The way through is to render the page in a real browser and
// pull the underlying list identity out of it.
//
// Once we have a ur…/ls… id we stop scraping immediately and switch to the
// GraphQL API: it paginates properly and returns complete metadata, whereas the
// HTML only ever holds the first screenful of a lazily-loaded grid.

// PageFetcher renders a URL and returns the resulting DOM. Implemented by
// internal/browser; injected so alias resolution stays unit-testable.
type PageFetcher interface {
	FetchHTML(ctx context.Context, url string, ready func(html string) bool) (string, error)
}

// AliasURL builds the watchlist page URL for a p.* profile id.
func AliasURL(alias string) string {
	return "https://www.imdb.com/user/" + alias + "/watchlist/"
}

var (
	// Ordered best-first. A keyed match is far safer than a bare id, which could
	// belong to a "recommended lists" module elsewhere on the page.
	reUserKeyed = regexp.MustCompile(`"userId"\s*:\s*"(ur\d{4,12})"`)
	reUserPath  = regexp.MustCompile(`/user/(ur\d{4,12})[/"]`)
	reListKeyed = regexp.MustCompile(`"(?:listId|mainListId|id)"\s*:\s*"(ls\d{6,12})"`)
	reListPath  = regexp.MustCompile(`/list/(ls\d{6,12})[/"]`)
	reUserBare  = regexp.MustCompile(`\bur\d{4,12}\b`)
	reListBare  = regexp.MustCompile(`\bls\d{6,12}\b`)

	reNextData = regexp.MustCompile(`(?s)<script[^>]+id="__NEXT_DATA__"[^>]*>(.*?)</script>`)
	reTConst   = regexp.MustCompile(`\btt\d{7,9}\b`)

	// The WAF challenge page, so we can tell "still solving" from "solved".
	reChallenge = regexp.MustCompile(`awsWafCookieDomainList|gokuProps|challenge\.js|captcha`)
)

// ExtractRef pulls a ur…/ls… reference out of a rendered watchlist page.
func ExtractRef(html string) (ListRef, bool) {
	// Prefer the embedded JSON payload: it describes this page, whereas the
	// surrounding markup also carries navigation and recommendation links.
	scope := html
	if m := reNextData.FindStringSubmatch(html); len(m) == 2 {
		scope = m[1]
	}

	for _, try := range []struct {
		re   *regexp.Regexp
		kind ListKind
	}{
		{reUserKeyed, KindWatchlist},
		{reListKeyed, KindCustom},
		{reUserPath, KindWatchlist},
		{reListPath, KindCustom},
	} {
		if m := try.re.FindStringSubmatch(scope); len(m) == 2 {
			return refFor(try.kind, m[1]), true
		}
	}
	// Same passes again over the whole document, in case the payload was absent.
	if scope != html {
		for _, try := range []struct {
			re   *regexp.Regexp
			kind ListKind
		}{
			{reUserKeyed, KindWatchlist},
			{reListKeyed, KindCustom},
			{reUserPath, KindWatchlist},
			{reListPath, KindCustom},
		} {
			if m := try.re.FindStringSubmatch(html); len(m) == 2 {
				return refFor(try.kind, m[1]), true
			}
		}
	}
	// Last resort: any bare id at all.
	if m := reUserBare.FindString(scope); m != "" {
		return refFor(KindWatchlist, m), true
	}
	if m := reListBare.FindString(scope); m != "" {
		return refFor(KindCustom, m), true
	}
	return ListRef{}, false
}

func refFor(kind ListKind, id string) ListRef {
	if kind == KindCustom {
		return ListRef{Kind: KindCustom, ID: id, Label: "List " + id}
	}
	return ListRef{Kind: KindWatchlist, ID: id, Label: "Watchlist " + id}
}

// ExtractTitleIDs harvests tt… ids from a rendered page, in document order.
//
// This is the fallback for when no ur…/ls… id can be found. It is deliberately
// scoped to the __NEXT_DATA__ payload when present, because scanning the whole
// document would also sweep up "more like this" recommendations and silently
// pad the watchlist with films the user never added.
func ExtractTitleIDs(html string) []string {
	scope := html
	if m := reNextData.FindStringSubmatch(html); len(m) == 2 {
		scope = m[1]
	}
	seen := make(map[string]bool)
	var out []string
	for _, id := range reTConst.FindAllString(scope, -1) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// PageSolved reports whether a rendered page is past the WAF challenge and
// carries watchlist content. Used as the browser's readiness predicate.
//
// Readiness is decided purely on the signals we actually need — a resolvable
// id, or some titles. An earlier version also required the document to exceed a
// byte threshold, which rejected small-but-complete pages and made the whole
// path untestable; the signal checks below already exclude a bare stub.
func PageSolved(html string) bool {
	if strings.TrimSpace(html) == "" {
		return false
	}
	// The interstitial ships the challenge script and nothing else of use.
	if reChallenge.MatchString(html) && !strings.Contains(html, "__NEXT_DATA__") {
		return false
	}
	if _, ok := ExtractRef(html); ok {
		return true
	}
	return len(ExtractTitleIDs(html)) > 0
}

// ResolveAlias turns a p.* profile link into something fetchable, by rendering
// the page in a browser and reading the real list id out of it.
//
// Returns the resolved reference, or — when the page yields no id but does list
// titles — the title ids as a fallback (ref.Kind will be KindCSV to signal that
// the caller should hydrate them individually).
func (c *Client) ResolveAlias(ctx context.Context, alias string) (ListRef, []string, error) {
	if c.Pages == nil {
		return ListRef{}, nil, newErr(ErrKindProfileAlias,
			"anonymised IMDb profile links need a browser to open",
			"Install Google Chrome, Chromium, Edge or Brave and try again — or use "+
				"your watchlist's CSV export instead.", nil)
	}

	html, err := c.Pages.FetchHTML(ctx, AliasURL(alias), PageSolved)
	if err != nil && html == "" {
		return ListRef{}, nil, newErr(ErrKindTransport,
			"couldn't open that IMDb profile in the browser",
			"Check your connection and try again. If it keeps failing, use your "+
				"watchlist's CSV export.", err)
	}

	if ref, ok := ExtractRef(html); ok {
		return ref, nil, nil
	}
	if ids := ExtractTitleIDs(html); len(ids) > 0 {
		return ListRef{Kind: KindCSV, Label: "Watchlist " + alias}, ids, nil
	}

	if reChallenge.MatchString(html) {
		return ListRef{}, nil, newErr(ErrKindUpstream,
			"IMDb's bot check didn't complete",
			"Try again in a moment. If it keeps happening, run with -browser-headful "+
				"so you can complete the check yourself, or use the CSV export.", err)
	}
	return ListRef{}, nil, newErr(ErrKindNotFound,
		"couldn't find a watchlist on that profile page",
		"Check the profile's watchlist is public, or use the CSV export.", err)
}
