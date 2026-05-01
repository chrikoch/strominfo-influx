package influxwrite

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	influxwriteapi "github.com/influxdata/influxdb-client-go/v2/api/write"

	"github.com/christian/strominfo-influx/internal/config"
	"github.com/christian/strominfo-influx/internal/model"
)

type Writer interface {
	Write(ctx context.Context, points []model.Point) error
	Close()
}

type Client struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	org      string
	bucket   string
}

func NewWriter(cfg config.Config) (*Client, error) {
	options := influxdb2.DefaultOptions().
		SetUseGZip(true).
		SetBatchSize(500).
		SetFlushInterval(uint(cfg.PollInterval / time.Millisecond))

	client := influxdb2.NewClientWithOptions(cfg.InfluxURL, cfg.InfluxToken, options)
	return &Client{
		client:   client,
		writeAPI: client.WriteAPIBlocking(cfg.InfluxOrg, cfg.InfluxBucket),
		org:      cfg.InfluxOrg,
		bucket:   cfg.InfluxBucket,
	}, nil
}

func NewWriterFromClient(client influxdb2.Client, org, bucket string) *Client {
	return &Client{
		client:   client,
		writeAPI: client.WriteAPIBlocking(org, bucket),
		org:      org,
		bucket:   bucket,
	}
}

func (c *Client) Write(ctx context.Context, points []model.Point) error {
	if len(points) == 0 {
		return nil
	}

	influxPoints := make([]*influxwriteapi.Point, 0, len(points))
	for _, point := range points {
		influxPoints = append(influxPoints, influxdb2.NewPoint(point.Measurement, point.Tags, point.Fields, point.Time))
	}

	if err := c.writeAPI.WritePoint(ctx, influxPoints...); err != nil {
		return fmt.Errorf("write points to influx bucket %s: %w", c.bucket, err)
	}

	return nil
}

func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
