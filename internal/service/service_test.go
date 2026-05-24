package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/christian/strominfo-influx/internal/model"
)

type stubCollector struct {
	points []model.Point
	err    error
	calls  int
}

func (s *stubCollector) Collect(context.Context) ([]model.Point, error) {
	s.calls++
	return s.points, s.err
}

type stubWriter struct {
	calls  int
	points []model.Point
	err    error
}

func (s *stubWriter) Write(_ context.Context, points []model.Point) error {
	s.calls++
	s.points = points
	return s.err
}

func (s *stubWriter) Close() {}

func TestRunReturnsValidationErrors(t *testing.T) {
	t.Parallel()

	svc := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Interval: time.Second,
	})

	if err := svc.Run(context.Background()); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRunExecutesFirstCycle(t *testing.T) {
	t.Parallel()

	collector := &stubCollector{
		points: []model.Point{{Measurement: "m", Fields: map[string]any{"v": 1}, Time: time.Now()}},
	}
	writer := &stubWriter{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := New(Dependencies{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Collector: collector,
		Writer:    writer,
		Interval:  time.Hour,
	})

	if err := svc.Run(ctx); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if collector.calls != 1 {
		t.Fatalf("expected first cycle to run once, got %d", collector.calls)
	}
	if writer.calls != 1 {
		t.Fatalf("expected writer to be called once, got %d", writer.calls)
	}
}

func TestRunContinuesAfterWriteError(t *testing.T) {
	t.Parallel()

	writer := &stubWriter{err: errors.New("boom")}
	collector := &stubCollector{
		points: []model.Point{{Measurement: "m", Fields: map[string]any{"v": 1}, Time: time.Now()}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	svc := New(Dependencies{
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Collector: collector,
		Writer:    writer,
		Interval:  time.Hour, // long interval so only the initial cycle runs
	})

	// The service should not return an error on write failure; it logs and continues.
	if err := svc.Run(ctx); err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	// The initial cycle must have been attempted.
	if writer.calls == 0 {
		t.Fatal("expected writer to be called at least once")
	}
}
