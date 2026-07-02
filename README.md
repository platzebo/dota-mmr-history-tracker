# Dota MMR History Tracker

Local-first Dota 2 MMR history tracker for Linux and Windows.

It logs into Steam locally via Steam Mobile QR auth, connects to the Dota 2 GameCoordinator, reads your own match history, extracts `previous_rank` + `rank_change`, stores the result in SQLite, and serves a local dashboard.

No manual MMR entry. No memory reading. No injection. No overlay hooks. No Dota client patching.

## Features

- QR-based Steam authentication; no Steam password flow in the app
- Automatic incremental sync with `--auto`
- Backfill older ranges with `--skip-pages`
- Local SQLite database
- Local browser dashboard with MMR graph, hover tooltips, hero names, hero stats, and recent matches
- CSV export
- GitHub Actions build artifacts for Linux and Windows

## Download / build

### Option A: GitHub Actions artifact

On GitHub, open:

```text
Actions -> build -> latest successful run -> Artifacts
```

Download the binary for your platform:

```text
dota-mmr-history-tracker-linux-amd64
dota-mmr-history-tracker-linux-arm64
dota-mmr-history-tracker-windows-amd64.exe
```

On Linux, make it executable:

```bash
chmod +x ./dota-mmr-history-tracker-linux-amd64
```

### Option B: build locally

Requires Go 1.26+.

```bash
go test ./...
go build -o dist/dota-mmr-history-tracker ./cmd/dota-mmr-history-tracker
```

Cross-compile examples:

```bash
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o dist/dota-mmr-history-tracker-linux-amd64 ./cmd/dota-mmr-history-tracker
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o dist/dota-mmr-history-tracker-windows-amd64.exe ./cmd/dota-mmr-history-tracker
```

## Quick start

Close Dota 2 before syncing. Steam may stay open.

```bash
./dota-mmr-history-tracker-linux-amd64 sync \
  --username your_steam_username \
  --qr \
  --auto \
  --matches 1000 \
  --timeout 10m
```

The tool prints a terminal QR code. Scan/confirm it with the Steam mobile app. After the QR confirmation it connects to Steam CM + Dota GC and imports new ranked MMR rows.

Then start the dashboard:

```bash
./dota-mmr-history-tracker-linux-amd64 serve
```

Open:

```text
http://127.0.0.1:8789
```

## Commands

### Incremental sync

Use this for normal daily/weekly updates:

```bash
./dota-mmr-history-tracker-linux-amd64 sync \
  --username your_steam_username \
  --qr \
  --auto \
  --matches 1000 \
  --timeout 10m
```

`--auto` loads known match IDs from SQLite, starts at the newest GameCoordinator page, imports only rows newer than the first known match, and stops when it sees a known `match_id`.

### Initial import / newest history

```bash
./dota-mmr-history-tracker-linux-amd64 sync \
  --username your_steam_username \
  --qr \
  --matches 5000 \
  --timeout 45m
```

### Backfill older history

One Dota GC page is 20 raw match-history rows. To start after roughly the newest 4,000 rows:

```bash
./dota-mmr-history-tracker-linux-amd64 sync \
  --username your_steam_username \
  --qr \
  --skip-pages 200 \
  --matches 5000 \
  --timeout 45m
```

Then continue farther back with `--skip-pages 400`, `--skip-pages 600`, etc. Imports use SQLite upserts, so repeated windows do not duplicate matches.

### Raw debug dump

```bash
./dota-mmr-history-tracker-linux-amd64 sync \
  --username your_steam_username \
  --qr \
  --matches 500 \
  --dump-raw ./dota-gc-history.jsonl
```

### Serve dashboard

```bash
./dota-mmr-history-tracker-linux-amd64 serve
```

Custom address:

```bash
./dota-mmr-history-tracker-linux-amd64 serve --addr 127.0.0.1:8790
```

### Export CSV

```bash
./dota-mmr-history-tracker-linux-amd64 export --out dota-mmr-history-tracker.csv
```

CSV columns:

```text
Date,Unix time,MatchID,Solo Queue,HeroID,Start MMR,Rank Change
```

## Data location

Default SQLite database:

- Linux: `$XDG_CONFIG_HOME/dota-mmr-history-tracker/ledger.sqlite` or `~/.config/dota-mmr-history-tracker/ledger.sqlite`
- Windows: `%AppData%\\dota-mmr-history-tracker\\ledger.sqlite`

Override it:

```bash
./dota-mmr-history-tracker-linux-amd64 sync --db ./ledger.sqlite --username your_steam_username --qr --auto
./dota-mmr-history-tracker-linux-amd64 serve --db ./ledger.sqlite
```

## Authentication model

The app intentionally supports QR auth and access-token auth only.

Password + Steam Guard code entry is not exposed because it is brittle, timing-sensitive, and encourages unsafe credential handling. QR auth is the intended user flow.

Advanced users can pass an existing Steam access token:

```bash
STEAM_ACCESS_TOKEN='...' ./dota-mmr-history-tracker-linux-amd64 sync --username your_steam_username --auto
```

## Rate-limit note

One sync request/page reads 20 raw history rows. Use conservative batch sizes for large backfills and keep the default `--page-delay 1s` enabled for GameCoordinator pacing.

## Architecture

```text
Go CLI
  -> Steam QR auth
  -> Steam CM login with token
  -> mark app 570 as playing
  -> Dota 2 GC ClientHello
  -> CMsgDOTAGetPlayerMatchHistory pages
  -> normalize previous_rank + rank_change rows
  -> SQLite upsert
  -> local HTTP dashboard + CSV export
```

## Security / scope

- Runs locally on your machine
- Does not host a remote service
- Does not ask for Steam password
- Does not persist Steam tokens yet
- Does not inspect Dota memory
- Does not inject into or modify the Dota client
- Uses an undocumented GameCoordinator/protobuf interface that can break if Valve changes it
