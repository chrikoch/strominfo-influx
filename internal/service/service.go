package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/christian/strominfo-influx/internal/model"
	"github.com/christian/strominfo-influx/internal/transform"
)

type Writer interface {
	Write(ctx context.Context, points []model.Point) error
	Close()
}

type Dependencies struct {
	Logger    *slog.Logger
	Collector transform.Collector
	Writer    Writer
	Interval  time.Duration
}

type Service struct {
	logger    *slog.Logger
	collector transform.Collector
	writer    Writer
	interval  time.Duration
}

func New(deps Dependencies) *Service {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		logger:    logger,
		collector: deps.Collector,
		writer:    deps.Writer,
		interval:  deps.Interval,
	}
}

func (s *Service) Run(ctx context.Context) error {
	if s.collector == nil {
		return errors.New("collector is required")
	}
	if s.writer == nil {
		return errors.New("writer is required")
	}
	if s.interval <= 0 {
		return errors.New("interval must be greater than zero")
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := s.runOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("service stopped")
			return nil
		case <-ticker.C:
			if err := s.runOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *Service) runOnce(ctx context.Context) error {
	started := time.Now()
	points, err := s.collector.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collect energy charts data: %w", err)
	}

	if err := s.writer.Write(ctx, points); err != nil {
		return fmt.Errorf("write points: %w", err)
	}

	s.logger.Info("ingestion cycle completed", "points", len(points), "duration", time.Since(started).String())
	return nil
}
