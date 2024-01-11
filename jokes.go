package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
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
