package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func loadConfig() {
	//Create the api folders if they don't exist
	err := checkAndCreateFolders("steamdata", "jokedata", "config", "moviedata")
	if err != nil {
		fmt.Println(err)
		os.Exit(125)
	}

	//Load Config
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		fmt.Println("Config file not found, creating default config file")
		err = os.WriteFile(configFile, []byte(`{
			"api": {
				"port": "8080"
			},
			"apikeys": {
				"steamapikey": "",
				"tmdbapitoken": ""
			}
		}`), 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(125)
		}
	} else {
		fmt.Println("Config file found, loading config file")
		js, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(125)
		}

		//decode json to struct
		err = json.Unmarshal(js, &config)
		if err != nil {
			fmt.Println(err)
			os.Exit(125)
		}
	}
}

// checkAndCreateFolders accepts a variadic slice of strings, each representing a folder path
func checkAndCreateFolders(folderPaths ...string) error {
	for _, folderPath := range folderPaths {
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			// Folder does not exist, create it
			err := os.MkdirAll(folderPath, os.ModePerm)
			if err != nil {
				return err
			}
			fmt.Println("Folder created successfully:", folderPath)
		} else {
			fmt.Println("Folder already exists:", folderPath)
		}
	}
	return nil
}
