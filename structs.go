package main

type steamTopResponse struct {
	UserID string   `json:"userId"`
	Top    int      `json:"top"`
	Games  []string `json:"games"`
}
