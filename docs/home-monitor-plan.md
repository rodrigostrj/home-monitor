# Home Monitor — Project Plan

> A home-readings platform built together by Rodrigo and his son. Starts with an
> ESP32 + DHT22 temperature sensor. Designed from day one to grow into a
> household dashboard covering energy, water, air quality, and anything else
> worth measuring.

---

## 1. Goals

### Technical goals

- Build a small, clean, **expansible** platform: a single data model (a
  *Reading* from a *Source*) that equally fits a sensor sample, a monthly
  energy bill, and a scraped water-usage row.
- Learn **Go** hands-on by building something real — consciously stepping away
  from the .NET reflexes where they would make the code un-idiomatic.
- Keep the stack small enough to run on a laptop today and move to a Raspberry
  Pi or small cloud VM later without rewrites.

### Family goals

- Give a nearly-9-year-old a genuine, end-to-end experience of how software and
  hardware fit together: *air → sensor → wire → chip → WiFi → server →
  screen*.
- Create moments along the way that are visibly "his": naming sensors, picking
  colours, deciding thresholds, showing the daily graph at school.
- Build the habit of small, working increments — every phase ends with
  something he can see.

---

## 2. Architecture overview

Four independent components over HTTP:

```
┌──────────────┐   POST /api/readings   ┌────────────────┐   GET /api/*   ┌──────────────┐
│   ESP32      │ ─────────────────────► │  Go API        │ ◄───────────── │  Angular     │
│  + DHT22     │    (every 30s)         │  + SQLite      │                │  dashboard   │
└──────────────┘                        └────────────────┘                └──────────────┘
                                               ▲
                                               │ (future)
                                        ┌────────────────┐
                                        │  Integration   │
                                        │  providers:    │
                                        │  water, energy │
                                        └────────────────┘
```

The critical design principle: the ESP32 is **not special**. It's just one
*Source* that publishes *Readings*. A future water-company scraper is another
Source that publishes Readings with a different `MetricType`. Same table, same
API, same widget system on the frontend.

---

## 3. Domain model

Two core entities:

### `Source`

| Field      | Type              | Notes                                                    |
|------------|-------------------|----------------------------------------------------------|
| `id`       | string (uuid)     | Stable identifier, used by the device/integration        |
| `name`     | string            | Human label: "Sala", "Quarto do João", "Contador água"   |
| `kind`     | enum              | `sensor`, `api_integration`, `manual`                    |
| `api_key`  | string (hashed)   | For ingest authentication                                |
| `config`   | JSON              | Free-form per-kind config (e.g. polling URL, credentials)|
| `created_at` | timestamp       |                                                          |

### `Reading`

| Field         | Type            | Notes                                                |
|---------------|-----------------|------------------------------------------------------|
| `id`          | int64           | Auto-increment                                       |
| `source_id`   | string (fk)     |                                                      |
| `metric_type` | string          | `temperature`, `humidity`, `water_m3`, `energy_kwh`… |
| `value`       | float64         |                                                      |
| `unit`        | string          | `°C`, `%`, `m3`, `kWh`…                              |
| `recorded_at` | timestamp (UTC) | When the measurement happened                        |
| `received_at` | timestamp (UTC) | When the API received it                             |
| `metadata`    | JSON (nullable) | Anything extra (battery level, signal strength…)     |

That's it. Every feature downstream composes from these two tables.

---

## 4. Technology stack

### Backend — Go

| Concern         | Choice                               | Why                                                 |
|-----------------|--------------------------------------|-----------------------------------------------------|
| HTTP            | stdlib `net/http` (Go 1.22+ ServeMux)| Idiomatic, no framework to learn on top of Go itself|
| Middleware      | `github.com/go-chi/chi/v5`           | Only if stdlib feels too bare — add later, not now  |
| DB driver       | `modernc.org/sqlite`                 | Pure Go, no CGO, easy cross-compile for the Pi      |
| SQL layer       | `sqlc`                               | Generates type-safe Go from SQL. Teaches SQL-first. |
| Migrations      | `goose`                              | Simple, no daemon, plays well with sqlc             |
| Logging         | `log/slog` (stdlib)                  | Structured logging, modern default                  |
| Config          | env vars + small `config` package    | Twelve-factor style; no viper needed at this scale  |
| Testing         | stdlib `testing` + `testify/assert`  | Standard Go combo                                   |
| Background jobs | stdlib `time.Ticker` in goroutines   | No scheduler needed until 3+ integrations           |

> **Coming from .NET:** no DI container — constructor injection is literally
> "pass the dependency as a function argument." No `IOptions<T>` — structs with
> env-loaded fields. No `async/await` — goroutines + channels. No LINQ — write
> the SQL yourself (sqlc handles the binding). Interfaces are **satisfied
> implicitly** (no `: IFoo`), which feels alien until it clicks.

### Frontend — Angular

- Angular 18+ with **standalone components** (no NgModules).
- **Signals** for state (closer to React hooks; simpler than RxJS for this app).
- **ECharts** via `ngx-echarts` for sparklines and time-series graphs.
- Tailwind CSS for styling (fast iteration, your son can pick colours by
  editing class names with you).
- A **widget registry** pattern: the dashboard is data-driven from a config,
  each widget type is a component.

### ESP32 firmware

- **PlatformIO** (VS Code extension), not the Arduino IDE — gives you a real
  `platformio.ini`, proper dependency management, and a project layout that
  lives cleanly in git. Your son seeing the firmware in the same git repo as
  the rest of the project is a subtle but real lesson.
- Libraries: `WiFi.h`, `HTTPClient.h`, `DHT sensor library` by Adafruit.
- `secrets.h` (git-ignored) for WiFi SSID, password, API endpoint, API key.

---

## 5. Repository structure

Monorepo, single git repo. Two meta-files live at the root: `docs/PLAN.md`
(this document — for humans) and `CLAUDE.md` (terse project rules, auto-loaded
by Claude Code in every session).

```
home-monitor/
├── README.md
├── CLAUDE.md                        # Claude Code project memory (auto-loaded)
├── docs/
│   └── PLAN.md                      # this document (human-readable design doc)
├── firmware/                        # ESP32 code (PlatformIO project)
│   ├── platformio.ini
│   ├── src/
│   │   └── main.cpp
│   ├── include/
│   │   └── secrets.h.example        # template, real one is git-ignored
│   └── .gitignore
├── backend/                         # Go API
│   ├── cmd/api/main.go              # entry point
│   ├── internal/
│   │   ├── api/                     # HTTP handlers + routing
│   │   ├── domain/                  # Source, Reading types
│   │   ├── storage/                 # sqlc-generated code + queries
│   │   ├── providers/               # (future) water, energy integrations
│   │   └── config/
│   ├── db/
│   │   ├── migrations/              # goose .sql files
│   │   └── queries/                 # sqlc .sql files
│   ├── sqlc.yaml
│   ├── go.mod
│   └── Makefile
├── frontend/                        # Angular app
│   ├── src/app/
│   │   ├── core/                    # API client, models
│   │   ├── widgets/                 # one folder per widget type
│   │   │   ├── temperature/
│   │   │   └── humidity/
│   │   ├── dashboard/
│   │   └── widget-registry.ts
│   └── package.json
└── docker-compose.yml               # API + (future) Postgres
```

---

## 6. Local development setup

Because the backend runs on the laptop, a few specifics matter:

- The ESP32 must reach the laptop over the local WiFi. `localhost` won't work
  — the ESP32 needs the laptop's **LAN IP** (e.g. `192.168.1.42`). Reserve it
  on the router if possible, so you don't chase IPs every week.
- The laptop firewall must allow inbound on the API port (default `8080`).
  On Linux Mint: `sudo ufw allow from 192.168.1.0/24 to any port 8080`.
- The API binds to `0.0.0.0:8080`, not `127.0.0.1`.
- Use **ngrok** or **Cloudflare Tunnel** later if you want to access the
  dashboard from outside the home, but that's an explicit Phase 7+ decision.

### Developer toolbox

- Go 1.22+
- Node.js 20+ and the Angular CLI
- PlatformIO extension in VS Code
- `sqlc`, `goose`, `air` (hot reload for Go)
- DBeaver or `sqlite3` CLI for poking at the database

---

## 7. Phased roadmap

Each phase ends with something visible and working. Don't skip that rule — it's
the single biggest predictor of whether a home project finishes, *and* it's
how your son stays engaged.

### Phase 0 — Hardware hello world (1 weekend, with son)
- Wire DHT22 to ESP32 on a breadboard. **He does the wiring, you coach.**
- Flash a sketch that prints temperature + humidity to the serial monitor.
- Goal he can articulate: *"the little chip is measuring the air."*

### Phase 1 — ESP32 publishes to a log-only backend (1 weekend)
- ESP32: connect to WiFi, POST JSON every 30s to the laptop's API.
- Backend: minimal Go server with a single `POST /api/readings` that logs the
  JSON body and returns `202 Accepted`. No database yet.
- Son's job: pick the sensor name, watch the logs scroll in a terminal.

### Phase 2 — Backend with persistence (your solo evenings)
- goose migrations for `sources` and `readings`.
- sqlc queries: `insertReading`, `getLatestReadingPerMetric`, `getReadingsInRange`.
- Endpoints:
  - `POST /api/sources/{id}/readings` — ingest, requires `X-Api-Key` header
  - `GET  /api/sources` — list sources
  - `GET  /api/sources/{id}/readings?metric=…&from=…&to=…`
  - `GET  /api/readings/latest?metric=…` — latest per source
- Table-driven tests for handlers. Integration tests with a real SQLite file.
- Swagger via `swaggo/swag` or a hand-written `openapi.yaml` — your choice.

### Phase 3 — Angular dashboard v1 (1–2 weekends, son co-designs)
- Single temperature tile: current value, unit, last-updated timestamp.
- 24h sparkline below it.
- Colour thresholds: blue when cold, orange when hot. **Your son picks the
  numbers** ("above 26 is hot, below 18 is cold").
- Polling every 10s is fine for v1 — skip WebSockets/SSE until Phase 5.

### Phase 4 — Widget system refactor (solo)
- Dashboard driven by a TypeScript config:
  ```ts
  const layout: WidgetConfig[] = [
    { type: 'temperature', sourceId: 'esp32-sala', title: 'Sala' },
    { type: 'humidity',    sourceId: 'esp32-sala', title: 'Humidade sala' },
  ];
  ```
- `WidgetRegistry` maps `type → component`. Adding a widget = writing a
  component + registering it. This is the payoff of the upfront abstraction.

### Phase 5 — Prove the abstraction with humidity
- DHT22 already gives humidity — publish it as a second `metric_type` from
  the same Source.
- Add a humidity widget. Target: **under one hour** from zero to visible on the
  dashboard. If it takes longer, the abstraction needs a rethink — do it now,
  not after integrations pile on.

### Phase 6 — Live updates (optional, fun)
- Server-Sent Events endpoint `GET /api/stream`.
- Dashboard subscribes and updates tiles in real time.
- Son moment: breathe warm air on the sensor, watch the number jump on the
  screen. Pure magic for a 9-year-old.

### Phase 7+ — Integrations (one per weekend, on your schedule)
- **Outdoor weather** (Open-Meteo, free, no key): a puller goroutine that
  writes `temperature` readings with a `source_id = 'open-meteo-porto'`.
  Instant "outside vs inside" comparison on the dashboard.
- **Energy** (E-Redes MyEnergy API if available, otherwise a monthly CSV
  import endpoint as a fallback).
- **Water** (Indaqua / Águas do Porto portal — check what's available).
- **Air quality** (QualAr API, publicly available in Portugal).

Every integration is the same pattern: a provider that writes `Reading`s + a
widget that reads them. No changes to the core.

---

## 8. Designing for your son's involvement

These are not add-ons — they are part of the plan.

- **Naming everything.** Sensors, widgets, even git branches (`feature/sala-temperature`).
- **Colour decisions.** Let him pick the palette. It will look worse than what
  you'd have chosen. Ship it anyway.
- **Threshold decisions.** "Too cold," "too hot," "too dry" — these are his.
- **The "raw numbers" page** where readings scroll in live. Kids love live data.
- **Daily fun-fact tile.** "Hottest today: 24.3°C at 15:12." Trivial to compute,
  huge engagement.
- **One commit a weekend with his name in the message.** He won't read git
  history at 9. He will at 15.

Age-appropriate concepts to teach along the way:

| Concept                    | How to explain it                                           |
|----------------------------|-------------------------------------------------------------|
| Sensor                     | "It's an eye for something the chip can't see — like heat." |
| WiFi                       | "Invisible radio between the chip and the laptop."          |
| API                        | "A door with rules about what you can ask for."             |
| Database                   | "A notebook that never forgets."                            |
| Graph                      | "Numbers drawn as a picture so we can see shapes."          |
| Version control            | "Save points, like in a videogame — you can go back."       |

---

## 9. Security and operations (light, but from day one)

- **API keys per source**, stored hashed in the DB. The ESP32 sends
  `X-Api-Key: …` on every request.
- **HTTPS** once you leave the laptop — Caddy as a reverse proxy in front of
  the Go API, automatic Let's Encrypt. Not needed for Phase 1–5.
- **Secrets out of git**: `firmware/include/secrets.h` and `backend/.env` are
  both `.gitignore`'d. Commit `.example` files instead.
- **UTC everywhere in storage**, convert to Europe/Lisbon on display.
- **Backups**: one line in a cron that copies `home-monitor.db` to a dated
  file. Add it the day you have data you'd be sad to lose.

---

## 10. Go learning milestones (explicit)

To make sure the "learn Go" goal actually happens and doesn't get
short-circuited by your .NET reflexes:

- **Week 1:** write the ingest handler in stdlib `net/http` only. No chi, no
  frameworks. Feel the ServeMux, context propagation, handler composition.
- **Week 2:** read *Effective Go* (short, official, still the best intro).
  Resist the urge to port Clean Architecture wholesale — Go projects are
  flatter on purpose.
- **Week 3:** write a goroutine + channel example for yourself (not in the
  project): a fan-out/fan-in pattern. This is where Go thinking clicks.
- **Week 4:** do the sqlc + goose integration. SQL-first is a mindset shift
  from EF Core.
- **Week 5:** write table-driven tests. Notice how little ceremony they have
  compared to xUnit.
- **Week 6:** build the first integration provider as a goroutine on a
  `time.Ticker`. Appreciate how much less machinery this is than a
  `BackgroundService` + `IHostedService` + DI registration.

---

## 11. API contract (draft)

### `POST /api/sources/{sourceId}/readings`

```http
POST /api/sources/esp32-sala/readings HTTP/1.1
Content-Type: application/json
X-Api-Key: <per-source key>

{
  "recorded_at": "2026-04-17T10:15:32Z",
  "samples": [
    { "metric_type": "temperature", "value": 22.4, "unit": "°C" },
    { "metric_type": "humidity",    "value": 58.1, "unit": "%"  }
  ],
  "metadata": { "rssi": -62, "uptime_s": 18342 }
}
```

Response: `202 Accepted` with body `{ "stored": 2 }`.

Batching multiple metrics in one request (instead of one POST per metric) means
the ESP32 wakes, reads, POSTs, sleeps. Saves WiFi time, saves battery if you
ever move to battery power.

### `GET /api/readings/latest?metric=temperature`

```json
[
  {
    "source_id": "esp32-sala",
    "source_name": "Sala",
    "metric_type": "temperature",
    "value": 22.4,
    "unit": "°C",
    "recorded_at": "2026-04-17T10:15:32Z"
  }
]
```

### `GET /api/sources/{id}/readings?metric=temperature&from=…&to=…`

Returns an array of readings ordered by `recorded_at` ascending. Used by the
sparkline/graph widgets.

---

## 12. Open questions to close before Phase 1

1. **Sensor placement** — where does the first ESP32 live? (affects WiFi
   range, power outlet, and naming).
2. **Sample interval** — 30s is a reasonable default; confirm. Going below
   10s produces a lot of data for no extra insight at this scale.
3. **Naturalization timing** — unrelated to the project, but if you move to
   Belgium mid-project, it's worth having the backend portable (Docker,
   SQLite file) rather than tied to a specific laptop setup.
4. **Public access** — do you want the dashboard reachable from your phone
   outside the house eventually? Answer dictates whether Caddy + Cloudflare
   Tunnel gets added at Phase 3 or Phase 7.

---

## Appendix A — ESP32 + DHT22 wiring

```
DHT22 (AM2302)         ESP32 WROOM-32
─────────────          ──────────────
VCC (pin 1)     ────►  3V3
DATA (pin 2)    ────►  GPIO 4   (with 10kΩ pull-up to 3V3)
NC  (pin 3)            (not connected)
GND (pin 4)     ────►  GND
```

A 10kΩ pull-up resistor between DATA and 3V3 is required. Some DHT22 breakout
boards already include it — check yours.

---

## Appendix B — Suggested first commits

Small, visible steps make for good git history and good teaching moments:

1. `chore: initial repo layout (firmware, backend, frontend placeholders)`
2. `firmware: serial-only DHT22 read loop`
3. `firmware: wifi connect + POST to hardcoded endpoint`
4. `backend: minimal go api, log-only ingest`
5. `backend: sqlite + goose migrations for sources and readings`
6. `backend: sqlc queries + wired-up ingest persistence`
7. `frontend: angular skeleton with single temperature tile`
8. `frontend: sparkline + threshold colouring (colours chosen by João)`
9. `frontend: widget registry refactor`
10. `firmware+frontend: humidity end-to-end`

---

*Last updated: 2026-04-17*
