package energycharts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://api.energy-charts.info"

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type PriceResponse struct {
	Unit        string    `json:"unit"`
	LicenseInfo string    `json:"license_info"`
	Deprecated  bool      `json:"deprecated"`
	UnixSeconds []int64   `json:"unix_seconds"`
	Price       []float64 `json:"price"`
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func NewClientWithBaseURL(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) FetchPrices(ctx context.Context, biddingZone string, startDate, endDate time.Time) (PriceResponse, error) {
	endpoint, err := url.Parse(c.baseURL + "/price")
	if err != nil {
		return PriceResponse{}, fmt.Errorf("parse base url: %w", err)
	}

	query := endpoint.Query()
	query.Set("bzn", biddingZone)
	query.Set("start", startDate.Format(time.DateOnly))
	query.Set("end", endDate.Format(time.DateOnly))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return PriceResponse{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PriceResponse{}, fmt.Errorf("request energy charts data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return PriceResponse{}, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var payload PriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return PriceResponse{}, fmt.Errorf("decode response: %w", err)
	}

	if len(payload.UnixSeconds) != len(payload.Price) {
		return PriceResponse{}, fmt.Errorf("mismatched array lengths: unix_seconds=%d price=%d", len(payload.UnixSeconds), len(payload.Price))
	}

	return payload, nil
}
