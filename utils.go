package main

import (
	"fmt"
	"os"
)

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
