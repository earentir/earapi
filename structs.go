package main

type steamTopResponse struct {
	UserID string   `json:"userId"`
	Top    int      `json:"top"`
	Games  []string `json:"games"`
}

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
}
