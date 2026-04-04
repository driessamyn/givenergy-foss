# givenergy-foss

A free, open source replacement for the GivEnergy cloud dashboard.
Monitor and control your GivEnergy inverter locally, without paying for a cloud subscription.

## Why this project?

GivEnergy have announced that their cloud portal is moving to a paid model.
Local API access remains free forever.

If you use Home Assistant, [givenergy-local](https://github.com/cdpuk/givenergy-local) already gives you local inverter monitoring.
This project takes a different approach: a **standalone, self-contained dashboard** with no dependency on Home Assistant or any other platform. Just your inverter, your hardware, and your data.

## Project aims

- **Data ownership** — rescue all your historical energy data from the GivEnergy cloud before the paywall hits, and keep future data under your control
- **Local-first monitoring** — a real-time dashboard powered entirely by your inverter's local interface
- **No subscription** — runs on your own hardware, no cloud account needed, but cloud-deployment friendly
- **Inverter control** — manage charge schedules, battery reserve, and operating modes
- **Pluggable storage** — choose where your data lives: local database, NAS, or cloud object storage
- **Feature parity** — replicate the key features of the GivEnergy cloud dashboard: power flow, historical charts, battery status, and system controls

## Planned architecture

The system will have three components that can run together on a single device or be split across your network.

```
                                +-----------------+
                                | GivEnergy Cloud |
                                +--------+--------+
                                         |
                               (one-time data export)
                                         |
                                         v
+-----------+  local protocol  +---------+---------+          +----------------+
|  Inverter +----------------->|     Collector     +--------->|    Dashboard   |
|  + dongle |    TCP :8899     |  polls inverter,  |          |  stores data,  |
+-----------+                  |  buffers locally   |          |  serves UI     |
                               +-------------------+          +----------------+
```

**[Importer](importer/)** — a command-line tool to download all your historical data from the GivEnergy cloud API. Run it once to rescue your data before the paywall kicks in. Supports resuming interrupted exports. **Available now.**

**Collector** — a lightweight service that will run on a Raspberry Pi (or similar) on your local network. It will connect to your inverter's WiFi dongle, read energy data at regular intervals, and forward it to the dashboard. Local buffering will ensure no readings are lost if the network drops.

**Dashboard** — a web application with a pluggable storage backend. Will receive data from the collector, store it, and serve a web interface showing real-time power flow, historical charts, battery status, and inverter controls.

## Quick start

The importer is the first available component. Download and run it in one line:

```sh
# Linux (x86_64)
curl -sL https://github.com/driessamyn/givenergy-foss/releases/latest/download/givenergy-import-linux-amd64 -o givenergy-import && chmod +x givenergy-import

# Linux (ARM64 / Raspberry Pi)
curl -sL https://github.com/driessamyn/givenergy-foss/releases/latest/download/givenergy-import-linux-arm64 -o givenergy-import && chmod +x givenergy-import

# macOS (Apple Silicon)
curl -sL https://github.com/driessamyn/givenergy-foss/releases/latest/download/givenergy-import-darwin-arm64 -o givenergy-import && chmod +x givenergy-import

# macOS (Intel)
curl -sL https://github.com/driessamyn/givenergy-foss/releases/latest/download/givenergy-import-darwin-amd64 -o givenergy-import && chmod +x givenergy-import
```

On Windows, download `givenergy-import-windows-amd64.exe` from the [latest release](https://github.com/driessamyn/givenergy-foss/releases/latest).

Then export your data:

```sh
# Create an API key at https://givenergy.cloud and set it
export GIVENERGY_API_TOKEN="your-api-key"

# Preview what will be exported
./givenergy-import --dry-run

# Export all historical data
./givenergy-import --output ./data
```

The importer downloads your energy data as CSV files, organised by date. It picks up where it left off if interrupted — just run the same command again.

You can also [build from source](importer/README.md#building) if you prefer.

## Contributing

This project is in its early stages. Help is especially welcome in these areas:

- Testing with other GivEnergy inverter models (Gen 1, Gen 2, Gen 3, AIO, AC)
- Mapping register values for different hardware
- Dashboard UI and UX design
- Documentation and deployment guides

## Licence

[Apache 2.0](LICENSE)
