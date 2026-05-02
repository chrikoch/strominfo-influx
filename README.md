# strominfo-influx

`strominfo-influx` holt Strommarktdaten von Energy Charts und schreibt sie nach InfluxDB 2.x.

V1 implementiert einen Go-Daemon fuer Day-Ahead-Preise aus Deutschland ueber `bzn=DE-LU`.
Pro Lauf verarbeitet das Tool die Preise fuer heute und morgen im Tagesbezug `Europe/Berlin`.
Bereits bekannte Preis-Tage werden im laufenden Prozess nicht erneut abgefragt; den
Folgetag versucht der Collector ab ca. 12:00 Uhr Berliner Zeit erneut, bis die Daten
vorliegen. Frequenzdaten werden sekundenweise ab dem letzten bekannten Timestamp
weitergeschrieben.

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

- Measurement: `energy_charts_frequency`
- Tags: `source=energy-charts`
- Field: `frequency_hz`
- Timestamp: Wert aus `unix_seconds`

## Beispielabfragen fuer Grafana

Grafana kann InfluxDB 2.x direkt per Flux abfragen. Die folgenden Beispiele passen zu den
Measurements und Feldern dieses Projekts.

Einfacher Preis-Query, z. B. fuer eine Tabelle oder einen Raw-Plot:

```flux
from(bucket: "strom")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "energy_charts_price")
  |> filter(fn: (r) => r._field == "price_eur_mwh")
  |> filter(fn: (r) => r.source == "energy-charts")
  |> filter(fn: (r) => r.bzn == "DE-LU")
  |> keep(columns: ["_time", "_value", "bzn"])
```

Aggregierter Preis-Query, z. B. fuer ein Zeitreihen-Panel mit einer Stunde Aufloesung:

```flux
from(bucket: "strom")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "energy_charts_price")
  |> filter(fn: (r) => r._field == "price_eur_mwh")
  |> filter(fn: (r) => r.source == "energy-charts")
  |> filter(fn: (r) => r.bzn == "DE-LU")
  |> aggregateWindow(every: 1h, fn: mean, createEmpty: false)
```

Frequenz-Query:

```flux
from(bucket: "strom")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "energy_charts_frequency")
  |> filter(fn: (r) => r._field == "frequency_hz")
  |> filter(fn: (r) => r.source == "energy-charts")
  |> aggregateWindow(every: 5m, fn: mean, createEmpty: false)
```

Wenn du ein anderes Bucket oder eine andere BZN nutzt, ersetze `strom` bzw. `DE-LU`
entsprechend.

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

## Container starten

Wenn du `strominfo-influx` nur nutzen und nicht selbst bauen willst, kannst du das fertige Container-Image aus der GitHub Container Registry starten.

Image:

```text
ghcr.io/chrikoch/strominfo-influx
```

Verfuegbare Tags werden in GitHub Actions gebaut und nach GHCR gepusht:

- `main` fuer den aktuellen Stand des `main`-Branches
- `v...` fuer Release-Tags wie `v1.0.0`
- `sha-...` fuer commitgenaue Builds

Fuer normale Nutzung ist ein Release-Tag wie `v1.0.0` die beste Wahl. Wenn du immer den aktuellen Stand aus `main` willst, kannst du `:main` verwenden. Ein Push eines `v...`-Tags erzeugt zusaetzlich ein GitHub Release mit Linux-Binaries fuer `amd64` und `arm64`.

Beispiel:

```bash
docker run --rm \
  -e INFLUX_URL=http://host.docker.internal:8086 \
  -e INFLUX_TOKEN=token \
  -e INFLUX_ORG=home \
  -e INFLUX_BUCKET=strom \
  ghcr.io/chrikoch/strominfo-influx:main
```

Hinweis:

- `host.docker.internal` funktioniert typischerweise mit Docker Desktop auf macOS und Windows.
- Unter Linux musst du fuer `INFLUX_URL` meist die echte Adresse deines InfluxDB-Hosts angeben, zum Beispiel `http://192.168.1.10:8086` oder einen Containernamen im selben Docker-Netzwerk.

## Docker lokal bauen

Wenn du das Image selbst bauen willst:

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

## CI/CD

GitHub Actions:

- fuehrt `go test ./...` aus
- startet dafuer InfluxDB 2.7 als Service-Container
- baut ein Multi-Arch-Image fuer `linux/amd64` und `linux/arm64`
- pusht Images bei Branch-/Tag-Builds nach GHCR
- erzeugt bei `v...`-Tags ein GitHub Release mit automatisch generierten Notes
- haengt dem Release Linux-Binaries fuer `amd64` und `arm64` an
