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

type netflixItem struct {
    Rank      int    `json:"rank"`
    Title     string `json:"title"`
    Poster    string `json:"poster"`
    PlayURL   string `json:"playUrl"`
    DetailURL string `json:"detailUrl"`
}

var reBg = regexp.MustCompile(`background-image:\s*url\(([^)]+)\)`) 

func parseBgURL(style string) string {
    m := reBg.FindStringSubmatch(style)
    if len(m) != 2 { return "" }
    u := strings.Trim(m[1], "\"'")
    return u
}

func absNetflixURL(base, href string) string {
    if href == "" { return href }
    if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
        return href
    }
    if strings.HasPrefix(href, "/") {
        return "https://www.netflix.com" + href
    }
    return href
}

func localScrapeNetflix(url string) ([]netflixItem, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil { return nil, err }
    req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")
    req.Header.Set("Accept-Language", "en-US,en;q=0.9")
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return nil, err }
    defer func(){ _ = resp.Body.Close() }()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("unexpected status: %s", resp.Status)
    }
    body, err := io.ReadAll(resp.Body)
    if err != nil { return nil, err }

    doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
    if err != nil { return nil, err }

    list := doc.Find(`ul[data-uia="top10-cards"]`).First()
    if list.Length() == 0 {
        return nil, errors.New("no recognized card list found")
    }

    var out []netflixItem
    list.Find("li").Each(func(i int, li *goquery.Selection) {
        title := strings.TrimSpace(li.Find(`div[data-uia="top10-card-logo"] img`).First().AttrOr("alt", ""))
        play := li.Find(`a[data-uia="play-on-netflix-link"]`).First().AttrOr("href", "")
        detail := li.Find(`a[data-uia="discover-more-link"]`).First().AttrOr("href", "")

        // Poster style is on the focused card div
        poster := ""
        li.Find(`div[data-uia="top10-card"]`).EachWithBreak(func(_ int, d *goquery.Selection) bool {
            style, ok := d.Attr("style")
            if ok {
                p := parseBgURL(style)
                if p != "" { poster = p; return false }
            }
            return true
        })

        // Fallback title from detail slug
        if title == "" && detail != "" {
            slug := detail
            if idx := strings.LastIndex(slug, "/"); idx >= 0 { slug = slug[idx+1:] }
            slug = strings.ReplaceAll(slug, "-", " ")
            title = strings.Title(strings.TrimSpace(slug))
        }

        item := netflixItem{
            Rank:      i + 1,
            Title:     title,
            Poster:    poster,
            PlayURL:   absNetflixURL(url, play),
            DetailURL: absNetflixURL(url, detail),
        }
        if item.Title != "" || item.Poster != "" {
            out = append(out, item)
        }
    })

    if len(out) == 0 {
        return nil, errors.New("no recognized card items found")
    }
    return out, nil
}


