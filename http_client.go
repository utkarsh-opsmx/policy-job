package main

import (
	"net/http"
	"time"
)

var httpClient *http.Client

func init() {
	//TODO: This may have to be changed in the future
	httpClient = NewHTTPClient(timeout)
}

func NewHTTPClient(clientTimeout int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(clientTimeout) * time.Second,
	}

}
