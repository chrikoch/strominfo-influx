# Repository Guidelines

## Project Structure & Module Organization

This repository contains a small Go service that fetches Energy Charts market data and writes it to InfluxDB 2.x.

- `cmd/strominfo-influx/`: application entrypoint
- `internal/config/`: env/flag loading and validation
- `internal/energycharts/`: HTTP client for the Energy Charts API
- `internal/transform/`: mapping API payloads into internal points
- `internal/influxwrite/`: InfluxDB write adapter
- `internal/service/`: daemon loop and orchestration
- `integration/`: integration tests for real InfluxDB writes
- `.github/workflows/ci.yml`: test and Docker image pipeline

Generated binaries should go to `bin/` and should not be committed.

## Build, Test, and Development Commands

- `go build ./cmd/strominfo-influx`: build the service binary
- `go build -o bin/strominfo-influx ./cmd/strominfo-influx`: build into `bin/`
- `go run ./cmd/strominfo-influx`: run locally using current env vars
- `go test ./...`: run all unit tests; integration tests auto-skip unless env vars are set
- `docker build -t strominfo-influx .`: build the container image

For restricted environments, use a writable Go cache, for example: `GOCACHE=/tmp/go-build go test ./...`.

## Coding Style & Naming Conventions

Use standard Go formatting and keep code `gofmt`-clean. Package names should stay short, lowercase, and purpose-driven (`config`, `service`, `transform`). Exported names use Go’s `CamelCase`; unexported helpers use `camelCase`.

Prefer small packages with explicit responsibilities. Keep config keys and Influx field/tag names stable unless the README and tests are updated in the same change.

## Testing Guidelines

Write table-driven or focused unit tests alongside the package under test in `*_test.go` files. Name tests clearly, for example `TestLoadFromEnvironment` or `TestFetchPricesSuccess`.

Integration coverage lives in `integration/` and expects:

- `INTEGRATION_INFLUX_URL`
- `INTEGRATION_INFLUX_TOKEN`
- `INTEGRATION_INFLUX_ORG`
- `INTEGRATION_INFLUX_BUCKET`

## Commit & Pull Request Guidelines

Current history uses short, imperative commit messages such as `Initial commit`. Keep that style: concise subject line, present tense, one logical change per commit.

Pull requests should include a short summary, note config or schema changes, list test commands run, and reference any related issue. If behavior changes, update `README.md` in the same PR.

## Release Process

Releases are driven by annotated git tags matching `v*`. Use a short release-note style tag annotation that summarizes the release, then push the tag to `origin` to trigger the GitHub Actions release job.

Example:

- `git tag -a v0.0.3 -m "v0.0.3" -m "- Add X" -m "- Fix Y"`
