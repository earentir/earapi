package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	steamGroup := r.Group("/steam")
	{
		steamGroup.GET("/top", steamTopHandler)
		steamGroup.GET("/getuserid", steamUserIDHandler)
		steamGroup.GET("/appid", steamAppIDHandler)
	}

	r.GET("/joke", jokeHandler)

	movieGroup := r.Group("/movie")
	{
		movieGroup.GET("/", movieHandler)
		movieGroup.GET("/search", movieSearchHandler)
		movieGroup.GET("/actor", movieActorHandler)
	}

	r.Run(":8080")
}

func steamTopHandler(c *gin.Context) {
	// Handle /steam/top here
}

func steamUserIDHandler(c *gin.Context) {
	// Handle /steam/getuserid here
}

func steamAppIDHandler(c *gin.Context) {
	// Handle /steam/appid here
}

func jokeHandler(c *gin.Context) {
	jokeType := c.DefaultQuery("type", "default")
	switch jokeType {
	case "geek":
		// Handle geek joke
	case "dad":
		// Handle dad joke
	default:
		// Handle default joke
	}
}

func movieHandler(c *gin.Context) {
	// Handle /movie here
}

func movieSearchHandler(c *gin.Context) {
	// Handle /movie/search here
}

func movieActorHandler(c *gin.Context) {
	// Handle /movie/actor here
}
