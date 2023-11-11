package ccs

import (
	"net/http"
	"net/http/cookiejar"
	"time"
)

type Client struct {
	endpoint string
	client   http.Client
}

type ClientOption func(*Client)

// NewClient returns new API client.
func NewClient(endpoint string, options ...ClientOption) *Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	c := Client{
		endpoint: endpoint,
		client: http.Client{
			Timeout: 5 * time.Second,
			Jar:     jar,
		},
	}
	for _, option := range options {
		option(&c)
	}
	return &c
}
