package imdb

// titleFields is the projection shared by every list query. Field names here are
// exact and fragile — note in particular `titleGenres { genres { genre { text } } }`,
// which is NOT the more obvious `genres { genres { genre } }` (that fails schema
// validation).
const titleFields = `
  id
  titleText { text }
  originalTitleText { text }
  releaseYear { year }
  titleType { id text }
  runtime { seconds }
  ratingsSummary { aggregateRating voteCount }
  primaryImage { url }
  plot { plotText { plainText } }
  titleGenres { genres { genre { text } } }
`

// qWatchlist reads a profile's predefined watchlist. Only classic ur… ids are
// accepted; p.* aliases are rejected upstream with BAD_USER_INPUT.
const qWatchlist = `query Watchlist($id: ID!, $first: Int!, $after: ID) {
  predefinedList(classType: WATCH_LIST, userId: $id) {
    items(first: $first, after: $after) {
      total
      pageInfo { hasNextPage endCursor }
      edges { node { item { ... on Title {` + titleFields + `} } } }
    }
  }
}`

// qList reads a custom ls… list. The items connection has the same shape as the
// watchlist one, so both decode into listResponse.
const qList = `query List($id: ID!, $first: Int!, $after: ID) {
  list(id: $id) {
    name { originalText }
    items(first: $first, after: $after) {
      total
      pageInfo { hasNextPage endCursor }
      edges { node { item { ... on Title {` + titleFields + `} } } }
    }
  }
}`

// qTitles hydrates titles by id, used by the CSV import path where we start with
// nothing but tt… ids. Callers build the aliased selection set; see buildTitlesQuery.
const qTitlesHeader = `query Titles {`

// gqlRequest is the JSON body posted to api.graphql.imdb.com.
type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// --- response decoding -------------------------------------------------------

type gqlError struct {
	Message    string `json:"message"`
	Path       []any  `json:"path"`
	Extensions struct {
		Code string `json:"code"`
	} `json:"extensions"`
}

type rawTitle struct {
	ID        string `json:"id"`
	TitleText struct {
		Text string `json:"text"`
	} `json:"titleText"`
	OriginalTitleText struct {
		Text string `json:"text"`
	} `json:"originalTitleText"`
	ReleaseYear struct {
		Year int `json:"year"`
	} `json:"releaseYear"`
	TitleType struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"titleType"`
	Runtime struct {
		Seconds int `json:"seconds"`
	} `json:"runtime"`
	RatingsSummary struct {
		AggregateRating float64 `json:"aggregateRating"`
		VoteCount       int     `json:"voteCount"`
	} `json:"ratingsSummary"`
	PrimaryImage struct {
		URL string `json:"url"`
	} `json:"primaryImage"`
	Plot struct {
		PlotText struct {
			PlainText string `json:"plainText"`
		} `json:"plotText"`
	} `json:"plot"`
	TitleGenres struct {
		Genres []struct {
			Genre struct {
				Text string `json:"text"`
			} `json:"genre"`
		} `json:"genres"`
	} `json:"titleGenres"`
}

// toTitle flattens the deeply nested GraphQL shape into our flat model.
func (r rawTitle) toTitle() Title {
	t := Title{
		IMDbID:     r.ID,
		Title:      r.TitleText.Text,
		Year:       r.ReleaseYear.Year,
		Type:       r.TitleType.ID,
		RuntimeSec: r.Runtime.Seconds,
		Rating:     r.RatingsSummary.AggregateRating,
		Votes:      r.RatingsSummary.VoteCount,
		Plot:       r.Plot.PlotText.PlainText,
		PosterURL:  r.PrimaryImage.URL,
		IMDbURL:    TitleURL(r.ID),
	}
	// Only keep the original title when it actually differs (e.g. Gentle / Szelíd);
	// the matcher uses it as a second chance against a Jellyfin library.
	if o := r.OriginalTitleText.Text; o != "" && o != t.Title {
		t.OrigTitle = o
	}
	for _, g := range r.TitleGenres.Genres {
		if g.Genre.Text != "" {
			t.Genres = append(t.Genres, g.Genre.Text)
		}
	}
	return t
}

type itemsConn struct {
	Total    int `json:"total"`
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
	Edges []struct {
		Node struct {
			Item rawTitle `json:"item"`
		} `json:"node"`
	} `json:"edges"`
}

type listResponse struct {
	Data struct {
		PredefinedList *struct {
			Items itemsConn `json:"items"`
		} `json:"predefinedList"`
		List *struct {
			Name struct {
				OriginalText string `json:"originalText"`
			} `json:"name"`
			Items itemsConn `json:"items"`
		} `json:"list"`
	} `json:"data"`
	Errors []gqlError `json:"errors"`
}
