package influxwrite

import (
	"context"
	"testing"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/christian/strominfo-influx/internal/model"
)

func TestWriteEmptyPoints(t *testing.T) {
	t.Parallel()

	client := influxdb2.NewClient("http://127.0.0.1:1", "token")
	defer client.Close()

	writer := NewWriterFromClient(client, "org", "bucket")
	if err := writer.Write(context.Background(), nil); err != nil {
		t.Fatalf("Write returned error for empty points: %v", err)
	}
}

func TestPointConversion(t *testing.T) {
	t.Parallel()

	point := model.Point{
		Measurement: "energy_charts_price",
		Tags: map[string]string{
			"source": "energy-charts",
		},
		Fields: map[string]any{
			"price_eur_mwh": 42.5,
		},
		Time: time.Unix(1704067200, 0).UTC(),
	}

	influxPoint := influxdb2.NewPoint(point.Measurement, point.Tags, point.Fields, point.Time)
	if influxPoint.Name() != "energy_charts_price" {
		t.Fatalf("unexpected measurement: %s", influxPoint.Name())
	}
}
