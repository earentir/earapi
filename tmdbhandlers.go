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
	resulttype, resulttitle, mediadata := tmdbapidata.SearchTMDB(config.Apikeys.Tmdbapitoken, search)

	c.JSON(http.StatusOK, gin.H{
		resulttype: resulttitle,
		"info":     mediadata,
	})
}
