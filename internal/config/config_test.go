package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaultsAndFlags(t *testing.T) {
	t.Parallel()

	cfg, err := Load([]string{"-bzn", "FR", "-poll-interval", "30m", "-influx-url", "http://localhost:8086", "-influx-token", "token", "-influx-org", "org", "-influx-bucket", "bucket"}, func(string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.BiddingZone != "FR" {
		t.Fatalf("expected bidding zone FR, got %q", cfg.BiddingZone)
	}
	if cfg.PollInterval != 30*time.Minute {
		t.Fatalf("expected poll interval 30m, got %s", cfg.PollInterval)
	}
	if cfg.HTTPTimeout != defaultHTTPTimeout {
		t.Fatalf("expected default http timeout %s, got %s", defaultHTTPTimeout, cfg.HTTPTimeout)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"ENERGY_CHARTS_BZN": "NL",
		"POLL_INTERVAL":     "45m",
		"HTTP_TIMEOUT":      "3s",
		"LOG_LEVEL":         "DEBUG",
		"INFLUX_URL":        "http://influx:8086",
		"INFLUX_TOKEN":      "token",
		"INFLUX_ORG":        "org",
		"INFLUX_BUCKET":     "bucket",
	}

	cfg, err := Load(nil, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.BiddingZone != "NL" {
		t.Fatalf("expected env bidding zone, got %q", cfg.BiddingZone)
	}
	if cfg.PollInterval != 45*time.Minute {
		t.Fatalf("expected env poll interval, got %s", cfg.PollInterval)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Fatalf("expected DEBUG log level, got %q", cfg.LogLevel)
	}
}

func TestValidateMissingInfluxFields(t *testing.T) {
	t.Parallel()

	_, err := Load(nil, func(key string) (string, bool) {
		switch key {
		case "INFLUX_URL":
			return "http://localhost:8086", true
		default:
			return "", false
		}
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	for _, fragment := range []string{"influx token is required", "influx org is required", "influx bucket is required"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected error to contain %q, got %v", fragment, err)
		}
	}
}

func TestErrorMessagesFlattensJoinedErrors(t *testing.T) {
	t.Parallel()

	messages := ErrorMessages(errors.Join(
		errors.New("first"),
		errors.Join(errors.New("second"), errors.New("third")),
	))

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
	if messages[0] != "first" || messages[1] != "second" || messages[2] != "third" {
		t.Fatalf("unexpected messages: %#v", messages)
	}
}
