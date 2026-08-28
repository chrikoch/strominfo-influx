package transform

import (
	"context"
	"fmt"
	"log/slog"

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
	logger      *slog.Logger

	lastPriceTime     time.Time
	lastFrequencyTime time.Time
}

func NewPriceCollector(fetcher Fetcher, biddingZone string) *PriceCollector {
	return NewPriceCollectorWithLogger(fetcher, biddingZone, slog.Default())
}

func NewPriceCollectorWithLogger(fetcher Fetcher, biddingZone string, logger *slog.Logger) *PriceCollector {
	if logger == nil {
		logger = slog.Default()
	}

	return &PriceCollector{
		fetcher:     fetcher,
		biddingZone: biddingZone,
		location:    berlinLocation(),
		now:         time.Now,
		logger:      logger,
	}
}

func (c *PriceCollector) Collect(ctx context.Context) ([]model.Point, error) {
	now := c.now().In(c.location)

	points := make([]model.Point, 0)

	// ----- price fetch (error tolerant) -----
	if priceWindowStart, priceRequestEnd, ok := c.priceRequestWindow(now); ok {
		priceResponse, err := c.fetcher.FetchPrices(ctx, c.biddingZone, priceWindowStart, priceRequestEnd)
		if err != nil {
			fmt.Printf("price fetch error: %v\n", err)
		} else {
			pricePoints, latest := c.pricePoints(priceResponse, priceWindowStart, priceRequestEnd)
			points = append(points, pricePoints...)
			if latest.After(c.lastPriceTime) {
				c.lastPriceTime = latest
			}
		}
	}

	// ----- frequency fetch (error tolerant) -----
	frequencyWindowStart, frequencyRequestEnd := c.frequencyRequestWindow(now)
	frequencyFetchStarted := time.Now()
	frequencyResponse, err := c.fetcher.FetchFrequency(ctx, frequencyWindowStart, frequencyRequestEnd)
	if err != nil {
		c.logger.Error("frequency fetch failed",
			"error", err,
			"start", frequencyWindowStart,
			"end", frequencyRequestEnd,
			"duration", time.Since(frequencyFetchStarted),
		)
	} else {
		frequencyPoints, latest := c.frequencyPoints(frequencyResponse, frequencyWindowStart, frequencyRequestEnd)
		points = append(points, frequencyPoints...)
		if latest.After(c.lastFrequencyTime) {
			c.lastFrequencyTime = latest
		}

		logAttrs := []any{
			"start", frequencyWindowStart,
			"end", frequencyRequestEnd,
			"duration", time.Since(frequencyFetchStarted),
			"response_points", len(frequencyResponse.UnixSeconds),
			"accepted_points", len(frequencyPoints),
		}
		if len(frequencyResponse.UnixSeconds) > 0 {
			logAttrs = append(logAttrs,
				"response_first_timestamp", time.Unix(frequencyResponse.UnixSeconds[0], 0).UTC(),
				"response_last_timestamp", time.Unix(frequencyResponse.UnixSeconds[len(frequencyResponse.UnixSeconds)-1], 0).UTC(),
			)
		}
		if len(frequencyPoints) > 0 {
			logAttrs = append(logAttrs,
				"accepted_first_timestamp", frequencyPoints[0].Time,
				"accepted_last_timestamp", frequencyPoints[len(frequencyPoints)-1].Time,
			)
		}
		c.logger.Info("frequency fetch completed", logAttrs...)
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
	start := time.Time{}
	if !c.lastFrequencyTime.IsZero() {
		// start from the second after the last successfully written point
		start = c.lastFrequencyTime.UTC().Add(time.Second)
	} else {
		// first run: start from 24h ago to get initial data
		start = now.UTC().Add(-24 * time.Hour)
	}

	// safety: never look back more than 24 h to avoid huge requests after long downtime
	maxLookback := now.UTC().Add(-24 * time.Hour)
	if start.Before(maxLookback) {
		start = maxLookback
	}

	end := now.UTC()
	return start, end
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
