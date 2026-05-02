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
	priceWindowStart, priceRequestEnd, priceWindowEnd := currentPriceWindow(now)
	frequencyWindowStart, frequencyRequestEnd := currentDayWindow(now)

	priceResponse, err := c.fetcher.FetchPrices(ctx, c.biddingZone, priceWindowStart, priceRequestEnd)
	if err != nil {
		return nil, err
	}
	frequencyResponse, err := c.fetcher.FetchFrequency(ctx, frequencyWindowStart, frequencyRequestEnd)
	if err != nil {
		return nil, err
	}

	points := make([]model.Point, 0, len(priceResponse.UnixSeconds)+len(frequencyResponse.UnixSeconds))
	for i, ts := range priceResponse.UnixSeconds {
		pointTime := time.Unix(ts, 0).UTC()
		if pointTime.Before(priceWindowStart) || !pointTime.Before(priceWindowEnd) {
			continue
		}

		points = append(points, model.Point{
			Measurement: MeasurementPrice,
			Tags: map[string]string{
				"source": SourceTagValue,
				"bzn":    c.biddingZone,
			},
			Fields: map[string]any{
				"price_eur_mwh": priceResponse.Price[i],
			},
			Time: pointTime,
		})
	}

	for i, ts := range frequencyResponse.UnixSeconds {
		pointTime := time.Unix(ts, 0).UTC()
		if pointTime.Before(frequencyWindowStart) || !pointTime.Before(frequencyRequestEnd) {
			continue
		}

		points = append(points, model.Point{
			Measurement: MeasurementFrequency,
			Tags: map[string]string{
				"source": SourceTagValue,
			},
			Fields: map[string]any{
				"frequency_hz": frequencyResponse.Data[i],
			},
			Time: pointTime,
		})
	}

	return points, validatePoints(points)
}

func currentPriceWindow(now time.Time) (time.Time, time.Time, time.Time) {
	windowStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	requestEnd := windowStart.AddDate(0, 0, 1)
	windowEnd := windowStart.AddDate(0, 0, 2)
	return windowStart, requestEnd, windowEnd
}

func currentDayWindow(now time.Time) (time.Time, time.Time) {
	windowStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	windowEnd := windowStart.AddDate(0, 0, 1)
	return windowStart, windowEnd
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
