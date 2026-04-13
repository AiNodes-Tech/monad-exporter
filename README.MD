# Monad-exporter

`monad-exporter` exposes a Prometheus `/metrics` HTTP endpoint and publishes Monad node status metrics.

## What It Does

- Collects local node metrics (via `METRICS_URL`).
- Checks JSON-RPC availability and additional node parameters.
- Can read staking data via `eth_call` (optional).
- Can fetch a Secp public key once at startup via `monad-keystore` (optional).

## Requirements

- Go `1.22+`
- Access to Monad node processes/utilities (if the related metrics are enabled)

## Project Structure

- Source code: `src/`
- Entry point: `src/cmd/monad-exporter/main.go`
- Build output binary: `bin/monad-exporter`

## Prepare

Install Go (if needed):

```bash
sudo apt update
sudo apt install golang-go -y
```

## Quick Start

1. Copy environment variables template:

```bash
cp .env.example .env
```

2. Edit `.env` if needed.

3. Build the binary:

```bash
make build
```

4. Run:

```bash
./bin/monad-exporter --config .env
```

## Add a systemd service

```bash
sudo tee /etc/systemd/system/monad-exporter.service > /dev/null <<EOF
[Unit]
Description=Monad Exporter
Wants=network-online.target
After=network-online.target

[Service]
ExecStart=/home/monad/monad-exporter/bin/monad-exporter
WorkingDirectory=/home/monad/monad-exporter
User=monad
Restart=always
RestartSec=5
EnvironmentFile=/home/monad/monad-exporter/.env

[Install]
WantedBy=multi-user.target
EOF
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable monad-exporter.service
sudo systemctl start monad-exporter.service
sudo systemctl status monad-exporter.service
```

After startup:

- Metrics: `http://127.0.0.1:9101/metrics` (or your configured port)
- Healthcheck: `http://127.0.0.1:9101/healthz`

> If the port is busy, set another one via `PORT` or `MONAD_EXPORTER_LISTEN`.

## Configuration

The exporter reads environment variables.  
The `--config .env` flag loads variables from a file, but current process environment variables take precedence.

Main variables:

- `MONAD_EXPORTER_LISTEN` - HTTP listen address (for example, `:9101`)
- `PORT` - alternative way to set port (for example, `9101`)
- `MONAD_HOME` - node home directory
- `METRICS_URL` - local node metrics URL
- `RPC_URL` - local JSON-RPC URL
- `EXTERNAL_RPC_TEMPLATE` - external RPC template (`%s` = network name)
- `MONAD_EXPORTER_HTTP_TIMEOUT` - HTTP timeout (for example, `15s`)
- `MPT_STORAGE` - path to MPT storage
- `RPC_LISTEN_PORT` - port used for RPC listen checks

Optional:

- `MONAD_EXPORTER_ENABLE_KEYSTORE=1` - enable one-time Secp pubkey read via `monad-keystore`
- `MONAD_EXPORTER_SECP_PUBLIC_KEY` - provide Secp pubkey directly (without keystore call)
- `MONAD_EXPORTER_ENABLE_STAKING=1` - enable staking metrics
- `MONAD_STAKING_ADDRESS` - staking precompile address (default: `0x0000000000000000000000000000000000001000`)
- `VALIDATOR_ID` - validator id for staking metrics
- `MONAD_EXPORTER_ENABLE_COINGECKO=1` - fetch MON/USD from CoinGecko (cached; see `MONAD_EXPORTER_COINGECKO_*` in `.env.example`)

See the full template in `.env.example`.

## Prometheus Scrape Config

Example:

```yaml
scrape_configs:
  - job_name: monad-exporter
    static_configs:
      - targets: ["127.0.0.1:9101"]
```

## Development

Build:

```bash
make build
```

Run locally without building:

```bash
cd src
go run ./cmd/monad-exporter --config ../.env
```
