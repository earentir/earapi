package main

import (
	"net/http"

	"github.com/earentir/tmdbapidata"
	"github.com/gin-gonic/gin"
)

func movieSearchHandler(c *gin.Context) {
	// Accept both `q` and `query` params; only default if both are empty
	search := c.Query("q")
	if search == "" {
		search = c.Query("query")
	}
	if search == "" {
		search = "Blade Runner"
	}
	resultKey, resultTitle, mediaData := tmdbapidata.SearchTMDB(config.Apikeys.Tmdbapitoken, search)
	out := gin.H{"info": mediaData}
	out[resultKey] = resultTitle
	c.JSON(http.StatusOK, out)
}
