package energycharts

import (
	"context"
	"io"
	"net/http"
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

	client := NewClientWithBaseURL("http://energycharts.test", &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.URL.Query().Get("bzn"); got != "DE-LU" {
				t.Fatalf("expected bzn DE-LU, got %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"unix_seconds":[1704067200,1704070800],"price":[10.5,12.25],"unit":"EUR / MWh"}`)),
			}, nil
		}),
	})

	resp, err := client.FetchPrices(context.Background(), "DE-LU")
	if err != nil {
		t.Fatalf("FetchPrices returned error: %v", err)
	}

	if len(resp.Price) != 2 || resp.Price[1] != 12.25 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestFetchPricesMismatchedArrays(t *testing.T) {
	t.Parallel()

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

	_, err := client.FetchPrices(context.Background(), "DE-LU")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "mismatched array lengths") {
		t.Fatalf("unexpected error: %v", err)
	}
}
