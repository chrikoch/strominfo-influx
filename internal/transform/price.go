package transform

import (
	"context"
	"fmt"
	"time"

	"github.com/christian/strominfo-influx/internal/energycharts"
	"github.com/christian/strominfo-influx/internal/model"
)

const (
	MeasurementPrice = "energy_charts_price"
	SourceTagValue   = "energy-charts"
)

type Fetcher interface {
	FetchPrices(ctx context.Context, biddingZone string) (energycharts.PriceResponse, error)
}

type Collector interface {
	Collect(ctx context.Context) ([]model.Point, error)
}

type PriceCollector struct {
	fetcher     Fetcher
	biddingZone string
}

func NewPriceCollector(fetcher Fetcher, biddingZone string) *PriceCollector {
	return &PriceCollector{
		fetcher:     fetcher,
		biddingZone: biddingZone,
	}
}

func (c *PriceCollector) Collect(ctx context.Context) ([]model.Point, error) {
	response, err := c.fetcher.FetchPrices(ctx, c.biddingZone)
	if err != nil {
		return nil, err
	}

	points := make([]model.Point, 0, len(response.UnixSeconds))
	for i, ts := range response.UnixSeconds {
		points = append(points, model.Point{
			Measurement: MeasurementPrice,
			Tags: map[string]string{
				"source": SourceTagValue,
				"bzn":    c.biddingZone,
			},
			Fields: map[string]any{
				"price_eur_mwh": response.Price[i],
			},
			Time: time.Unix(ts, 0).UTC(),
		})
	}

	return points, validatePoints(points)
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
