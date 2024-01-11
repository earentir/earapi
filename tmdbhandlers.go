package main

import (
	"net/http"

	"github.com/earentir/tmdbapidata"
	"github.com/gin-gonic/gin"
)

func movieSearchHandler(c *gin.Context) {
	search := c.DefaultQuery("query", "Blade Runner")
	resulttype, resulttitle, mediadata := tmdbapidata.SearchTMDB(config.Apikeys.Tmdbapitoken, search)

	c.JSON(http.StatusOK, gin.H{
		resulttype: resulttitle,
		"info":     mediadata,
	})
}
