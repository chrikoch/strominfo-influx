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
	start    time.Time
	end      time.Time
}

func (s *stubFetcher) FetchPrices(_ context.Context, _ string, startDate, endDate time.Time) (energycharts.PriceResponse, error) {
	s.start = startDate
	s.end = endDate
	return s.response, s.err
}

func TestPriceCollectorCollect(t *testing.T) {
	t.Parallel()

	location := berlinLocation()
	now := time.Date(2026, time.May, 1, 9, 30, 0, 0, location)

	fetcher := &stubFetcher{
		response: energycharts.PriceResponse{
			UnixSeconds: []int64{
				time.Date(2026, time.April, 30, 21, 59, 0, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 2, 12, 0, 0, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC).Unix(),
			},
			Price: []float64{1, 99.5, 101.25, 2},
		},
	}

	collector := NewPriceCollector(fetcher, "DE-LU")
	collector.location = location
	collector.now = func() time.Time { return now }

	points, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].Measurement != MeasurementPrice {
		t.Fatalf("unexpected measurement: %s", points[0].Measurement)
	}
	if points[0].Tags["source"] != SourceTagValue {
		t.Fatalf("unexpected source tag: %s", points[0].Tags["source"])
	}
	if points[0].Time != time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("unexpected first timestamp: %s", points[0].Time)
	}
	if points[1].Time != time.Date(2026, time.May, 2, 12, 0, 0, 0, time.UTC) {
		t.Fatalf("unexpected second timestamp: %s", points[1].Time)
	}
	if got := fetcher.start.Format(time.DateOnly); got != "2026-05-01" {
		t.Fatalf("unexpected fetch start: %s", got)
	}
	if got := fetcher.end.Format(time.DateOnly); got != "2026-05-02" {
		t.Fatalf("unexpected fetch end: %s", got)
	}
}
