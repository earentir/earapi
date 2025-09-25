package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type NetflixTopItem struct {
	Rank      int    `json:"rank"`
	Title     string `json:"title"`
	Poster    string `json:"poster"`
	PlayURL   string `json:"playUrl"`
	DetailURL string `json:"detailUrl"`
}

var bgImageRegexp = regexp.MustCompile(`background-image:\s*url\(([^)]+)\)`) // extracts URL inside css url(...)

func extractBackgroundURL(style string) string {
	m := bgImageRegexp.FindStringSubmatch(style)
	if len(m) != 2 {
		return ""
	}
	u := strings.TrimSpace(m[1])
	u = strings.Trim(u, `"'`)
	return u
}

func absoluteURL(base, href string) string {
	if href == "" {
		return href
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return "https://www.netflix.com" + href
	}
	return href
}

// ScrapeNetflix fetches a Tudum Top 10 page and extracts items from the new card-based DOM
func ScrapeNetflix(url string) ([]NetflixTopItem, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
    defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var items []NetflixTopItem
	root := doc.Find(`ul[data-uia="top10-cards"]`).First()
	if root.Length() == 0 {
		return nil, errors.New("no recognized card list found")
	}
	root.Find("li").Each(func(i int, s *goquery.Selection) {
		card := s.Find(`div[data-uia="top10-card"][data-sel="top10-card"]`).First()
		styleAttr, _ := card.Attr("style")
		poster := extractBackgroundURL(styleAttr)

		title := strings.TrimSpace(s.Find(`div[data-uia="top10-card-logo"] img`).First().AttrOr("alt", ""))
		play := s.Find(`a[data-uia="play-on-netflix-link"]`).First().AttrOr("href", "")
		detail := s.Find(`a[data-uia="discover-more-link"]`).First().AttrOr("href", "")

		item := NetflixTopItem{
			Rank:      i + 1,
			Title:     title,
			Poster:    poster,
			PlayURL:   absoluteURL(url, play),
			DetailURL: absoluteURL(url, detail),
		}
		// Only add if we at least have a title or poster to avoid empty entries
		if item.Title != "" || item.Poster != "" {
			items = append(items, item)
		}
	})

	if len(items) == 0 {
		return nil, errors.New("no recognized card items found")
	}
	return items, nil
}


