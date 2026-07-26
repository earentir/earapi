package imdb

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	reUser    = regexp.MustCompile(`^ur\d{4,12}$`)
	reList    = regexp.MustCompile(`^ls\d{6,12}$`)
	reTitle   = regexp.MustCompile(`^tt\d{6,10}$`)
	reAlias   = regexp.MustCompile(`^p\.[A-Za-z0-9._-]{6,}$`)
	reAnyUser = regexp.MustCompile(`\bur\d{4,12}\b`)
	reAnyList = regexp.MustCompile(`\bls\d{6,12}\b`)
)

// aliasHint is shown only when a p.* link cannot be resolved after all, which
// normally means no browser is installed. These ids are opaque hashes that
// IMDb's GraphQL API rejects, and the page behind them is served only after an
// AWS WAF JavaScript challenge — so resolving one means rendering it in a real
// browser (see alias.go). Where that isn't possible, the CSV export still is.
const aliasHint = "Anonymised IMDb profile links (p.…) have to be opened in a browser. " +
	"Install Google Chrome, Chromium, Edge or Brave and try again — or open your " +
	"watchlist on IMDb, use Export, and drop the CSV here instead."

// ParseRef normalizes user input into a ListRef.
//
// Accepted:
//
//	https://www.imdb.com/user/ur12345678/watchlist/   → watchlist
//	ur12345678                                        → watchlist
//	https://www.imdb.com/list/ls123456789/            → custom list
//	ls123456789                                       → custom list
//	https://www.imdb.com/user/p.abc123…/watchlist/     → alias, resolved via a browser
func ParseRef(input string) (ListRef, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return ListRef{}, newErr(ErrKindInvalidInput, "no watchlist link or id given", "", nil)
	}

	// Bare ids first — cheapest and unambiguous.
	switch {
	case reUser.MatchString(s):
		return ListRef{Kind: KindWatchlist, ID: s, Label: "Watchlist " + s}, nil
	case reList.MatchString(s):
		return ListRef{Kind: KindCustom, ID: s, Label: "List " + s}, nil
	case reAlias.MatchString(s):
		return ListRef{Kind: KindAlias, ID: s, Label: "Watchlist " + s}, nil
	}

	// Otherwise treat it as a URL (tolerating a missing scheme).
	probe := s
	if !strings.Contains(probe, "://") {
		probe = "https://" + probe
	}
	u, err := url.Parse(probe)
	if err != nil {
		return ListRef{}, newErr(ErrKindInvalidInput, "that doesn't look like an IMDb link or id", "", err)
	}
	if h := strings.ToLower(u.Hostname()); h != "" && !strings.HasSuffix(h, "imdb.com") {
		return ListRef{}, newErr(ErrKindInvalidInput, "only imdb.com links are supported", "", nil)
	}

	path := u.Path
	// Check the alias before the generic id sweep: an alias URL contains no ur/ls
	// id of its own, and must be routed to browser-backed resolution.
	for _, seg := range strings.Split(path, "/") {
		if reAlias.MatchString(seg) {
			return ListRef{Kind: KindAlias, ID: seg, Label: "Watchlist " + seg}, nil
		}
	}
	if m := reAnyList.FindString(path); m != "" {
		return ListRef{Kind: KindCustom, ID: m, Label: "List " + m}, nil
	}
	if m := reAnyUser.FindString(path); m != "" {
		return ListRef{Kind: KindWatchlist, ID: m, Label: "Watchlist " + m}, nil
	}

	return ListRef{}, newErr(ErrKindInvalidInput,
		"couldn't find a watchlist (ur…) or list (ls…) id in that link", "", nil)
}

// IsTitleID reports whether s is a well-formed IMDb title id.
func IsTitleID(s string) bool { return reTitle.MatchString(strings.TrimSpace(s)) }
