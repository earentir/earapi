package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// curl "http://localhost:8080/joke?type=geek"
func jokeHandler(c *gin.Context) {
	jokeType := c.DefaultQuery("type", "geek")
	switch jokeType {
	case "geek":
		jokes, err := loadRandomJoke("jokedata/geekjokes.json")
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "cant load geekjokes.json",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"joke": jokes,
		})

	case "excuse", "bofh":
		excuse, err := loadRandomJoke("jokedata/bofh.json")
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "cant load bofh.json",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"excuse": excuse,
		})
		// Handle bofh excuse
	default:
		// Handle default joke
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error": "unknown type",
		})
	}
}
