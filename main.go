// package main
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	apiversion = "v0.0.19"
	configFile = "config/earapi.json"
	config     earapiSettings
)

func main() {

	loadConfig()

	//setup gin to build the API
	r := gin.Default()

	// Handler for the root path
	r.GET("/", func(c *gin.Context) { rootHandler(c, r) })

	steamGroup := r.Group("/steam")
	{
		// steamGroup.GET("/", steamHandler)
		steamGroup.GET("/top", steamTopHandler)
		steamGroup.GET("/getuserid", steamUserIDHandler)
		steamGroup.GET("/appsused", steamUserAppsUsedHandler)
		steamGroup.GET("/appdata", steamAppDataHandler)
		steamGroup.GET("/search", searchSteamAppHandler)
	}

	r.GET("/joke", jokeHandler)

	tmdbGroup := r.Group("/tmdb")
	{
		// movieGroup.GET("/", movieHandler)
		tmdbGroup.GET("/search", movieSearchHandler)
		// movieGroup.GET("/actor", movieActorHandler)
	}

	netflixGroup := r.Group("/netflix")
	{
		netflixGroup.GET("/top", netflixTopHandler)
	}

	r.GET("/version", versionHandler)

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
	signals := make(chan os.Signal)
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
