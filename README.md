# strominfo-influx

`strominfo-influx` holt Strommarktdaten von Energy Charts und schreibt sie nach InfluxDB 2.x.

V1 implementiert einen Go-Daemon fuer Day-Ahead-Preise aus Deutschland ueber `bzn=DE-LU`.

## Konfiguration

Das Tool nutzt Umgebungsvariablen, optional durch Flags ueberschrieben.

Pflichtfelder:

- `INFLUX_URL`
- `INFLUX_TOKEN`
- `INFLUX_ORG`
- `INFLUX_BUCKET`

Optionale Felder:

- `ENERGY_CHARTS_BZN` Standard: `DE-LU`
- `POLL_INTERVAL` Standard: `15m`
- `HTTP_TIMEOUT` Standard: `10s`
- `LOG_LEVEL` Standard: `INFO`

Beispiel:

```bash
export INFLUX_URL=http://localhost:8086
export INFLUX_TOKEN=token
export INFLUX_ORG=home
export INFLUX_BUCKET=strom

go run ./cmd/strominfo-influx --poll-interval=15m
```

## Influx-Schema

- Measurement: `energy_charts_price`
- Tags: `source=energy-charts`, `bzn=<zone>`
- Field: `price_eur_mwh`
- Timestamp: Wert aus `unix_seconds`

## Tests

Unit- und Integrationstests laufen mit:

```bash
go test ./...
```

Die Integrationstests werden nur ausgefuehrt, wenn diese Variablen gesetzt sind:

- `INTEGRATION_INFLUX_URL`
- `INTEGRATION_INFLUX_TOKEN`
- `INTEGRATION_INFLUX_ORG`
- `INTEGRATION_INFLUX_BUCKET`

## Docker

Build lokal:

```bash
docker build -t strominfo-influx .
```

## Lokaler Build

Binary lokal bauen, ohne Docker:

```bash
go build -o bin/strominfo-influx ./cmd/strominfo-influx
```

Danach direkt starten:

```bash
./bin/strominfo-influx --poll-interval=15m
```

Run:

```bash
docker run --rm \
  -e INFLUX_URL=http://host.docker.internal:8086 \
  -e INFLUX_TOKEN=token \
  -e INFLUX_ORG=home \
  -e INFLUX_BUCKET=strom \
  ghcr.io/<owner>/strominfo-influx:latest
```

## CI/CD

GitHub Actions:

- fuehrt `go test ./...` aus
- startet dafuer InfluxDB 2.7 als Service-Container
- baut ein Multi-Arch-Image fuer `linux/amd64` und `linux/arm64`
- pusht Images bei Branch-/Tag-Builds nach GHCR
