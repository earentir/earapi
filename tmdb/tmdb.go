package tmdbapi

import (
	"net/http"

	"github.com/earentir/tmdbapidata"
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

func SearchHandler(cfg *earapiSettings) gin.HandlerFunc {
	return func(c *gin.Context) {
		search := c.DefaultQuery("query", "Blade Runner")
		resulttype, resulttitle, mediadata := tmdbapidata.SearchTMDB(cfg.Apikeys.Tmdbapitoken, search)

		c.JSON(http.StatusOK, gin.H{
			resulttype: resulttitle,
			"info":     mediadata,
		})
	}
}
