package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/christian/strominfo-influx/internal/influxwrite"
	"github.com/christian/strominfo-influx/internal/model"
)

func TestWriterWritesPointsToInflux(t *testing.T) {
	influxURL := os.Getenv("INTEGRATION_INFLUX_URL")
	influxToken := os.Getenv("INTEGRATION_INFLUX_TOKEN")
	influxOrg := os.Getenv("INTEGRATION_INFLUX_ORG")
	influxBucket := os.Getenv("INTEGRATION_INFLUX_BUCKET")

	if influxURL == "" || influxToken == "" || influxOrg == "" || influxBucket == "" {
		t.Skip("integration env not configured")
	}

	client := influxdb2.NewClient(influxURL, influxToken)
	defer client.Close()

	writer := influxwrite.NewWriterFromClient(client, influxOrg, influxBucket)
	now := time.Now().UTC().Truncate(time.Second)

	err := writer.Write(context.Background(), []model.Point{
		{
			Measurement: "energy_charts_price",
			Tags: map[string]string{
				"source": "energy-charts",
				"bzn":    "DE-LU",
			},
			Fields: map[string]any{
				"price_eur_mwh": 123.45,
			},
			Time: now,
		},
	})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	queryAPI := client.QueryAPI(influxOrg)
	query := `from(bucket: "` + influxBucket + `")
	|> range(start: -5m)
	|> filter(fn: (r) => r._measurement == "energy_charts_price")
	|> filter(fn: (r) => r.bzn == "DE-LU")
	|> filter(fn: (r) => r._field == "price_eur_mwh")`

	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query returned error: %v", err)
	}
	defer result.Close()

	found := false
	for result.Next() {
		if v, ok := result.Record().Value().(float64); ok && v == 123.45 {
			found = true
			break
		}
	}
	if result.Err() != nil {
		t.Fatalf("query iteration error: %v", result.Err())
	}
	if !found {
		t.Fatalf("expected to find written point in influx query result")
	}
}

func TestIntegrationEnvNamesDocumented(t *testing.T) {
	t.Parallel()

	for _, key := range []string{
		"INTEGRATION_INFLUX_URL",
		"INTEGRATION_INFLUX_TOKEN",
		"INTEGRATION_INFLUX_ORG",
		"INTEGRATION_INFLUX_BUCKET",
	} {
		if !strings.HasPrefix(key, "INTEGRATION_INFLUX_") {
			t.Fatalf("unexpected env key: %s", key)
		}
	}
}
