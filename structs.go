package main

import "earapi/netflix"

type earapiSettings = netflixapi.EarapiSettings

type steamTopResponse struct {
	UserID string   `json:"userId"`
	Top    int      `json:"top"`
	Games  []string `json:"games"`
}
