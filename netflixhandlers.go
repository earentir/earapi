package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/earentir/netflixtudumscrapper"
	"github.com/gin-gonic/gin"
)

func netflixTopHandler(c *gin.Context) {
	var url string
	mediaType := c.DefaultQuery("type", "films")
	country := c.Query("country")
	week := c.Query("week")

	switch mediaType {
	case "films", "movies":
		if country == "" {
			url = "https://www.netflix.com/tudum/top10"
		} else {
			url = fmt.Sprintf("https://www.netflix.com/tudum/top10/%s", country)
		}

	case "series", "tv":
		if country == "" {
			url = "https://www.netflix.com/tudum/top10/tv"
		} else {
			url = fmt.Sprintf("https://www.netflix.com/tudum/top10/%s/tv", country)
		}

	case "pop", "popular":
		url = "https://www.netflix.com/tudum/top10/most-popular/tv"
	}

	if week != "" {
		url = fmt.Sprintf("%s?week=%s", url, week)
	}

	fmt.Println(url)

	movies, err := netflixtudumscrapper.ScrapeNetflix(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Best-effort: fill missing detailUrl by guessing Tudum slug from title.
	// Only attempts a lightweight HEAD request; if it fails, leaves value empty.
	client := &http.Client{Timeout: 3 * time.Second}
	for i := range movies {
		// Access exported fields if available; ignore if not present in struct.
		// We rely on the netflixtudumscrapper types exposing Title and DetailUrl.
		title := movies[i].Title
		if title == "" {
			continue
		}
		if movies[i].DetailURL != "" {
			continue
		}
		slug := slugifyForTudum(title)
		if slug == "" {
			continue
		}
		candidate := fmt.Sprintf("https://www.netflix.com/tudum/%s", slug)
		req, _ := http.NewRequest(http.MethodHead, candidate, nil)
		resp, err := client.Do(req)
		if err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 400 {
			movies[i].DetailURL = candidate
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}

	c.JSON(http.StatusOK, movies)
}

// slugifyForTudum converts a title into a Tudum-like slug.
// Example: "The Thursday Murder Club" -> "the-thursday-murder-club"
func slugifyForTudum(title string) string {
	// Lowercase, remove anything that's not a letter/number/space or hyphen
	n := strings.ToLower(strings.TrimSpace(title))
	// Replace common punctuation with space
	n = strings.ReplaceAll(n, "&", " and ")
	n = strings.ReplaceAll(n, "/", " ")
	n = strings.ReplaceAll(n, "\\", " ")
	n = strings.ReplaceAll(n, ":", " ")
	n = strings.ReplaceAll(n, ";", " ")
	n = strings.ReplaceAll(n, ",", " ")
	n = strings.ReplaceAll(n, ".", " ")
	n = strings.ReplaceAll(n, "'", "")
	n = strings.ReplaceAll(n, "\"", "")

    // Collapse any non-alphanumeric (except spaces and hyphens)
    reNonAlnum := regexp.MustCompile(`[^a-z0-9-\s]`)
    n = reNonAlnum.ReplaceAllString(n, " ")

    // Collapse whitespace to single hyphens
    reSpaces := regexp.MustCompile(`[\s-]+`)
    slug := reSpaces.ReplaceAllString(n, "-")

	// Trim residual hyphens
	slug = strings.Trim(slug, "-")
	return slug
}
