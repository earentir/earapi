package netflixapi

import (
	"fmt"
	"net/http"

	"github.com/earentir/netflixtudumscrapper"
	"github.com/gin-gonic/gin"
)

type EarapiSettings struct {
	API struct {
		IP   string `json:"ip"`
		Port string `json:"port"`
	} `json:"api"`
	Apikeys struct {
		Steamapikey  string `json:"steamapikey"`
		Tmdbapitoken string `json:"tmdbapitoken"`
	} `json:"apikeys"`
}

type earapiSettings = EarapiSettings

func TopHandler(cfg *earapiSettings) gin.HandlerFunc {
	_ = cfg
	return func(c *gin.Context) {
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

		movies, err := netflixtudumscrapper.ScrapeNetflix(url)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, movies)
	}
}
