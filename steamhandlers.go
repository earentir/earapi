package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/earentir/steamapidata"
	"github.com/gin-gonic/gin"
)

func steamUserIDHandler(c *gin.Context) {
	usernameToLookup := c.DefaultQuery("username", "earentir")
	fmt.Println("username provided", usernameToLookup)
	steamID, statusCode, err := steamapidata.GetSteamID(config.Apikeys.Steamapikey, usernameToLookup)
	if err != nil {
		// If there's an error, return it as a JSON response
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if statusCode == 200 {
		// If successful, return the Steam ID in a JSON response
		c.JSON(http.StatusOK, gin.H{
			"steamID": steamID,
		})
	}

}

func steamAppIDToName(c *gin.Context) {
	appIDStr := c.DefaultQuery("appid", "1086940")

	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		fmt.Println("appid needs to be an int")
	}

	gameDetails, err := steamapidata.SteamAppDetails(appID)
	if err != nil {
		fmt.Println(err)
	} else {

		c.JSON(http.StatusOK, gin.H{
			"name": gameDetails.Name,
		})
	}
}

func steamAppNameToID(c *gin.Context) {
	appName := c.DefaultQuery("name", "1086940")

	appID, err := strconv.Atoi(appName)
	if err != nil {
		fmt.Println("appid needs to be an int")
	}

	gameDetails, err := steamapidata.SteamAppDetails(appID)
	if err != nil {
		fmt.Println(err)
	} else {

		c.JSON(http.StatusOK, gin.H{
			"name": gameDetails.Name,
		})
	}
}

func steamAppDataHandler(c *gin.Context) {
	appIDStr := c.DefaultQuery("appid", "1086940")

	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		fmt.Println("appid needs to be an int")
	}

	gameDetails, err := steamapidata.SteamAppDetails(appID)
	if err != nil {
		fmt.Println(err)

		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"msg":     err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"msg":     "",
			"data": gin.H{
				"appid":            gameDetails.SteamAppid,
				"name":             gameDetails.Name,
				"type":             gameDetails.Type,
				"free":             gameDetails.IsFree,
				"dlc":              gameDetails.Dlc,
				"shortdescription": gameDetails.ShortDescription,
				"headerimage:":     gameDetails.HeaderImage,
				"capsuleimagev5":   gameDetails.CapsuleImagev5,
				"releasedate":      gameDetails.ReleaseDate.Date,
				"genres":           gameDetails.Genres,
				"tags":             gameDetails.Categories,
			},
		})
	}
}

func steamUserAppsUsedHandler(c *gin.Context) {
	userID := c.DefaultQuery("userid", "76561198011985757")

	games, err := steamapidata.SteamUserAppsUsed(config.Apikeys.Steamapikey, userID)
	if err != nil {
		fmt.Println(err)
	} else {
		c.JSON(http.StatusOK, gin.H{
			"games": games,
		})
	}
}

func steamTopHandler(c *gin.Context) {
	userID := c.DefaultQuery("userid", "76561198011985757")
	topCount := c.DefaultQuery("count", "10")
	sortOn := c.DefaultQuery("sortby", "playtime")

	topCountInt, err := strconv.Atoi(topCount)
	if err != nil {
		fmt.Println("count needs to be an int")
	}

	games, err := steamapidata.SteamUserAppsUsed(config.Apikeys.Steamapikey, userID)
	if err != nil {
		fmt.Println(err)
	} else {
		response := steamapidata.SortApps(games, sortOn, topCountInt)
		c.JSON(http.StatusOK, response)
	}
}

func searchSteamAppHandler(c *gin.Context) {
	app := c.DefaultQuery("app", "Baldur's Gate 3")

	foundApp, err := steamapidata.SteamSearchApp(app)
	if err != nil {
		fmt.Println(err)
	} else {
		c.JSON(http.StatusOK, gin.H{
			"app": foundApp,
		})
	}

}
