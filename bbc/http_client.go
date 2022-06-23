package bbc

import (
	"net"
	"net/http"
	"time"
)

func newClient() *http.Client {
	client := &http.Client{
		Timeout: 600 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			MaxIdleConnsPerHost:   20,
			MaxIdleConns:          100,
			IdleConnTimeout:       900 * time.Second,
			TLSHandshakeTimeout:   100 * time.Second,
			ResponseHeaderTimeout: 600 * time.Second,
			ExpectContinueTimeout: 5 * time.Second,
		},
	}
	return client
}
