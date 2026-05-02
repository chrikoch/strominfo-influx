package transform

import (
	"context"
	"testing"
	"time"

	"github.com/christian/strominfo-influx/internal/energycharts"
)

type scriptedFetcher struct {
	priceResponses     []energycharts.PriceResponse
	frequencyResponses []energycharts.FrequencyResponse
	err                error

	priceCalls     int
	frequencyCalls int

	priceStarts     []time.Time
	priceEnds       []time.Time
	frequencyStarts []time.Time
	frequencyEnds   []time.Time
}

func (s *scriptedFetcher) FetchPrices(_ context.Context, _ string, startDate, endDate time.Time) (energycharts.PriceResponse, error) {
	s.priceStarts = append(s.priceStarts, startDate)
	s.priceEnds = append(s.priceEnds, endDate)

	idx := s.priceCalls
	s.priceCalls++
	if len(s.priceResponses) == 0 {
		return energycharts.PriceResponse{}, s.err
	}
	if idx >= len(s.priceResponses) {
		idx = len(s.priceResponses) - 1
	}
	return s.priceResponses[idx], s.err
}

func (s *scriptedFetcher) FetchFrequency(_ context.Context, startDate, endDate time.Time) (energycharts.FrequencyResponse, error) {
	s.frequencyStarts = append(s.frequencyStarts, startDate)
	s.frequencyEnds = append(s.frequencyEnds, endDate)

	idx := s.frequencyCalls
	s.frequencyCalls++
	if len(s.frequencyResponses) == 0 {
		return energycharts.FrequencyResponse{}, s.err
	}
	if idx >= len(s.frequencyResponses) {
		idx = len(s.frequencyResponses) - 1
	}
	return s.frequencyResponses[idx], s.err
}

func TestPriceCollectorSkipsAlreadyKnownPriceDayBeforeNoon(t *testing.T) {
	t.Parallel()

	location := berlinLocation()
	now := time.Date(2026, time.May, 1, 9, 30, 0, 0, location)

	fetcher := &scriptedFetcher{
		priceResponses: []energycharts.PriceResponse{
			{
				UnixSeconds: []int64{
					berlinTimestamp(location, 2026, time.May, 1, 0, 0, 0),
					berlinTimestamp(location, 2026, time.May, 1, 12, 0, 0),
					berlinTimestamp(location, 2026, time.May, 2, 0, 0, 0),
				},
				Price: []float64{10.0, 11.0, 12.0},
			},
		},
		frequencyResponses: []energycharts.FrequencyResponse{
			{
				UnixSeconds: []int64{
					berlinTimestamp(location, 2026, time.May, 1, 0, 0, 0),
					berlinTimestamp(location, 2026, time.May, 1, 0, 0, 1),
				},
				Data: []float64{49.9, 50.0},
			},
			{
				UnixSeconds: []int64{
					berlinTimestamp(location, 2026, time.May, 1, 0, 0, 0),
					berlinTimestamp(location, 2026, time.May, 1, 0, 0, 1),
					berlinTimestamp(location, 2026, time.May, 1, 0, 0, 2),
				},
				Data: []float64{49.9, 50.0, 50.1},
			},
		},
	}

	collector := NewPriceCollector(fetcher, "DE-LU")
	collector.location = location
	collector.now = func() time.Time { return now }

	points, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("first Collect returned error: %v", err)
	}
	if got := fetcher.priceCalls; got != 1 {
		t.Fatalf("expected one price fetch on first collect, got %d", got)
	}
	if got := fetcher.frequencyCalls; got != 1 {
		t.Fatalf("expected one frequency fetch on first collect, got %d", got)
	}
	if got := len(points); got != 4 {
		t.Fatalf("expected 4 points on first collect, got %d", got)
	}
	if got := fetcher.priceStarts[0].Format(time.DateOnly); got != "2026-05-01" {
		t.Fatalf("unexpected first price fetch start: %s", got)
	}
	if got := fetcher.priceEnds[0].Format(time.DateOnly); got != "2026-05-02" {
		t.Fatalf("unexpected first price fetch end: %s", got)
	}
	if got := fetcher.frequencyStarts[0]; got != time.Date(2026, time.May, 1, 0, 0, 0, 0, location) {
		t.Fatalf("unexpected first frequency fetch start: %s", got)
	}
	if got := fetcher.frequencyEnds[0]; got != now.UTC().Add(time.Second) {
		t.Fatalf("unexpected first frequency fetch end: %s", got)
	}

	points, err = collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("second Collect returned error: %v", err)
	}
	if got := fetcher.priceCalls; got != 1 {
		t.Fatalf("expected price fetch to be skipped before noon, got %d calls", got)
	}
	if got := fetcher.frequencyCalls; got != 2 {
		t.Fatalf("expected frequency fetch on second collect, got %d calls", got)
	}
	if got := len(points); got != 1 {
		t.Fatalf("expected only one new frequency point on second collect, got %d", got)
	}
	if points[0].Measurement != MeasurementFrequency {
		t.Fatalf("unexpected measurement on second collect: %s", points[0].Measurement)
	}
	if got := points[0].Fields["frequency_hz"]; got != 50.1 {
		t.Fatalf("unexpected new frequency value: %#v", got)
	}
	if got := fetcher.frequencyStarts[1]; got != time.Date(2026, time.May, 1, 0, 0, 2, 0, location).UTC() {
		t.Fatalf("unexpected second frequency fetch start: %s", got)
	}
	if got := fetcher.frequencyEnds[1]; got != now.UTC().Add(time.Second) {
		t.Fatalf("unexpected second frequency fetch end: %s", got)
	}
}

func TestPriceCollectorRetriesTomorrowAfterNoon(t *testing.T) {
	t.Parallel()

	location := berlinLocation()
	now := time.Date(2026, time.May, 1, 13, 0, 0, 0, location)

	fetcher := &scriptedFetcher{
		priceResponses: []energycharts.PriceResponse{
			{
				UnixSeconds: []int64{
					berlinTimestamp(location, 2026, time.May, 1, 0, 0, 0),
					berlinTimestamp(location, 2026, time.May, 1, 12, 0, 0),
				},
				Price: []float64{10.0, 11.0},
			},
			{
				UnixSeconds: []int64{
					berlinTimestamp(location, 2026, time.May, 2, 0, 0, 0),
					berlinTimestamp(location, 2026, time.May, 2, 12, 0, 0),
				},
				Price: []float64{12.0, 13.0},
			},
		},
		frequencyResponses: []energycharts.FrequencyResponse{
			{},
			{},
		},
	}

	collector := NewPriceCollector(fetcher, "DE-LU")
	collector.location = location
	collector.now = func() time.Time { return now }

	points, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("first Collect returned error: %v", err)
	}
	if got := fetcher.priceCalls; got != 1 {
		t.Fatalf("expected one price fetch on first collect, got %d", got)
	}
	if got := fetcher.priceStarts[0].Format(time.DateOnly); got != "2026-05-01" {
		t.Fatalf("unexpected first price fetch start: %s", got)
	}
	if got := fetcher.priceEnds[0].Format(time.DateOnly); got != "2026-05-03" {
		t.Fatalf("unexpected first price fetch end after noon: %s", got)
	}
	if got := len(points); got != 2 {
		t.Fatalf("expected only current-day price points on first collect, got %d", got)
	}

	points, err = collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("second Collect returned error: %v", err)
	}
	if got := fetcher.priceCalls; got != 2 {
		t.Fatalf("expected second price fetch for tomorrow, got %d calls", got)
	}
	if got := fetcher.priceStarts[1].Format(time.DateOnly); got != "2026-05-02" {
		t.Fatalf("unexpected second price fetch start: %s", got)
	}
	if got := fetcher.priceEnds[1].Format(time.DateOnly); got != "2026-05-03" {
		t.Fatalf("unexpected second price fetch end: %s", got)
	}
	if got := len(points); got != 2 {
		t.Fatalf("expected tomorrow's price points on second collect, got %d", got)
	}
	if points[0].Time != time.Date(2026, time.May, 1, 22, 0, 0, 0, time.UTC) {
		t.Fatalf("unexpected first returned timestamp on second collect: %s", points[0].Time)
	}

	points, err = collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("third Collect returned error: %v", err)
	}
	if got := fetcher.priceCalls; got != 2 {
		t.Fatalf("expected price fetch to stop once tomorrow is known, got %d calls", got)
	}
	if got := len(points); got != 0 {
		t.Fatalf("expected no new points after tomorrow is known, got %d", got)
	}
	if got := fetcher.frequencyCalls; got != 3 {
		t.Fatalf("expected frequency to keep polling for newer seconds, got %d calls", got)
	}
}

func berlinTimestamp(location *time.Location, year int, month time.Month, day, hour, minute, second int) int64 {
	return time.Date(year, month, day, hour, minute, second, 0, location).UTC().Unix()
}
