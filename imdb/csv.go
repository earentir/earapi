package imdb

import (
	"context"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"time"
)

// ImportCSV reads an IMDb export CSV and returns a Watchlist.
//
// This is the escape hatch for anonymised p.* profile links, which cannot be
// resolved through the API at all (see parse.go). IMDb's own export is available
// from the watchlist page when signed in.
//
// Columns are located by header name rather than position, because IMDb's
// watchlist, list and ratings exports each ship a different column set. Only
// "Const" (the tt… id) is required; anything else present is used as a fallback
// when hydrate is false or a title fails to hydrate.
func ImportCSV(ctx context.Context, r io.Reader, owner string, c *Client, hydrate bool) (*Watchlist, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // IMDb exports have ragged rows
	cr.LazyQuotes = true

	header, err := cr.Read()
	if err != nil {
		return nil, newErr(ErrKindInvalidInput, "that file doesn't look like a CSV",
			"Export your watchlist from IMDb and upload the .csv file unchanged.", err)
	}

	col := make(map[string]int, len(header))
	for i, h := range header {
		// Strip a UTF-8 BOM on the first header cell, else "Const" never matches.
		h = strings.TrimPrefix(h, "\ufeff")
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	constIdx, ok := col["const"]
	if !ok {
		return nil, newErr(ErrKindInvalidInput, "no \"Const\" column in that CSV",
			"Use the CSV exactly as IMDb exports it — the Const column holds the title ids.", nil)
	}

	at := func(rec []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	var (
		ids      []string
		fallback = map[string]Title{}
		seen     = map[string]bool{}
	)
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // tolerate a malformed row rather than losing the import
		}
		if constIdx >= len(rec) {
			continue
		}
		id := strings.TrimSpace(rec[constIdx])
		if !IsTitleID(id) || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)

		t := Title{
			IMDbID:    id,
			Title:     at(rec, "title"),
			OrigTitle: at(rec, "original title"),
			Type:      normalizeCSVType(at(rec, "title type")),
			IMDbURL:   TitleURL(id),
		}
		if v, err := strconv.Atoi(at(rec, "year")); err == nil {
			t.Year = v
		}
		if v, err := strconv.ParseFloat(at(rec, "imdb rating"), 64); err == nil {
			t.Rating = v
		}
		if v, err := strconv.Atoi(at(rec, "num votes")); err == nil {
			t.Votes = v
		}
		if v, err := strconv.Atoi(at(rec, "runtime (mins)")); err == nil {
			t.RuntimeSec = v * 60
		}
		if g := at(rec, "genres"); g != "" {
			for _, part := range strings.Split(g, ",") {
				if p := strings.TrimSpace(part); p != "" {
					t.Genres = append(t.Genres, p)
				}
			}
		}
		if t.OrigTitle == t.Title {
			t.OrigTitle = ""
		}
		fallback[id] = t
	}

	if len(ids) == 0 {
		return nil, newErr(ErrKindInvalidInput, "no IMDb title ids found in that CSV",
			"The Const column should contain ids like tt0111161.", nil)
	}

	wl := &Watchlist{
		Source:    ListRef{Kind: KindCSV, Label: "Imported CSV"},
		Owner:     owner,
		Name:      "Imported CSV",
		FetchedAt: time.Now().UTC(),
		Titles:    []Title{},
	}

	// Hydrating gives us posters, plots and canonical genres that the CSV lacks.
	if hydrate && c != nil {
		hydrated, err := c.FetchTitles(ctx, ids, nil)
		if err == nil {
			got := make(map[string]bool, len(hydrated))
			for _, t := range hydrated {
				got[t.IMDbID] = true
			}
			wl.Titles = hydrated
			// Keep anything IMDb wouldn't hydrate, using the CSV's own columns.
			for _, id := range ids {
				if !got[id] {
					wl.Titles = append(wl.Titles, fallback[id])
				}
			}
			wl.Count = len(wl.Titles)
			return wl, nil
		}
		// Hydration is best-effort: a network failure shouldn't lose the import.
	}

	for _, id := range ids {
		wl.Titles = append(wl.Titles, fallback[id])
	}
	wl.Count = len(wl.Titles)
	return wl, nil
}

// normalizeCSVType maps the CSV's display names onto the ids the GraphQL API uses,
// so filters behave the same whichever import path produced the list.
func normalizeCSVType(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "movie", "feature":
		return "movie"
	case "tv series", "tvseries":
		return "tvSeries"
	case "tv mini series", "tv mini-series", "tvminiseries":
		return "tvMiniSeries"
	case "tv movie", "tvmovie":
		return "tvMovie"
	case "short", "tv short":
		return "short"
	case "video":
		return "video"
	case "video game", "videogame":
		return "videoGame"
	case "tv episode", "tvepisode":
		return "tvEpisode"
	case "":
		return ""
	default:
		return strings.TrimSpace(s)
	}
}

// ExportCSV writes titles as CSV, round-tripping into other tools (including
// back into this app).
func ExportCSV(w io.Writer, titles []Title) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write([]string{
		"Const", "Title", "Original Title", "Year", "Title Type",
		"IMDb Rating", "Num Votes", "Runtime (mins)", "Genres", "URL",
	}); err != nil {
		return err
	}
	for _, t := range titles {
		runtime := ""
		if t.RuntimeSec > 0 {
			runtime = strconv.Itoa(t.RuntimeSec / 60)
		}
		rating := ""
		if t.Rating > 0 {
			rating = strconv.FormatFloat(t.Rating, 'f', 1, 64)
		}
		votes := ""
		if t.Votes > 0 {
			votes = strconv.Itoa(t.Votes)
		}
		year := ""
		if t.Year > 0 {
			year = strconv.Itoa(t.Year)
		}
		if err := cw.Write([]string{
			t.IMDbID, t.Title, t.OrigTitle, year, t.Type,
			rating, votes, runtime, t.GenreString(), t.IMDbURL,
		}); err != nil {
			return err
		}
	}
	return cw.Error()
}
