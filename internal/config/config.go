package config

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	defaultBiddingZone  = "DE-LU"
	defaultPollInterval = 15 * time.Minute
	defaultHTTPTimeout  = 10 * time.Second
	defaultLogLevel     = "INFO"
)

type LookupEnvFunc func(string) (string, bool)

type LogLevel string

func (l LogLevel) Level() slog.Level {
	switch strings.ToUpper(string(l)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type Config struct {
	BiddingZone  string
	PollInterval time.Duration
	HTTPTimeout  time.Duration
	LogLevel     LogLevel
	InfluxURL    string
	InfluxToken  string
	InfluxOrg    string
	InfluxBucket string
}

type joinUnwrapper interface {
	Unwrap() []error
}

func Load(args []string, lookupEnv LookupEnvFunc) (Config, error) {
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	cfg := Config{}
	fs := flag.NewFlagSet("strominfo-influx", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&cfg.BiddingZone, "bzn", envString(lookupEnv, "ENERGY_CHARTS_BZN", defaultBiddingZone), "Energy Charts bidding zone")
	fs.DurationVar(&cfg.PollInterval, "poll-interval", envDuration(lookupEnv, "POLL_INTERVAL", defaultPollInterval), "Polling interval")
	fs.DurationVar(&cfg.HTTPTimeout, "http-timeout", envDuration(lookupEnv, "HTTP_TIMEOUT", defaultHTTPTimeout), "HTTP timeout")

	logLevel := envString(lookupEnv, "LOG_LEVEL", defaultLogLevel)
	fs.StringVar(&logLevel, "log-level", logLevel, "Log level: DEBUG, INFO, WARN, ERROR")

	cfg.InfluxURL = envString(lookupEnv, "INFLUX_URL", "")
	cfg.InfluxToken = envString(lookupEnv, "INFLUX_TOKEN", "")
	cfg.InfluxOrg = envString(lookupEnv, "INFLUX_ORG", "")
	cfg.InfluxBucket = envString(lookupEnv, "INFLUX_BUCKET", "")

	fs.StringVar(&cfg.InfluxURL, "influx-url", cfg.InfluxURL, "InfluxDB URL")
	fs.StringVar(&cfg.InfluxToken, "influx-token", cfg.InfluxToken, "InfluxDB token")
	fs.StringVar(&cfg.InfluxOrg, "influx-org", cfg.InfluxOrg, "InfluxDB organization")
	fs.StringVar(&cfg.InfluxBucket, "influx-bucket", cfg.InfluxBucket, "InfluxDB bucket")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.LogLevel = LogLevel(logLevel)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var errs []error

	if strings.TrimSpace(c.BiddingZone) == "" {
		errs = append(errs, errors.New("bidding zone must not be empty"))
	}
	if c.PollInterval <= 0 {
		errs = append(errs, errors.New("poll interval must be greater than zero"))
	}
	if c.HTTPTimeout <= 0 {
		errs = append(errs, errors.New("http timeout must be greater than zero"))
	}
	if strings.TrimSpace(c.InfluxURL) == "" {
		errs = append(errs, errors.New("influx url is required"))
	}
	if strings.TrimSpace(c.InfluxToken) == "" {
		errs = append(errs, errors.New("influx token is required"))
	}
	if strings.TrimSpace(c.InfluxOrg) == "" {
		errs = append(errs, errors.New("influx org is required"))
	}
	if strings.TrimSpace(c.InfluxBucket) == "" {
		errs = append(errs, errors.New("influx bucket is required"))
	}

	switch strings.ToUpper(string(c.LogLevel)) {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		errs = append(errs, fmt.Errorf("unsupported log level %q", c.LogLevel))
	}

	return errors.Join(errs...)
}

func envString(lookupEnv LookupEnvFunc, key, fallback string) string {
	value, ok := lookupEnv(key)
	if !ok {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(lookupEnv LookupEnvFunc, key string, fallback time.Duration) time.Duration {
	value, ok := lookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func ErrorMessages(err error) []string {
	if err == nil {
		return nil
	}

	var joined joinUnwrapper
	if errors.As(err, &joined) {
		messages := make([]string, 0, len(joined.Unwrap()))
		for _, wrapped := range joined.Unwrap() {
			messages = append(messages, ErrorMessages(wrapped)...)
		}
		return messages
	}

	return []string{err.Error()}
}
