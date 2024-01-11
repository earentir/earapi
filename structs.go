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
}
