# Home Monitor

A home-readings platform built by Rodrigo and his son.

Starts with an ESP32 + DHT22 temperature/humidity sensor. Designed to grow into
a household dashboard covering energy, water, air quality, and anything else
worth measuring.

See [docs/home-monitor-plan.md](docs/home-monitor-plan.md) for the full design doc.

## Structure

```
firmware/    ESP32 + DHT22 (PlatformIO)
backend/     Go API + SQLite
frontend/    Angular dashboard
docs/        Design docs and plans
```

## Quick start

See the plan doc for phase-by-phase setup instructions.
