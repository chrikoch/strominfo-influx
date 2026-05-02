package transform

import (
	"context"
	"testing"
	"time"

	"github.com/christian/strominfo-influx/internal/energycharts"
)

type stubFetcher struct {
	priceResponse     energycharts.PriceResponse
	frequencyResponse energycharts.FrequencyResponse
	err               error
	priceStart        time.Time
	priceEnd          time.Time
	frequencyStart    time.Time
	frequencyEnd      time.Time
}

func (s *stubFetcher) FetchPrices(_ context.Context, _ string, startDate, endDate time.Time) (energycharts.PriceResponse, error) {
	s.priceStart = startDate
	s.priceEnd = endDate
	return s.priceResponse, s.err
}

func (s *stubFetcher) FetchFrequency(_ context.Context, startDate, endDate time.Time) (energycharts.FrequencyResponse, error) {
	s.frequencyStart = startDate
	s.frequencyEnd = endDate
	return s.frequencyResponse, s.err
}

func TestPriceCollectorCollect(t *testing.T) {
	t.Parallel()

	location := berlinLocation()
	now := time.Date(2026, time.May, 1, 9, 30, 0, 0, location)

	fetcher := &stubFetcher{
		priceResponse: energycharts.PriceResponse{
			UnixSeconds: []int64{
				time.Date(2026, time.April, 30, 21, 59, 0, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 2, 12, 0, 0, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC).Unix(),
			},
			Price: []float64{1, 99.5, 101.25, 2},
		},
		frequencyResponse: energycharts.FrequencyResponse{
			UnixSeconds: []int64{
				time.Date(2026, time.April, 30, 23, 59, 59, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC).Unix(),
				time.Date(2026, time.May, 2, 0, 0, 0, 0, time.UTC).Unix(),
			},
			Data: []float64{49.9, 50.01, 50.02, 49.98},
		},
	}

	collector := NewPriceCollector(fetcher, "DE-LU")
	collector.location = location
	collector.now = func() time.Time { return now }

	points, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(points) != 5 {
		t.Fatalf("expected 5 points, got %d", len(points))
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
	if points[2].Measurement != MeasurementFrequency {
		t.Fatalf("unexpected third measurement: %s", points[2].Measurement)
	}
	if points[2].Fields["frequency_hz"] != 49.9 {
		t.Fatalf("unexpected third frequency field: %#v", points[2].Fields["frequency_hz"])
	}
	if got := fetcher.priceStart.Format(time.DateOnly); got != "2026-05-01" {
		t.Fatalf("unexpected fetch start: %s", got)
	}
	if got := fetcher.priceEnd.Format(time.DateOnly); got != "2026-05-02" {
		t.Fatalf("unexpected fetch end: %s", got)
	}
	if got := fetcher.frequencyStart.Format(time.DateOnly); got != "2026-05-01" {
		t.Fatalf("unexpected frequency fetch start: %s", got)
	}
	if got := fetcher.frequencyEnd.Format(time.DateOnly); got != "2026-05-02" {
		t.Fatalf("unexpected frequency fetch end: %s", got)
	}
}
