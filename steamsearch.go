package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const steamGamesCacheFile = "steamdata/steamgames.json"

type steamSearchAppList struct {
	Apps []steamSearchApp `json:"apps"`
}

type steamSearchApp struct {
	AppID int    `json:"appid"`
	Name  string `json:"name"`
}

type steamStoreAppListResponse struct {
	Response struct {
		Apps []struct {
			AppID int    `json:"appid"`
			Name  string `json:"name"`
		} `json:"apps"`
		HaveMoreResults bool   `json:"have_more_results"`
		LastAppID       uint32 `json:"last_appid"`
	} `json:"response"`
}

func searchSteamApp(apiKey, input string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("steam API key is not configured")
	}

	appList, err := loadSteamSearchAppList(apiKey)
	if err != nil {
		return "", err
	}

	return findSteamSearchApp(appList, input)
}

func loadSteamSearchAppList(apiKey string) (*steamSearchAppList, error) {
	if _, err := os.Stat(steamGamesCacheFile); err == nil {
		file, err := os.ReadFile(steamGamesCacheFile)
		if err != nil {
			return nil, err
		}

		appList := &steamSearchAppList{}
		if err := json.Unmarshal(file, appList); err != nil {
			return nil, err
		}
		if len(appList.Apps) > 0 {
			return appList, nil
		}
	}

	appList, err := fetchSteamStoreAppList(apiKey)
	if err != nil {
		return nil, err
	}

	file, err := json.MarshalIndent(appList, "", " ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(steamGamesCacheFile, file, 0644); err != nil {
		return nil, err
	}

	return appList, nil
}

func fetchSteamStoreAppList(apiKey string) (*steamSearchAppList, error) {
	const maxResults = 50000

	var apps []steamSearchApp
	var lastAppID uint32

	for {
		url := fmt.Sprintf(
			"https://api.steampowered.com/IStoreService/GetAppList/v1/?key=%s&max_results=%d&include_games=true&include_dlc=true&include_software=true&include_videos=true&include_hardware=true",
			apiKey,
			maxResults,
		)
		if lastAppID > 0 {
			url += fmt.Sprintf("&last_appid=%d", lastAppID)
		}

		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("steam store app list request failed: %s", strings.TrimSpace(string(body)))
		}

		var page steamStoreAppListResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}

		for _, app := range page.Response.Apps {
			apps = append(apps, steamSearchApp{
				AppID: app.AppID,
				Name:  app.Name,
			})
		}

		if !page.Response.HaveMoreResults {
			break
		}
		if page.Response.LastAppID == 0 || page.Response.LastAppID == lastAppID {
			break
		}
		lastAppID = page.Response.LastAppID
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf("steam store app list was empty")
	}

	return &steamSearchAppList{Apps: apps}, nil
}

func cleanSteamSearchString(s string) string {
	var cleaned strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			cleaned.WriteRune(r)
		}
	}
	return strings.ToLower(cleaned.String())
}

func findSteamSearchApp(appList *steamSearchAppList, input string) (string, error) {
	cleanedInput := cleanSteamSearchString(input)

	if appID, err := strconv.Atoi(input); err == nil {
		for _, app := range appList.Apps {
			if app.AppID == appID {
				return app.Name, nil
			}
		}
		return "", fmt.Errorf("AppID %d not found", appID)
	}

	for _, app := range appList.Apps {
		if cleanSteamSearchString(app.Name) == cleanedInput {
			return strconv.Itoa(app.AppID), nil
		}
	}
	return "", fmt.Errorf("Game %s not found", input)
}
