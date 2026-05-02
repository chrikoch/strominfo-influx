package transform

import (
	"context"
	"fmt"
	"time"

	"github.com/christian/strominfo-influx/internal/energycharts"
	"github.com/christian/strominfo-influx/internal/model"
)

const (
	MeasurementPrice     = "energy_charts_price"
	MeasurementFrequency = "energy_charts_frequency"
	SourceTagValue       = "energy-charts"
)

type Fetcher interface {
	FetchPrices(ctx context.Context, biddingZone string, startDate, endDate time.Time) (energycharts.PriceResponse, error)
	FetchFrequency(ctx context.Context, startDate, endDate time.Time) (energycharts.FrequencyResponse, error)
}

type Collector interface {
	Collect(ctx context.Context) ([]model.Point, error)
}

type PriceCollector struct {
	fetcher     Fetcher
	biddingZone string
	location    *time.Location
	now         func() time.Time

	lastPriceTime     time.Time
	lastFrequencyTime time.Time
}

func NewPriceCollector(fetcher Fetcher, biddingZone string) *PriceCollector {
	return &PriceCollector{
		fetcher:     fetcher,
		biddingZone: biddingZone,
		location:    berlinLocation(),
		now:         time.Now,
	}
}

func (c *PriceCollector) Collect(ctx context.Context) ([]model.Point, error) {
	now := c.now().In(c.location)

	points := make([]model.Point, 0)

	if priceWindowStart, priceRequestEnd, ok := c.priceRequestWindow(now); ok {
		priceResponse, err := c.fetcher.FetchPrices(ctx, c.biddingZone, priceWindowStart, priceRequestEnd)
		if err != nil {
			return nil, err
		}

		pricePoints, latest := c.pricePoints(priceResponse, priceWindowStart, priceRequestEnd)
		points = append(points, pricePoints...)
		if latest.After(c.lastPriceTime) {
			c.lastPriceTime = latest
		}
	}

	frequencyWindowStart, frequencyRequestEnd := c.frequencyRequestWindow(now)
	frequencyResponse, err := c.fetcher.FetchFrequency(ctx, frequencyWindowStart, frequencyRequestEnd)
	if err != nil {
		return nil, err
	}

	frequencyPoints, latest := c.frequencyPoints(frequencyResponse, frequencyWindowStart, frequencyRequestEnd)
	points = append(points, frequencyPoints...)
	if latest.After(c.lastFrequencyTime) {
		c.lastFrequencyTime = latest
	}

	return points, validatePoints(points)
}

func (c *PriceCollector) priceRequestWindow(now time.Time) (time.Time, time.Time, bool) {
	// Day-ahead prices are stable after publication, so we only request the next
	// unseen day and, after noon, allow one extra day for the following release.
	windowStart := dayStart(now, c.location)
	tomorrowStart := windowStart.AddDate(0, 0, 1)
	maxFetchDay := windowStart
	if !now.Before(noonInLocation(now, c.location)) {
		maxFetchDay = tomorrowStart
	}
	windowEnd := maxFetchDay.AddDate(0, 0, 1)

	start := windowStart
	if !c.lastPriceTime.IsZero() {
		start = dayStart(c.lastPriceTime.In(c.location), c.location).AddDate(0, 0, 1)
	}

	if start.After(maxFetchDay) {
		return time.Time{}, time.Time{}, false
	}

	return start, windowEnd, true
}

func (c *PriceCollector) frequencyRequestWindow(now time.Time) (time.Time, time.Time) {
	windowStart := dayStart(now, c.location)
	start := windowStart
	if !c.lastFrequencyTime.IsZero() {
		start = c.lastFrequencyTime.UTC().Truncate(time.Second).Add(time.Second)
		if start.Before(windowStart) {
			start = windowStart
		}
	}

	return start, now.UTC().Truncate(time.Second).Add(time.Second)
}

func (c *PriceCollector) pricePoints(response energycharts.PriceResponse, windowStart, windowEnd time.Time) ([]model.Point, time.Time) {
	points := make([]model.Point, 0, len(response.UnixSeconds))
	latest := c.lastPriceTime

	for i, ts := range response.UnixSeconds {
		pointTime := time.Unix(ts, 0).UTC()
		if pointTime.Before(windowStart) || !pointTime.Before(windowEnd) {
			continue
		}
		if !c.lastPriceTime.IsZero() && !pointTime.After(c.lastPriceTime) {
			continue
		}

		points = append(points, model.Point{
			Measurement: MeasurementPrice,
			Tags: map[string]string{
				"source": SourceTagValue,
				"bzn":    c.biddingZone,
			},
			Fields: map[string]any{
				"price_eur_mwh": response.Price[i],
			},
			Time: pointTime,
		})
		if pointTime.After(latest) {
			latest = pointTime
		}
	}

	return points, latest
}

func (c *PriceCollector) frequencyPoints(response energycharts.FrequencyResponse, windowStart, windowEnd time.Time) ([]model.Point, time.Time) {
	points := make([]model.Point, 0, len(response.UnixSeconds))
	latest := c.lastFrequencyTime

	for i, ts := range response.UnixSeconds {
		pointTime := time.Unix(ts, 0).UTC()
		if pointTime.Before(windowStart) || !pointTime.Before(windowEnd) {
			continue
		}
		if !c.lastFrequencyTime.IsZero() && !pointTime.After(c.lastFrequencyTime) {
			continue
		}

		points = append(points, model.Point{
			Measurement: MeasurementFrequency,
			Tags: map[string]string{
				"source": SourceTagValue,
			},
			Fields: map[string]any{
				"frequency_hz": response.Data[i],
			},
			Time: pointTime,
		})
		if pointTime.After(latest) {
			latest = pointTime
		}
	}

	return points, latest
}

func dayStart(now time.Time, location *time.Location) time.Time {
	now = now.In(location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
}

func noonInLocation(now time.Time, location *time.Location) time.Time {
	now = now.In(location)
	return time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, location)
}

func berlinLocation() *time.Location {
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(fmt.Sprintf("load Europe/Berlin location: %v", err))
	}
	return location
}

func validatePoints(points []model.Point) error {
	for i, point := range points {
		if point.Measurement == "" {
			return fmt.Errorf("point %d has empty measurement", i)
		}
		if len(point.Fields) == 0 {
			return fmt.Errorf("point %d has no fields", i)
		}
		if point.Time.IsZero() {
			return fmt.Errorf("point %d has zero timestamp", i)
		}
	}
	return nil
}
