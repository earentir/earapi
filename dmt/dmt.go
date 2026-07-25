// Package dmt converts date/time values into Discord magic timestamp tags.
package dmt

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// FormatStyle is one Discord timestamp display style.
type FormatStyle struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// Styles are the Discord timestamp format codes (same order as the DMT page).
var Styles = []FormatStyle{
	{Code: "f", Name: "Short Date/Time", Description: "Month Day, Year Time", Example: "January 15, 2026 3:45 PM"},
	{Code: "F", Name: "Long Date/Time", Description: "Weekday, Month Day, Year Time", Example: "Thursday, January 15, 2026 3:45 PM"},
	{Code: "d", Name: "Short Date", Description: "DD/MM/YYYY", Example: "15/01/2026"},
	{Code: "D", Name: "Long Date", Description: "Month Day, Year", Example: "January 15, 2026"},
	{Code: "t", Name: "Short Time", Description: "HH:MM", Example: "3:45 PM"},
	{Code: "T", Name: "Long Time", Description: "HH:MM:SS", Example: "3:45:00 PM"},
	{Code: "R", Name: "Relative", Description: "Relative to now", Example: "in 2 hours"},
}

// Timestamp is one Discord tag plus metadata.
type Timestamp struct {
	Style   string `json:"style"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Example string `json:"example"`
}

// Result is the API response for a DMT conversion.
type Result struct {
	Unix       int64       `json:"unix"`
	ISO8601    string      `json:"iso8601"`
	UTC        string      `json:"utc"`
	Local      string      `json:"local,omitempty"`
	Completed  bool        `json:"completed"`
	Tag        string      `json:"tag"`
	Style      string      `json:"style"`
	Timestamps []Timestamp `json:"timestamps"`
}

// Input holds the fields used to resolve a moment in time.
type Input struct {
	// Unix is seconds since epoch. If set (>0 or explicitly provided), it wins.
	Unix          *int64
	DateTime      string // RFC3339 / ISO8601, optionally without zone
	Year          int
	Month         int
	Day           int
	Hour          int
	Minute        int
	Second        int
	Offset        string // e.g. "+03:00", "-0500", "Z"; empty = UTC for component form
	Style         string // f F d D t T R, or index "0".."6"
	Complete      bool   // round up to next 5-minute boundary (DMT "Complete")
	HasComponents bool
}

// Convert builds Discord timestamp tags for the given input.
func Convert(in Input) (*Result, error) {
	t, err := resolveTime(in)
	if err != nil {
		return nil, err
	}

	completed := false
	if in.Complete {
		t = completeToFiveMinutes(t)
		completed = true
	}

	unix := t.Unix()
	style, err := resolveStyle(in.Style)
	if err != nil {
		return nil, err
	}

	timestamps := make([]Timestamp, 0, len(Styles))
	for _, s := range Styles {
		timestamps = append(timestamps, Timestamp{
			Style:   s.Code,
			Name:    s.Name,
			Tag:     fmt.Sprintf("<t:%d:%s>", unix, s.Code),
			Example: s.Example,
		})
	}

	primary := fmt.Sprintf("<t:%d:%s>", unix, style.Code)
	return &Result{
		Unix:       unix,
		ISO8601:    t.Format(time.RFC3339),
		UTC:        t.UTC().Format("2006-01-02 15:04:05 UTC"),
		Local:      t.Format("2006-01-02 15:04:05 MST"),
		Completed:  completed,
		Tag:        primary,
		Style:      style.Code,
		Timestamps: timestamps,
	}, nil
}

func resolveStyle(style string) (FormatStyle, error) {
	style = strings.TrimSpace(style)
	if style == "" {
		return Styles[0], nil
	}

	if i, err := strconv.Atoi(style); err == nil {
		if i < 0 || i >= len(Styles) {
			return FormatStyle{}, fmt.Errorf("format index must be 0-%d", len(Styles)-1)
		}
		return Styles[i], nil
	}

	for _, s := range Styles {
		if s.Code == style {
			return s, nil
		}
	}
	return FormatStyle{}, fmt.Errorf("unknown format %q (use f, F, d, D, t, T, R or 0-6)", style)
}

func resolveTime(in Input) (time.Time, error) {
	if in.Unix != nil {
		return time.Unix(*in.Unix, 0).UTC(), nil
	}

	if strings.TrimSpace(in.DateTime) != "" {
		return parseDateTime(in.DateTime, in.Offset)
	}

	if in.HasComponents {
		if in.Year < 1970 || in.Year > 2100 {
			return time.Time{}, fmt.Errorf("year must be between 1970 and 2100")
		}
		if in.Month < 1 || in.Month > 12 {
			return time.Time{}, fmt.Errorf("month must be 1-12")
		}
		if in.Day < 1 || in.Day > 31 {
			return time.Time{}, fmt.Errorf("day must be 1-31")
		}
		if in.Hour < 0 || in.Hour > 23 || in.Minute < 0 || in.Minute > 59 || in.Second < 0 || in.Second > 59 {
			return time.Time{}, fmt.Errorf("invalid time components")
		}

		loc, err := parseOffsetLocation(in.Offset)
		if err != nil {
			return time.Time{}, err
		}
		t := time.Date(in.Year, time.Month(in.Month), in.Day, in.Hour, in.Minute, in.Second, 0, loc)
		return t, nil
	}

	return time.Time{}, fmt.Errorf("provide unix, datetime, or year/month/day (and optional hour/minute/second)")
}

func parseDateTime(raw, offset string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			// layouts without zone are UTC unless offset override is given
			if layout == time.RFC3339 || layout == time.RFC3339Nano {
				return t, nil
			}
			loc, err := parseOffsetLocation(offset)
			if err != nil {
				return time.Time{}, err
			}
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid datetime %q (use RFC3339 or YYYY-MM-DDTHH:MM:SS)", raw)
}

func parseOffsetLocation(offset string) (*time.Location, error) {
	offset = strings.TrimSpace(offset)
	if offset == "" || offset == "Z" || offset == "z" || strings.EqualFold(offset, "UTC") {
		return time.UTC, nil
	}

	// +03:00 / -05:00 / +0300 / -0500
	s := strings.ReplaceAll(offset, ":", "")
	if len(s) == 5 && (s[0] == '+' || s[0] == '-') {
		hours, err1 := strconv.Atoi(s[1:3])
		mins, err2 := strconv.Atoi(s[3:5])
		if err1 != nil || err2 != nil || hours > 14 || mins > 59 {
			return nil, fmt.Errorf("invalid offset %q", offset)
		}
		secondsEast := hours*3600 + mins*60
		if s[0] == '-' {
			secondsEast = -secondsEast
		}
		return time.FixedZone(offset, secondsEast), nil
	}

	// raw minutes east of UTC, e.g. 180 or -300
	if mins, err := strconv.Atoi(offset); err == nil {
		return time.FixedZone(fmt.Sprintf("UTC%+d", mins), mins*60), nil
	}

	return nil, fmt.Errorf("invalid offset %q (use +03:00, -0500, Z, or minutes east of UTC)", offset)
}

// completeToFiveMinutes mirrors the DMT page "Complete" button:
// round up to the next 5-minute boundary from midnight (or leave if already aligned).
func completeToFiveMinutes(t time.Time) time.Time {
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	interval := 5 * time.Minute
	fromMidnight := t.Sub(startOfDay)
	if fromMidnight%interval == 0 {
		return t.Truncate(time.Second)
	}
	rounded := time.Duration(math.Ceil(float64(fromMidnight)/float64(interval))) * interval
	return startOfDay.Add(rounded)
}
