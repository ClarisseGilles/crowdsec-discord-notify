package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	token, err := a.login(ctx)
	if err != nil {
		return err
	}
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
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "crowdsec-discord-notify/1")
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("delete decision %s: %s", value, response.Status)
	}
	var deleted struct {
		NbDeleted string `json:"nbDeleted"`
	}
	if json.NewDecoder(response.Body).Decode(&deleted) != nil || deleted.NbDeleted == "" || deleted.NbDeleted == "0" {
		return fmt.Errorf("delete decision %s: nothing removed", value)
	}
	return nil
}

func (a *App) login(ctx context.Context) (string, error) {
	endpoint, err := url.Parse(a.config.LAPIURL)
	if err != nil {
		return "", err
	}
	endpoint = endpoint.JoinPath("v1", "watchers", "login")
	body, err := json.Marshal(map[string]string{
		"machine_id": a.config.MachineID,
		"password":   a.config.MachinePassword,
	})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "crowdsec-discord-notify/1")
	response, err := a.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login: %s", response.Status)
	}
	var auth struct {
		Token string `json:"token"`
	}
	if json.NewDecoder(response.Body).Decode(&auth) != nil || auth.Token == "" {
		return "", fmt.Errorf("login: empty token")
	}
	return auth.Token, nil
}