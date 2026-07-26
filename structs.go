package main

// removed unused steamTopResponse

type earapiSettings struct {
	API struct {
		Port string `json:"port"`
	} `json:"api"`
	Apikeys struct {
		Steamapikey  string `json:"steamapikey"`
		Tmdbapitoken string `json:"tmdbapitoken"`
	} `json:"apikeys"`
	Youtube struct {
		ClientID       string `json:"client_id"`
		ClientSecret   string `json:"client_secret"`
		RefreshToken   string `json:"refresh_token"`
		DefaultChannel string `json:"default_channel_id"`
		CacheMinutes   int    `json:"cache_minutes"`
	} `json:"youtube"`
	Watchlist struct {
		CacheMinutes   int    `json:"cache_minutes"` // IMDb list disk cache; 0 disables
		BrowserPath    string `json:"browser_path"`  // optional Chrome/Chromium path for p.* aliases
		BrowserHeadful bool   `json:"browser_headful"`
	} `json:"watchlist"`
}
