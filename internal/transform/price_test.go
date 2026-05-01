package transform

import (
	"context"
	"testing"
	"time"

	"github.com/christian/strominfo-influx/internal/energycharts"
)

type stubFetcher struct {
	response energycharts.PriceResponse
	err      error
}

func (s stubFetcher) FetchPrices(context.Context, string) (energycharts.PriceResponse, error) {
	return s.response, s.err
}

func TestPriceCollectorCollect(t *testing.T) {
	t.Parallel()

	collector := NewPriceCollector(stubFetcher{
		response: energycharts.PriceResponse{
			UnixSeconds: []int64{1704067200},
			Price:       []float64{99.5},
		},
	}, "DE-LU")

	points, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
	if points[0].Measurement != MeasurementPrice {
		t.Fatalf("unexpected measurement: %s", points[0].Measurement)
	}
	if points[0].Tags["source"] != SourceTagValue {
		t.Fatalf("unexpected source tag: %s", points[0].Tags["source"])
	}
	if points[0].Time != time.Unix(1704067200, 0).UTC() {
		t.Fatalf("unexpected timestamp: %s", points[0].Time)
	}
}
