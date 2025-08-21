package jokesapi

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type bofhJokes struct {
	Bofh []string `json:"bofh"`
}

type geekJokes struct {
	Geek [][]string `json:"geekjokes"`
}

// loadRandomJoke loads a JSON file, parses it, and returns a random joke
func loadRandomJoke(filePath string) (interface{}, error) {
	// Read the JSON file
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Seed the random number generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Try unmarshalling into bofhJokes
	var bofh bofhJokes
	err = json.Unmarshal(fileContents, &bofh)
	if err == nil && len(bofh.Bofh) > 0 {
		// If successful and jokes are present, return a random BOFH joke
		return bofh.Bofh[r.Intn(len(bofh.Bofh))], nil
	}

	// Try unmarshalling into geekJokes
	var geek geekJokes
	err = json.Unmarshal(fileContents, &geek)
	if err == nil && len(geek.Geek) > 0 {
		// If successful and jokes are present, return a random array of Geek jokes
		return geek.Geek[r.Intn(len(geek.Geek))], nil
	}

	return "", fmt.Errorf("no jokes found or unrecognized joke format in the file")
}

// Handler provides an HTTP endpoint that returns a joke or excuse based on type
func Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		default:
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error": "unknown type",
			})
		}
	}
}
