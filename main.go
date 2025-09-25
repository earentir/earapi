// package main
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	ytpackage "earapi/youtube"
)

var (
	apiversion = "v0.0.28"
	configFile = "config/earapi.json"
	config     earapiSettings
)

func main() {

    loadConfig()

    // CLI helper flags for headless OAuth
    authURL := flag.Bool("youtube-auth-url", false, "print YouTube OAuth URL and exit")
    authCode := flag.String("youtube-auth-code", "", "exchange OAuth code for refresh token")
    flag.Parse()

    if *authURL {
        fmt.Println(ytpackage.BuildAuthURL(config.Youtube.ClientID, config.Youtube.ClientSecret))
        return
    }
    if *authCode != "" {
        rt, _, err := ytpackage.ExchangeCode(context.Background(), config.Youtube.ClientID, config.Youtube.ClientSecret, *authCode)
        if err != nil {
            fmt.Println("exchange error:", err)
            return
        }
        if rt != "" {
            config.Youtube.RefreshToken = rt
            saveConfig()
            fmt.Println("Saved refresh token to config.")
        } else {
            fmt.Println("No refresh token received; ensure AccessTypeOffline and ApprovalForce.")
        }
        return
    }

	//setup gin to build the API
	r := gin.Default()

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

	// r.Run(fmt.Sprintf("%s%s", ":", config.API.Port))

	httpserver :=
		&http.Server{
			Addr:    fmt.Sprintf("%s%s", ":", config.API.Port),
			Handler: r,
		}

	go func() {
		if err := httpserver.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println(err)
		}

	}()

    //setup channels for capturing the termination signal from the OS
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
		"version": apiversion,
	})
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
