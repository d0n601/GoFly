package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	DefaultBaseURL = "https://aviationweather.gov"
	historyHours   = 720
)

var ErrNotFound = errors.New("no METAR data found for station")

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{BaseURL: DefaultBaseURL, HTTPClient: httpClient}
}

func (c *Client) GetMetars(ctx context.Context, icao string, history bool) ([]Metar, error) {
	q := url.Values{}
	q.Set("ids", strings.ToUpper(icao))
	q.Set("format", "json")
	if history {
		q.Set("hours", strconv.Itoa(historyHours))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/data/metar?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("api error: status %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNoContent || len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, icao)
	}
	var metars []Metar
	if err := json.Unmarshal(body, &metars); err != nil {
		return nil, fmt.Errorf("decode METAR response: %w", err)
	}
	return metars, nil
}
