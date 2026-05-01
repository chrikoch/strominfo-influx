package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/christian/strominfo-influx/internal/config"
	"github.com/christian/strominfo-influx/internal/energycharts"
	"github.com/christian/strominfo-influx/internal/influxwrite"
	"github.com/christian/strominfo-influx/internal/service"
	"github.com/christian/strominfo-influx/internal/transform"
)

func main() {
	cfg, err := config.Load(os.Args[1:], os.LookupEnv)
	if err != nil {
		slog.Error("failed to load config", "error", strings.Join(config.ErrorMessages(err), "; "))
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel.Level(),
	}))

	client := energycharts.NewClient(cfg.HTTPTimeout)
	collector := transform.NewPriceCollector(client, cfg.BiddingZone)
	writer, err := influxwrite.NewWriter(cfg)
	if err != nil {
		logger.Error("failed to create influx writer", "error", err)
		os.Exit(1)
	}
	defer writer.Close()

	svc := service.New(service.Dependencies{
		Logger:    logger,
		Collector: collector,
		Writer:    writer,
		Interval:  cfg.PollInterval,
	})

	if err := svc.Run(context.Background()); err != nil {
		logger.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}
