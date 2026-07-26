// Package browser drives a headless Chromium-family browser.
//
// IMDb protects its pages with an AWS WAF challenge: the server answers a plain
// HTTP request with HTTP 202 and a JavaScript proof-of-work that must run before
// the real page is served. No HTTP client can satisfy that, so anything that
// needs the rendered page — notably anonymised p.* profile links, which the
// GraphQL API refuses — goes through a real browser here.
package browser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// realUA avoids Chrome's headless user agent, which advertises "HeadlessChrome"
// and is one of the cheapest bot signals for a WAF to key on.
const realUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// ErrNotFound is returned when no usable browser is installed.
var ErrNotFound = errors.New("no Chromium-based browser found")

// Renderer loads pages in a headless browser.
type Renderer struct {
	ExecPath   string        // browser binary; discovered when empty
	ProfileDir string        // persistent profile, so a solved challenge is reused
	Timeout    time.Duration // per-page budget
	Headless   bool
}

// candidates lists the browsers we know how to drive, best first.
func candidates() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Vivaldi.app/Contents/MacOS/Vivaldi",
		}
	case "windows":
		var out []string
		for _, base := range []string{
			os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData"),
		} {
			if base == "" {
				continue
			}
			out = append(out,
				filepath.Join(base, `Google\Chrome\Application\chrome.exe`),
				filepath.Join(base, `Microsoft\Edge\Application\msedge.exe`),
				filepath.Join(base, `BraveSoftware\Brave-Browser\Application\brave.exe`),
				filepath.Join(base, `Chromium\Application\chrome.exe`),
			)
		}
		return out
	default:
		return []string{
			"/usr/bin/google-chrome", "/usr/bin/google-chrome-stable",
			"/usr/bin/chromium", "/usr/bin/chromium-browser",
			"/usr/bin/microsoft-edge", "/usr/bin/brave-browser",
			"/snap/bin/chromium", "/usr/bin/chrome",
		}
	}
}

// Find locates a browser binary. EARAPI_BROWSER overrides discovery.
func Find() (string, error) {
	if p := strings.TrimSpace(os.Getenv("EARAPI_BROWSER")); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("EARAPI_BROWSER=%s: %w", p, err)
		}
		return p, nil
	}
	for _, p := range candidates() {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}
	// Anything on PATH, for distros that install under a different prefix.
	for _, name := range []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser",
		"microsoft-edge", "brave-browser", "chrome",
	} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", ErrNotFound
}

// New builds a Renderer, discovering a browser and profile directory.
func New(profileDir string) (*Renderer, error) {
	path, err := Find()
	if err != nil {
		return nil, err
	}
	if profileDir == "" {
		cache, cerr := os.UserCacheDir()
		if cerr != nil {
			cache = os.TempDir()
		}
		profileDir = filepath.Join(cache, "earapi", "browser")
	}
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return nil, fmt.Errorf("browser profile dir: %w", err)
	}
	return &Renderer{
		ExecPath:   path,
		ProfileDir: profileDir,
		Timeout:    90 * time.Second,
		Headless:   true,
	}, nil
}

// Name returns a human-readable browser name for status messages.
func (r *Renderer) Name() string {
	base := filepath.Base(r.ExecPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// Ready reports whether the rendered DOM now holds the content we want.
// Returning false keeps polling until the deadline.
//
// Declared as an alias, not a defined type, so *Renderer satisfies interfaces
// written in terms of the plain func signature (imdb.PageFetcher).
type Ready = func(html string) bool

// FetchHTML loads url and returns the rendered DOM once ready reports true.
//
// The WAF challenge reloads the page after solving itself, so we poll the DOM
// rather than trusting a single load event.
func (r *Renderer) FetchHTML(ctx context.Context, url string, ready Ready) (string, error) {
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}

	opts := append([]chromedp.ExecAllocatorOption{},
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.ExecPath(r.ExecPath),
		chromedp.UserDataDir(r.ProfileDir),
		chromedp.UserAgent(realUA),
		chromedp.WindowSize(1440, 1000),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("disable-background-networking", true),
		// navigator.webdriver is the other obvious automation tell.
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
	)
	if r.Headless {
		opts = append(opts, chromedp.Flag("headless", "new"))
	} else {
		opts = append(opts, chromedp.Flag("headless", false))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	tabCtx, cancelTab := chromedp.NewContext(allocCtx)
	defer cancelTab()

	if err := chromedp.Run(tabCtx, chromedp.Navigate(url)); err != nil {
		return "", fmt.Errorf("navigate %s: %w", url, err)
	}

	var last string
	for {
		var html string
		if err := chromedp.Run(tabCtx,
			chromedp.OuterHTML("html", &html, chromedp.ByQuery),
		); err == nil && html != "" {
			last = html
			if ready == nil || ready(html) {
				return html, nil
			}
		}

		select {
		case <-ctx.Done():
			if last != "" {
				// Hand back whatever rendered; the caller can decide whether the
				// partial page is still usable and give a better error if not.
				return last, context.Cause(ctx)
			}
			return "", fmt.Errorf("loading %s: %w", url, ctx.Err())
		case <-time.After(750 * time.Millisecond):
		}
	}
}
