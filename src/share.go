package main

//createShareHandler handles the /create endpoint

type createShareRequest struct {
	Urls []string `json:"urls"`
}

type fetchShareRequest struct {
	Key string `json:"key"`
}
