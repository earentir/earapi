// package main
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var (
	apiversion = "v0.0.13"
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

	movieGroup := r.Group("/movie")
	{
		// movieGroup.GET("/", movieHandler)
		movieGroup.GET("/search", movieSearchHandler)
		// movieGroup.GET("/actor", movieActorHandler)
	}

	netflixGroup := r.Group("/netflix")
	{
		netflixGroup.GET("/top", netflixTopHandler)
	}

	r.GET("/version", versionHandler)

	r.Run(fmt.Sprintf("%s%s", ":", config.API.Port))
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
	c.JSON(http.StatusOK, gin.H{
		"endpoints": endpoints,
	})
}
