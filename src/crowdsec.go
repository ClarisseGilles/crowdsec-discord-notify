package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type alert struct {
	Scope         string   `json:"scope"`
	Value         string   `json:"value"`
	Scenario      string   `json:"scenario"`
	Duration      string   `json:"duration"`
	Country       string   `json:"country"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	City          string   `json:"city"`
	Maliciousness float64  `json:"maliciousness"`
	Domain        string   `json:"domain"`
	Meta          []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"meta"`
}

func (a *App) deleteDecision(ctx context.Context, scope, value string) error {
	endpoint, err := url.Parse(a.config.LAPIURL)
	if err != nil {
		return err
	}
	endpoint = endpoint.JoinPath("v1", "decisions")
	query := endpoint.Query()
	if scope == "Range" {
		query.Set("range", value)
	} else {
		query.Set("ip", value)
	}
	query.Set("type", "ban")
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("X-Api-Key", a.config.BouncerKey)
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("delete decision %s: %s", value, response.Status)
	}
	return nil
}
