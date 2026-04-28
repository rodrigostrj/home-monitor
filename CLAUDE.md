# Home Monitor — Claude Code Project Memory

Full design doc: @./docs/PLAN.md

## Project

A home-readings platform. ESP32 + DHT22 today; will grow to energy, water,
weather, and air-quality integrations. Built by a senior .NET dev (20+ years)
*learning Go*, sometimes with a 9-year-old collaborator watching or helping.

## Domain (non-negotiable)

- Core entities: **Source** and **Reading**. Every metric — indoor temperature,
  humidity, kWh, m³ of water, AQI — is a `Reading` with a `metric_type`.
- **Never add sensor-specific or metric-specific tables.** New data sources =
  new Source rows + new widget components. No schema changes.
- Source `kind` is one of: `ESP32` (physical sensor boards) or `ExternalAPI`
  (integrations that pull from external APIs, e.g. Open-Meteo).
- Timestamps are UTC in storage, local on display.

## Phase status

- Phase 0 — Hardware hello world: **not started** (parts collected, not yet wired)
- Phase 0.5 — Docker foundation: **not started**
- Phase 1 — ESP32 publishes to backend: **firmware half done** (serial only, no WiFi/POST yet); backend done
- Phase 2a/2b — Data layer + handlers: **done**
- Phase 3 — Angular dashboard v1: **in design** (Option C mockup chosen, not yet built)
- Phase 4+ — not started

## Stack

- **Backend:** Go 1.22+, stdlib `net/http` (ServeMux), `sqlc`, `goose`,
  `modernc.org/sqlite`, `log/slog`, env-var config, `testify/assert`.
- **Frontend:** Angular 18+ standalone components, signals, Bootstrap 5,
  `ngx-echarts`. Widget-registry pattern.
- **Firmware:** PlatformIO, Arduino framework, ESP32 WROOM-32, DHT22 on GPIO 4.
- **Weather integration:** Separate Go service (`integrations/weather/`) that
  polls Open-Meteo and writes readings to the same API. Runs alongside the main
  API on the Raspberry Pi.

## Go conventions (I'm learning — this matters)

- **Idiomatic Go over ported .NET patterns.** No DI containers, no
  `IOptions<T>`-style wrappers, no Clean Architecture layering for its own
  sake. Constructor injection = passing arguments. Interfaces are satisfied
  implicitly; don't invent abstractions until there's a second implementation.
- **SQL-first via sqlc.** Don't suggest GORM or EF-Core-style ORMs.
- **stdlib first.** No chi, gin, echo, viper, cobra, wire unless the stdlib
  genuinely can't do the job. Justify every third-party dep.
- **Errors are values.** No panics in request paths. Wrap with `%w`.
- **Table-driven tests.** Keep test data next to the test.
- When explaining Go idioms, a brief ".NET analogy" aside is welcome; a Go
  solution that *looks like* the .NET version is not.

## Frontend conventions

- Standalone components only. No NgModules.
- Signals for state. Use RxJS only where streams are genuinely needed.
- No component-level CSS files — Tailwind utility classes in templates.
- Every widget goes under `frontend/src/app/widgets/<type>/` and registers
  itself in `widget-registry.ts`.

## ESP32 conventions

- `secrets.h` is git-ignored; `secrets.h.example` is committed.
- Batch all samples from one read cycle into a single POST body.
- 30-second sample interval unless explicitly changed.

## Working style

- **Small, visible steps.** Each change should end in something that runs.
  Prefer 10 commits a day over one giant one.
- **Explain as if a sharp 9-year-old might read over the shoulder.** Keep
  jargon honest but brief; skip chest-thumping framework tours.
- **Ask before adding dependencies, new tools, or new directories.**
- **Don't auto-run migrations, git commands, or destructive actions** without
  confirmation.
- Commit messages: conventional commits (`feat:`, `fix:`, `chore:`, `docs:`).
- When in doubt about scope, check `docs/PLAN.md` for the phase we're in.

## Out of scope (for now)

- Auth beyond per-source API keys.
- MQTT (stay on HTTP until we have 5+ devices).
- Kubernetes, Docker Swarm, cloud deploys (laptop-local during development).
- Frameworks on top of Angular (no NgRx, no Nx).
