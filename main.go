// package main
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	wlpackage "earapi/watchlist"
	ytpackage "earapi/youtube"
)

var (
	configFile = "config/earapi.json"
	config     earapiSettings
	appVersion = "v0.0.31"
)

func main() {
	var authURL bool
	var authCode string
	var authDevice bool

	cmd := &cobra.Command{
		Use:     "earapi",
		Short:   "Ear API server",
		Version: appVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			loadConfig()

			if authURL {
				fmt.Println(ytpackage.BuildAuthURL(config.Youtube.ClientID, config.Youtube.ClientSecret))
				return nil
			}
			if authCode != "" {
				rt, _, err := ytpackage.ExchangeCode(context.Background(), config.Youtube.ClientID, config.Youtube.ClientSecret, authCode)
				if err != nil {
					fmt.Println("exchange error:", err)
					return nil
				}
				if rt != "" {
					config.Youtube.RefreshToken = rt
					saveConfig()
					fmt.Println("Saved refresh token to config.")
				} else {
					fmt.Println("No refresh token received; ensure AccessTypeOffline and ApprovalForce.")
				}
				return nil
			}

			if authDevice {
				ctx := context.Background()
				start, err := ytpackage.StartDeviceFlow(ctx, config.Youtube.ClientID)
				if err != nil {
					fmt.Println("device flow start error:", err)
					return nil
				}
				fmt.Printf("Visit: %s\nEnter code: %s\n", start.VerificationURL, start.UserCode)
				rt, _, err := ytpackage.PollDeviceToken(ctx, config.Youtube.ClientID, config.Youtube.ClientSecret, start.DeviceCode, start.Interval)
				if err != nil {
					fmt.Println("device flow poll error:", err)
					return nil
				}
				if rt != "" {
					config.Youtube.RefreshToken = rt
					saveConfig()
					fmt.Println("Saved refresh token to config.")
				}
				return nil
			}

			runAPIServer()
			return nil
		},
	}

	cmd.Flags().BoolVar(&authURL, "youtube-auth-url", false, "print YouTube OAuth URL and exit")
	cmd.Flags().StringVar(&authCode, "youtube-auth-code", "", "exchange OAuth code for refresh token")
	cmd.Flags().BoolVar(&authDevice, "youtube-auth-device", false, "start OAuth device flow for headless auth")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func runAPIServer() {
	// setup gin to build the API
	r := gin.Default()
	r.Use(corsMiddleware())

	// Handler for the root path
	r.GET("/", func(c *gin.Context) { rootHandler(c, r) })

	steamv1Group := r.Group("/steam/v1/")
	{
		// steamGroup.GET("/", steamHandler)
		steamv1Group.GET("/top", steamTopHandler)
		steamv1Group.GET("/getuserid", steamUserIDHandler)
		steamv1Group.GET("/appsused", steamUserAppsUsedHandler)
		steamv1Group.GET("/appdata", steamAppDataHandler)
		steamv1Group.GET("/search", searchSteamAppHandler)
	}

	r.GET("/joke", jokeHandler)

	tmdbGroup := r.Group("/tmdb/v1/")
	{
		// movieGroup.GET("/", movieHandler)
		tmdbGroup.GET("/search", movieSearchHandler)
		// movieGroup.GET("/actor", movieActorHandler)
	}

	netflixGroup := r.Group("/netflix/v1/")
	{
		netflixGroup.GET("/top", netflixTopHandler)
	}

	tilecalcGroup := r.Group("/tilecalc/v1/")
	{
		tilecalcGroup.GET("/arrange", tilecalcArrangeHandler)
		tilecalcGroup.GET("/coverage", tilecalcCoverageHandler)
	}

	dmtGroup := r.Group("/dmt/v1/")
	{
		dmtGroup.GET("/timestamp", dmtTimestampHandler)
		dmtGroup.GET("/formats", dmtFormatsHandler)
	}

	r.GET("/version", versionHandler)

	// youtube routes
	{
		ytcfg := ytpackage.Config{
			ClientID:       config.Youtube.ClientID,
			ClientSecret:   config.Youtube.ClientSecret,
			RefreshToken:   config.Youtube.RefreshToken,
			DefaultChannel: config.Youtube.DefaultChannel,
			CacheMinutes:   config.Youtube.CacheMinutes,
			OnRefresh: func(newToken string) error {
				// Persist new refresh token back into config and file if rotated
				config.Youtube.RefreshToken = newToken
				// best-effort write
				f, err := os.ReadFile(configFile)
				if err == nil && len(f) > 0 {
					// naive replace; for a robust approach, re-marshal config struct
					// but to avoid altering other fields, we re-encode the struct.
					type cfgAlias earapiSettings
					b, err2 := json.MarshalIndent(cfgAlias(config), "\t\t\t\t\t\t\t\t", "\t")
					if err2 == nil {
						_ = os.WriteFile(configFile, b, 0644)
					}
				}
				return nil
			},
		}
		ytsvc, err := ytpackage.New(context.Background(), ytcfg)
		if err != nil {
			fmt.Println("YouTube init error:", err)
		} else {
			ytpackage.RegisterRoutes(r, ytsvc)
		}
	}

	// IMDb watchlist + Jellyfin playlist routes
	{
		cacheMinutes := config.Watchlist.CacheMinutes
		if cacheMinutes == 0 {
			cacheMinutes = 360 // 6h default when unset / old config
		}
		var cacheTTL time.Duration
		if cacheMinutes < 0 {
			cacheTTL = 0 // negative disables disk cache
		} else {
			cacheTTL = time.Duration(cacheMinutes) * time.Minute
		}
		wlsvc := wlpackage.New(wlpackage.Config{
			CacheDir:       "watchlistdata",
			CacheTTL:       cacheTTL,
			BrowserPath:    config.Watchlist.BrowserPath,
			BrowserHeadful: config.Watchlist.BrowserHeadful,
		})
		wlpackage.RegisterRoutes(r, wlsvc)
		if wlsvc.BrowserName != "" {
			fmt.Println("Watchlist alias resolution via:", wlsvc.BrowserName)
		} else {
			fmt.Println("Watchlist: no browser found — p.* IMDb aliases need CSV import")
		}
	}

	httpserver := &http.Server{
		Addr:    fmt.Sprintf("%s%s", ":", config.API.Port),
		Handler: r,
	}

	go func() {
		if err := httpserver.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println(err)
		}
	}()

	// setup channels for capturing the termination signal from the OS
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	<-signals
	fmt.Println("Shutting down the API")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpserver.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}
}

func versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": appVersion,
	})
}

// corsMiddleware allows the tools UI (GitHub Pages + local proxy) to call the API.
func corsMiddleware() gin.HandlerFunc {
	allowed := map[string]bool{
		"https://earentir.github.io": true,
		"http://127.0.0.1:8766":      true,
		"http://localhost:8766":      true,
		"http://127.0.0.1:8080":      true,
		"http://localhost:8080":      true,
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Accept")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func rootHandler(c *gin.Context, r *gin.Engine) {
	routes := r.Routes()
	var endpoints []string
	for _, route := range routes {
		endpoints = append(endpoints, fmt.Sprintf("%s - %s", route.Method, route.Path))
	}
	endpoints = append(endpoints, fmt.Sprintf("%s - %s", "GET", "/doc"))
	c.JSON(http.StatusOK, gin.H{
		"endpoints": endpoints,
	})
}
