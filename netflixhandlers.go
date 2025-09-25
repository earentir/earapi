package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func netflixTopHandler(c *gin.Context) {
	var url string
	mediaType := c.DefaultQuery("type", "films")
	country := c.Query("country")
	week := c.Query("week")

	switch mediaType {
	case "films", "movies":
		if country == "" {
			url = "https://www.netflix.com/tudum/top10"
		} else {
			url = fmt.Sprintf("https://www.netflix.com/tudum/top10/%s", country)
		}

	case "series", "tv":
		if country == "" {
			url = "https://www.netflix.com/tudum/top10/tv"
		} else {
			url = fmt.Sprintf("https://www.netflix.com/tudum/top10/%s/tv", country)
		}

	case "pop", "popular":
		url = "https://www.netflix.com/tudum/top10/most-popular/tv"
	}

	if week != "" {
		url = fmt.Sprintf("%s?week=%s", url, week)
	}

	fmt.Println(url)

    movies, err := ScrapeNetflix(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, movies)
}
