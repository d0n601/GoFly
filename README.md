# gofly ✈️

A command line tool that fetches aviation weather for an airport and prints a
short human-readable report. It can show the latest
[METAR](https://aviationweather.gov) observation (plus the raw METAR) or, with
`--forecast`, an hourly model forecast from
[Open-Meteo](https://open-meteo.com) for a moment in time.

Use for automations to message you current/forcast conditions for flights:

```console
$ ./gofly KEDU
KEDU Davis/University Arpt, CA, US
Observed: 2026-09-09 13:35 PDT (20:35 UTC)
Wind 360° 7kt | Vis 10+ | 35°C/4°C | Altimeter 30.02 inHg
METAR KEDU 092035Z AUTO 36007KT 330V030 10SM 35/04 A3002 RMK AO1
```

…or what to expect at a planned arrival several days out:

```console
$ ./gofly KEDU "2026-09-16 07:00" --forecast
KEDU Davis/University Arpt, CA, US
Forecast: 2026-09-16 06:00 PDT (13:00 UTC)
Wind 242° 03kt | Vis 15+ SM | Sky CLR | 13.1°C/9.3°C | Altimeter 29.96 inHg
Forecast: 2026-09-16 07:00 PDT (14:00 UTC)
Wind 256° 03kt G3 | Vis 15+ SM | Sky CLR | 14.1°C/8.3°C | Altimeter 29.98 inHg
Forecast: 2026-09-16 08:00 PDT (15:00 UTC)
Wind 274° 02kt G4 | Vis 15+ SM | Sky CLR | 15.8°C/7.5°C | Altimeter 30.00 inHg
```

## ✨ Features

- 🛬 Airport lookup by ICAO identifier (e.g. `KEDU`), case-insensitive (3-4
  letters).
- ⏰ Optional time argument to fetch the observation closest to a moment in
  time (past lookups and "conditions for planned arrival" both work).
- 🔮 `--forecast` to get a model forecast (from Open-Meteo) around a moment in
  time instead of an observation — this works for future times up to about 16
  days out.
- 🌍 Time arguments without an explicit offset are interpreted in the default
  timezone (Los Angeles, `America/Los_Angeles`, by default), with daylight
  saving handled automatically. The default is a single constant you can change
  to any IANA zone — see [Timezone](#timezone).
- 🕐 No arguments for time means "now" (latest available observation).
- 📋 Human-readable summary plus the raw METAR, ready to copy into a message.
- 📦 Single static binary, zero external runtime dependencies, IANA timezone
  database embedded.

## 📦 Install

Requirements: [Go](https://go.dev/dl/) 1.26.5 or newer. The project uses the
standard library only, so there are no third-party dependencies to download.

From a checkout of the repository:

```console
go build -o gofly .
```

The resulting binary is self-contained. To install it onto your `PATH`:

```console
go install .
```

(Or copy the built binary anywhere you like, e.g. `cp gofly ~/.local/bin/`.)

## 🚀 Usage

```
gofly <ICAO> [TIME] [--forecast]
```

### 🧩 Arguments

- `ICAO` — airport identifier, 3-4 letters (case-insensitive), e.g. `KEDU`.
- `TIME` — optional. Omit it for current weather. Accepted formats:
  - `HH:MM` — that time today in the default timezone, e.g. `14:30`
  - `YYYY-MM-DD HH:MM` or `YYYY-MM-DDTHH:MM` — that date/time in the default
    timezone
  - RFC 3339 with an explicit offset, e.g.
    `2026-09-09T20:35:00Z` (UTC) or `2026-09-09T14:35:00-07:00` (PDT) —
    used as-is, ignoring the default timezone

### 🚩 Flags

- `--forecast`, `-f` — print a model forecast (three hourly steps centred on
  the time, defaulting to now) instead of an observation. It can be written
  before or after the positional arguments.

### 💡 Examples

```console
$ ./gofly KEDU
KEDU Davis/University Arpt, CA, US
Observed: 2026-09-09 13:35 PDT (20:35 UTC)
Wind 360° 7kt | Vis 10+ | 35°C/4°C | Altimeter 30.02 inHg
METAR KEDU 092035Z AUTO 36007KT 330V030 10SM 35/04 A3002 RMK AO1

# Conditions at a specific date/time (closest observation is used)
$ ./gofly KEDU "2026-09-08 14:30"
KEDU Davis/University Arpt, CA, US
Observed: 2026-09-08 14:35 PDT (21:35 UTC)
Wind 020° 8kt | Vis 9 | 33°C/5°C | Altimeter 29.97 inHg
METAR KEDU 082135Z AUTO 02008KT 9SM 33/05 A2997 RMK AO1

# Time only: today at 14:30 in the default timezone
$ ./gofly KEDU 14:30

# Explicit UTC time
$ ./gofly KEDU "2026-09-08T21:35:00Z"

# Forecast for the next few hours (centred on now)
$ ./gofly KEDU --forecast

# Forecast around a planned arrival on a specific day
$ ./gofly KEDU "2026-09-16 07:00" --forecast
```

For an observation at a future time, the tool reports the latest available
observation (current conditions). For a past time, the observation closest to
that moment is selected. With `--forecast`, the requested time is used
directly and three hourly steps (one hour before, the hour itself, and one
hour after) are printed, whether the time is in the past or the future.

### 📄 Output

An **observation** report contains:

1. The ICAO ID and station name
2. The observation time in the default timezone and UTC
3. Key conditions: wind (direction/speed), visibility, temperature/dewpoint,
   and altimeter (converted to inHg)
4. The raw METAR text

A **forecast** report shows the ICAO ID and station name once, then one block
per hour. Each block has a `Forecast:` line (local time and UTC) followed by:

- 🌬️ Wind: direction, speed in knots, and gust (only when stronger than the
  sustained wind)
- 👀 Visibility, in statute miles (`15+ SM` above 10 nm-equivalent)
- ⛅ Sky: the derived ceiling — `CLR`, `FG` (fog), or `BKN`/`SCT` with the
  layer base in feet above ground
- 🌧️ A short phrase for active precipitation or other significant weather
- 🌡️ Temperature/dewpoint and the altimeter (in inHg)
- 💧 Precipitation amount and probability (only when rain/snow is expected)

### 🔢 Exit codes

- `0` — ✅ success
- `1` — ❌ error (network/API failure, unknown station, invalid time)
- `2` — 🚫 usage error (wrong number of arguments, unknown flag)

## 🌍 Timezone

gfly uses one default timezone for two things: interpreting times you type
without an explicit offset (`14:30`, `2026-09-09 14:30`) and printing the local
time in reports. It defaults to Los Angeles.

To change it, edit a single string — the `DefaultTimeZone` constant at
[`timeparse/timeparse.go:18`](timeparse/timeparse.go#L18):

```go
const DefaultTimeZone = "America/Los_Angeles"
```

For example, to default to Chicago instead:

```go
const DefaultTimeZone = "America/Chicago"
```

Take an IANA "zone" name. Daylight saving is applied automatically, so
Chicago shows `CDT` in summer and `CST` in winter, Los Angeles shows `PDT`/`PST`,
and Phoenix (most of Arizona) stays on `MST` all year because it observes no
daylight saving. The North American timezones (Pacific, Mountain, Central,
Eastern, and Alaska) all work out of the box, so does anything else in the
IANA database. The IANA timezone database is embedded in the binary, so a
configured zone works even on a machine with no system tz database.

## ⚙️ How it works

The tool queries the public
[aviationweather.gov data API](https://aviationweather.gov/api/data/metar) at
`/api/data/metar`. Current weather is a single query; a requested time fetches
up to the last 720 hours of observations (the API maximum) and the tool
selects the observation whose timestamp is closest to the requested time.
The timezone conversion and selection happen locally, so no extra lookups are
needed.

If the station is unknown or has no data, the API responds with HTTP 204 and
the tool reports `no METAR data found for <ID>`.

For `--forecast`, the tool reuses the station's coordinates and elevation taken
from its METAR record and queries the
[Open-Meteo forecast API](https://open-meteo.com) for one hour before, the
hour of, and one hour after the requested time. Wind is requested in knots.
The "Sky" ceiling is derived locally: the lowest pressure level (925 hPa down
to 100 hPa) whose cloud cover reaches 50% is reported as `BKN`, otherwise the
lowest level at 20% or more is `SCT`, and fog is preferred when the hour has a
fog weather code. Open-Meteo provides about 16 days of future and 3 months of
past hourly data; times outside that range produce a clear error.

## ⚠️ Limitations

- 🔮 Forecasts are model data, not an official TAF. The Sky/ceiling and gust
  values are derived by the tool and are an approximation, so treat them as
  indicative rather than authoritative for the flight.
- 📅 Forecast range: Open-Meteo covers roughly 16 days ahead and 3 months back;
  outside that, the tool reports the API's out-of-range error.
- 📜 Observation history: the aviationweather.gov API retains on the order of a
  week of METARs per station. For times older than that, the oldest available
  observation is returned — the `Observed:` line always shows the actual
  observation time.
- 🌍 The time argument defaults to the default timezone (Los Angeles); an
  offset-free time is interpreted in that zone unless you give an explicit
  offset. See [Timezone](#timezone) to change the default.

## 🛠️ Development

Project layout:

```
main.go            binary entry point
api/               aviationweather.gov (METAR) and Open-Meteo (forecast) clients,
                   plus ceiling/step derivation
timeparse/         time argument parsing and the default-timezone setting
pick/              closest-observation selection
render/            human-readable report formatting (observation + forecast)
app/               argument handling and orchestration
```

Run the test suite and checks:

```console
go test ./...
go vet ./...
gofmt -l .
```

Tests use `net/http/httptest` mock servers throughout — no network access is
needed to run them.

## ✍️ Credits

Written by Qwen3.6-27B 🤖, using [OpenCode](https://opencode.ai) as the coding
agent harness.
