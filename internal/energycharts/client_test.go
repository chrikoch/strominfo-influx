package energycharts

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchPricesSuccess(t *testing.T) {
	t.Parallel()

	startDate := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	endDate := startDate.AddDate(0, 0, 1)

	client := NewClientWithBaseURL("http://energycharts.test", &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.URL.Query().Get("bzn"); got != "DE-LU" {
				t.Fatalf("expected bzn DE-LU, got %q", got)
			}
			if got := r.URL.Query().Get("start"); got != "2026-05-01" {
				t.Fatalf("expected start 2026-05-01, got %q", got)
			}
			if got := r.URL.Query().Get("end"); got != "2026-05-02" {
				t.Fatalf("expected end 2026-05-02, got %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"unix_seconds":[1704067200,1704070800],"price":[10.5,12.25],"unit":"EUR / MWh"}`)),
			}, nil
		}),
	})

	resp, err := client.FetchPrices(context.Background(), "DE-LU", startDate, endDate)
	if err != nil {
		t.Fatalf("FetchPrices returned error: %v", err)
	}

	if len(resp.Price) != 2 || resp.Price[1] != 12.25 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestFetchFrequencySuccess(t *testing.T) {
	t.Parallel()

	startDate := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 1)

	client := NewClientWithBaseURL("http://energycharts.test", &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.URL.Query().Get("region"); got != "DE-Freiburg" {
				t.Fatalf("expected region DE-Freiburg, got %q", got)
			}
			if got := r.URL.Query().Get("start"); got != strconv.FormatInt(startDate.Unix(), 10) {
				t.Fatalf("expected start %d, got %q", startDate.Unix(), got)
			}
			if got := r.URL.Query().Get("end"); got != strconv.FormatInt(endDate.Unix(), 10) {
				t.Fatalf("expected end %d, got %q", endDate.Unix(), got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"unix_seconds":[1704067200,1704067201],"data":[49.98,50.01]}`)),
			}, nil
		}),
	})

	resp, err := client.FetchFrequency(context.Background(), startDate, endDate)
	if err != nil {
		t.Fatalf("FetchFrequency returned error: %v", err)
	}

	if len(resp.Data) != 2 || resp.Data[0] != 49.98 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestFetchPricesMismatchedArrays(t *testing.T) {
	t.Parallel()

	startDate := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 1)

	client := NewClientWithBaseURL("http://energycharts.test", &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"unix_seconds":[1704067200],"price":[10.5,12.25]}`)),
			}, nil
		}),
	})

	_, err := client.FetchPrices(context.Background(), "DE-LU", startDate, endDate)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mismatched array lengths") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchFrequencyMismatchedArrays(t *testing.T) {
	t.Parallel()

	startDate := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 1)

	client := NewClientWithBaseURL("http://energycharts.test", &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"unix_seconds":[1704067200],"data":[49.98,50.01]}`)),
			}, nil
		}),
	})

	_, err := client.FetchFrequency(context.Background(), startDate, endDate)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mismatched array lengths") {
		t.Fatalf("unexpected error: %v", err)
	}
}
