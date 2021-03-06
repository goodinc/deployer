package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	PayloadMissing = errors.New("Payload missing")
)

type SlackPayload struct {
	UnfurlLinks bool   `json:"unfurl_links,omitempty"`
	Username    string `json:"username,omitempty"`
	IconEmoji   string `json:"icon_emoji,omitempty"`
	IconUrl     string `json:"icon_url,omitempty"`
	Channel     string `json:"channel,omitempty"`
	Text        string `json:"text"`
}

type SlackClient struct {
	webhook    string
	httpClient http.Client
}

// NewSlackClient returns a Client with the provided webhook url (default timeout to 10 seconds)
func NewSlackClient(webhook string) *SlackClient {
	httpClient := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	c := &SlackClient{
		webhook:    webhook,
		httpClient: httpClient,
	}
	return c
}

// Send sends a text message to the default channel unless overridden
// https://api.slack.com/incoming-webhooks
func (c *SlackClient) Send(p SlackPayload) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}

	res, err := c.httpClient.Post(c.webhook, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	s, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if string(s) != "ok" {
		return errors.New("Slack error: " + string(s))
	}

	return nil
}
