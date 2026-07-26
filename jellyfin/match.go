package jellyfin

import (
	"strings"
	"unicode"

	"earapi/imdb"
)

// MatchMethod records how confident a match is, so the UI can flag the fuzzy ones.
type MatchMethod string

const (
	MatchIMDb      MatchMethod = "imdb"       // provider id — exact
	MatchTitleYear MatchMethod = "title_year" // normalized title + year — fuzzy
)

// Match pairs a watchlist title with a library item.
type Match struct {
	Title      imdb.Title  `json:"title"`
	ItemID     string      `json:"item_id"`
	ItemName   string      `json:"item_name"`
	ItemYear   int         `json:"item_year"`
	Method     MatchMethod `json:"method"`
	Confidence string      `json:"confidence"` // exact | probable
}

// MatchResult is the outcome of matching a whole list.
type MatchResult struct {
	Matched   []Match      `json:"matched"`
	Unmatched []imdb.Title `json:"unmatched"`
	Stats     MatchStats   `json:"stats"`
}

// MatchStats summarises a match run.
type MatchStats struct {
	Total       int `json:"total"`
	Matched     int `json:"matched"`
	Unmatched   int `json:"unmatched"`
	ByIMDb      int `json:"by_imdb"`
	ByTitleYear int `json:"by_title_year"`
	LibrarySize int `json:"library_size"`
	LibraryIMDb int `json:"library_with_imdb"`
}

// articles are stripped from the front of a title before comparison, since
// libraries and IMDb disagree about "The"/"A" often enough to matter.
var articles = []string{"the ", "a ", "an ", "le ", "la ", "les ", "el ", "der ", "die ", "das "}

// normalizeTitle folds a title into a comparison key: lowercase, no diacritics,
// no punctuation, no leading article, collapsed whitespace.
func normalizeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if d, ok := foldDiacritic(r); ok {
			b.WriteString(d)
			continue
		}
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == ':' || r == '.':
			b.WriteRune(' ')
			// everything else (punctuation, quotes) is dropped
		}
	}
	s = strings.Join(strings.Fields(b.String()), " ")

	for _, a := range articles {
		if strings.HasPrefix(s, a) {
			s = s[len(a):]
			break
		}
	}
	return strings.TrimSpace(s)
}

// diacritics maps accented Latin letters onto ASCII. Covers the Latin-1 and
// Latin Extended-A characters that turn up in film titles (Szelíd, Amélie,
// Rashômon, …). Some fold to more than one letter, so the values are strings.
var diacritics = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ė': "e", 'ę': "e", 'ě': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i", 'į': "i", 'ı': "i",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ő': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ů': "u", 'ű': "u",
	'ý': "y", 'ÿ': "y",
	'ñ': "n", 'ń': "n", 'ň': "n",
	'ç': "c", 'ć': "c", 'č': "c",
	'š': "s", 'ś': "s", 'ş': "s",
	'ž': "z", 'ź': "z", 'ż': "z",
	'ğ': "g", 'ģ': "g",
	'ł': "l", 'ļ': "l",
	'ř': "r", 'ť': "t", 'ď': "d", 'đ': "d",
	'æ': "ae", 'œ': "oe", 'ß': "ss", 'ð': "th", 'þ': "th",
}

func foldDiacritic(r rune) (string, bool) {
	s, ok := diacritics[r]
	return s, ok
}

// yearsCompatible allows ±1, since release-year disagreements between IMDb and
// a library's metadata source are common around festival/wide-release splits.
func yearsCompatible(a, b int) bool {
	if a == 0 || b == 0 {
		return true // no year on one side — don't reject on that alone
	}
	d := a - b
	return d >= -1 && d <= 1
}

// Match resolves titles against the library index.
//
// Order matters: the IMDb provider id is authoritative and tried first; the
// title+year fallback only runs when there is no id match, and everything it
// finds is flagged "probable" so a human can review it before a playlist write.
func (idx *LibraryIndex) Match(titles []imdb.Title) *MatchResult {
	// Non-nil slices: these serialise to JSON and the UI calls .filter/.length
	// on them directly.
	res := &MatchResult{
		Matched:   []Match{},
		Unmatched: []imdb.Title{},
		Stats: MatchStats{
			Total:       len(titles),
			LibrarySize: idx.Total,
			LibraryIMDb: idx.WithIMDb,
		},
	}
	used := make(map[string]bool, len(titles)) // one library item per playlist entry

	for _, t := range titles {
		if it, ok := idx.byIMDb[strings.ToLower(t.IMDbID)]; ok && !used[it.ID] {
			used[it.ID] = true
			res.Matched = append(res.Matched, Match{
				Title: t, ItemID: it.ID, ItemName: it.Name, ItemYear: it.ProductionYear,
				Method: MatchIMDb, Confidence: "exact",
			})
			res.Stats.ByIMDb++
			continue
		}

		if it, ok := idx.matchByTitle(t, used); ok {
			used[it.ID] = true
			res.Matched = append(res.Matched, Match{
				Title: t, ItemID: it.ID, ItemName: it.Name, ItemYear: it.ProductionYear,
				Method: MatchTitleYear, Confidence: "probable",
			})
			res.Stats.ByTitleYear++
			continue
		}

		res.Unmatched = append(res.Unmatched, t)
	}

	res.Stats.Matched = len(res.Matched)
	res.Stats.Unmatched = len(res.Unmatched)
	return res
}

// matchByTitle tries the primary title then the original title (e.g. Gentle,
// then Szelíd), requiring a compatible year in both cases.
func (idx *LibraryIndex) matchByTitle(t imdb.Title, used map[string]bool) (Item, bool) {
	for _, name := range []string{t.Title, t.OrigTitle} {
		key := normalizeTitle(name)
		if key == "" {
			continue
		}
		for _, it := range idx.byTitle[key] {
			if used[it.ID] || !yearsCompatible(t.Year, it.ProductionYear) {
				continue
			}
			return it, true
		}
	}
	return Item{}, false
}
