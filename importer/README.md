# givenergy-import

Command-line tool to export your historical energy data from the GivEnergy cloud API. Produces CSV files organised by date, ready for analysis or import into the dashboard (when available).

Pre-built binaries for common platforms will be published soon. For now, build from source.

## Building

Requires [Go 1.23+](https://go.dev/dl/).

```sh
cd importer
go build -o givenergy-import .
```

Cross-compile for other platforms:

```sh
# Raspberry Pi (64-bit)
GOOS=linux GOARCH=arm64 go build -o givenergy-import-linux-arm64 .

# Linux server
GOOS=linux GOARCH=amd64 go build -o givenergy-import-linux-amd64 .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o givenergy-import-darwin-arm64 .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o givenergy-import-darwin-amd64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o givenergy-import.exe .
```

## Usage

You need a GivEnergy cloud API key. Create one at [givenergy.cloud](https://givenergy.cloud) under Account Settings > API Keys.

Set it as an environment variable or pass it directly:

```sh
# Via environment variable (recommended)
export GIVENERGY_API_TOKEN="your-api-key"
./givenergy-import --output ./data

# Via flag
./givenergy-import --token "your-api-key" --output ./data
```

### Options

```
--token string       GivEnergy API token (or set GIVENERGY_API_TOKEN)
--output string      Output directory (default: ./data)
--from string        Start date YYYY-MM-DD (default: inverter commission date)
--to string          End date YYYY-MM-DD (default: yesterday)
--rate-limit int     Max API requests per minute (default: 250)
--dry-run            Show what would be exported without making data requests
--verbose            Log each date as it completes
--version            Print version and exit
```

### Examples

```sh
# Preview the export plan
./givenergy-import --dry-run

# Export everything from the beginning
./givenergy-import --output ./data

# Export a specific date range
./givenergy-import --from 2024-01-01 --to 2024-12-31

# Resume an interrupted export (just run the same command again)
./givenergy-import --output ./data
```

## Output format

Data is written as CSV files, one per day, in a year/month folder structure:

```
data/
  export-metadata.json
  {inverter-serial}/                 # e.g. CE1234G567
    data-points/
      2023/
        05/
          2023-05-18.csv
          2023-05-19.csv
        06/
          ...
      2024/
        ...
    events/
      events.csv
```

Each data-points CSV contains one row per reading (roughly every five minutes) with columns for solar generation, grid import/export, battery state, consumption, and inverter status.

The events CSV contains system events (e.g. BMS alerts) with start and end times.

`export-metadata.json` records inverter details, the date range, and export statistics.

## Resuming

The importer checks which date files already exist before fetching. Dates with a complete CSV are skipped. If an export is interrupted, run the same command again — it picks up where it left off.

## Rate limiting

The GivEnergy API allows 300 requests per minute. The importer defaults to 250 req/min to leave headroom. A full export of three years of data takes roughly 10-15 minutes.

## Multiple inverters

If your account has multiple inverters, the importer exports data for all of them. Each inverter's data is stored under its own serial number directory.

## Licence

[Apache 2.0](../LICENSE)
