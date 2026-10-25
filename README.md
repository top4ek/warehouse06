# warehouse06

A catalog of retro software for Soviet home computers: Vector-06C, BK, Radio-86RK, and more.
Content lives in the `storage/` git tree (markdown + files), is indexed in SQLite, and is served via a Go API and React SPA.
Content repo can be found [here](https://github.com/top4ek/retro_files).

Inspired by [caglrc.cc/scalar](https://caglrc.cc/scalar/categories/all/).

WARNING: Neuroslop ahead.

## Stack

| Layer | Technologies |
|-------|--------------|
| Backend | Go 1.27, chi, SQLite (FTS5), goldmark, bluemonday |
| Frontend | React 19, TypeScript, Vite, Ant Design |
| Emulator | [vector06js](https://github.com/svofski/vector06js) — vendored in `frontend/emulator-src/`, bundled at build time |
| Containers | Podman/Docker, Alpine 3.24.3; Debian 13 (E2E); Node.js 24.22.1 LTS; reflex (dev) |

## Repository layout

```
cmd/server/           — HTTP server entry point
internal/
  config/             — config (YAML + env)
  delivery/http/      — REST API
  parser/             — README.md and frontmatter parsing
  repository/         — SQLite + FTS
  sync/               — storage sync from git remote
  staticfiles/        — storage, frontend/dist, and emulator static files
frontend/
  src/                — React SPA
  emulator-src/       — vector06js sources (vendored, no nested .git)
  scripts/            — emulator bundle build and vendor updates
  public/emulator/    — build output (gitignored)
  dist/               — production SPA + emulator build
storage/              — content (platforms, authors, ROM/FDD/zip)
build/
  dev/                — dev image, reflex
  prod/               — production Containerfile, compose example
```

## Quick start

### Requirements

- Docker or Podman (with Compose)
- git
- make

Go, Node.js, and CGO toolchains run inside containers — nothing extra on the host.

Run `make help` for all targets.

### First-time setup

```bash
make setup
```

### Development

```bash
make dev
# or: make setup && make up-build
```

- Backend: http://localhost:8080 (reflex restarts Go on changes)
- Frontend: http://localhost:5173 (Vite HMR)

Health check: `curl http://localhost:8080/ping` → `pong`.

### Production

```bash
make prod    # copies build/prod/config.yaml.example if needed, then compose up --build
```

Compose file: [`build/prod/docker.compose.example.yaml`](build/prod/docker.compose.example.yaml). Example config: [`build/prod/config.yaml.example`](build/prod/config.yaml.example).

Production uses a persistent SQLite database in the `warehouse06-data` Docker volume (`file:/app/data/warehouse06.db`). Content is mounted from `storage/` at the repository root.

The image is built from [`build/prod/Containerfile`](build/prod/Containerfile): Node frontend build → Go server build → Alpine runtime.

Go stays on the `1.27` release line without a patch pin. Make build targets pull the latest base images; CI also checks for the latest Go 1.27 patch.

## Configuration

Example YAML config: [`build/dev/config.yaml.example`](build/dev/config.yaml.example).

| Parameter | Description |
|-----------|-------------|
| `storage_dir` | Path to the content directory (default `storage`) |
| `storage_url` | Git remote for periodic sync |
| `server.port` | HTTP port (8080) |
| `gopher.enabled` | Enable the read-only Gopher TCP service (default `true`, env `GOPHER_ENABLED`) |
| `gopher.listen_addr` | Internal TCP bind address (default `:7070`, env `GOPHER_LISTEN_ADDR`) |
| `gopher.host` | Public hostname advertised in menus (default `localhost`, env `GOPHER_HOST`) |
| `gopher.port` | Public port advertised in menus (default `7070`, env `GOPHER_PORT`) |
| `database.dsn` | SQLite DSN (`:memory:` or file path) |
| `sync.interval_hours` | Sync interval in hours; `0` — startup only |

Precedence: defaults → YAML → environment variables (env wins).

To load a YAML file, set `CONFIG_PATH` to its path. If `CONFIG_PATH` is unset or points to a missing file, the app uses only defaults + environment variables.

## Gopher

Gopher is enabled by default in development and production. Set `gopher.enabled: false` or `GOPHER_ENABLED=false` to disable it.
Connect to `gopher://localhost:7070/` in the development stack.
Menus provide platforms, authors, tags, search, entry cards, full text and original file downloads.
The service shares the current library catalog with HTTP and follows the same file extension restrictions.
Menus and generated text use UTF-8; menus are paginated and limited to 32 KiB, 800 lines, 255 bytes per line and 72 display columns per label.

For Lynx in a UTF-8 terminal, explicitly set the assumed document and display charsets:

```sh
lynx -assume_charset=UTF-8 -display_charset=UTF-8 gopher://localhost:7070/
```

Lynx's default assumed charset can garble Cyrillic Gopher menus even when the terminal uses UTF-8.

For a public deployment, set `gopher.host` in `build/prod/config.yaml` to the hostname clients can reach. Production Compose publishes `70:7070` by default, and the sample config advertises port 70 while keeping `gopher.listen_addr: ":7070"`.
The advertised public address is independent of the listener address.
Long search queries use compact page links backed by an in-memory FIFO cache of 512 distinct long queries.
Those links expire on restart or after eviction; the client receives an explicit search-expired error and can re-enter the query.
Short search queries remain embedded in their page links.
Requests must fit within 4096 bytes including CRLF and arrive within 60 seconds.
Transfers refresh a 60-second write timeout for each chunk of at most 512 bytes, allowing downloads to continue beyond one minute.
An enabled Gopher listener that cannot start causes the backend startup to fail.
Gopher closes its connections during backend shutdown.

This is a native Gopher TCP service; it does not expose Gopher menus over HTTP(S).
Acceptance is covered by automated TCP tests; testing on a physical Vector-06C is outside the scope of this feature.

## API

Interactive documentation (schemas, parameters, try-it-out) is served at `/api/docs` when the server is running, generated from the OpenAPI spec at [`internal/delivery/http/docs/openapi.yaml`](internal/delivery/http/docs/openapi.yaml).

All routes are under the `/api` prefix:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/status` | Sync status |
| GET | `/api/entries` | Entry list (filters, pagination) |
| GET | `/api/entries/search` | Full-text search (FTS) |
| GET | `/api/entries/{path}` | Single entry by path |
| GET | `/api/authors` | Author list |
| GET | `/api/authors/{dir}` | Author page |
| GET | `/api/tags` | Tags |
| GET | `/api/platforms` | Platforms |

Static files:

- `/` — React SPA from `frontend/dist`
- `/emulator/` — vector06js (build bundle)
- `/{platform}/...` — files from `storage/` (ROM, FDD, zip, images)

## Content (`storage/`)

Each entry is a directory with a `README.md` (frontmatter + markdown). Top-level platforms include `vector06c`, `bk`, `rk86`, `authors`, …

The syncer can pull updates from `storage_url` on startup (and on a schedule).

## Vector-06C emulator

The built-in emulator is vendored upstream code without a nested git repo:

1. Sources in [`frontend/emulator-src/`](frontend/emulator-src/), pinned versions in [`vendor.lock.json`](frontend/emulator-src/vendor.lock.json).
2. `npm run build:emulator` (via `make dev` or `make ci`) — concatenates JS and copies zip.js (Web Workers) and assets into `public/emulator/`.
3. Vite copies `public/emulator/` to `dist/emulator/`.

### Updating emulator upstream

```bash
make vendor-emulator                  # use refs from vendor.lock.json
make vendor-emulator REF=<sha>        # bump vector06js to a commit
```

Vendoring applies the local iframe/input and zip.js integration patch from
`frontend/scripts/emulator.patch`, and generates bin2wav types from
`frontend/scripts/bin2wav-tape.d.ts`. The check compares the complete generated
contents, including stale files, against the pinned upstream sources.

Commit vendored sources, the lock and adaptation changes together. Do not commit
`public/emulator/`.

## Tests

```bash
make test    # Go tests in dev backend container
make ci      # frontend build + tests (same as CI)
```

Go tests require CGO and the `sqlite_fts5` build tag (enabled automatically via `make test` / `make ci`):

```bash
CGO_ENABLED=1 go test -tags sqlite_fts5 ./...
```

Running bare `go test ./...` without these flags will fail on FTS5 schema initialization.

CI (`.github/workflows/ci.yml`) runs Go and frontend checks on GitHub runners,
Playwright E2E, vendor verification, production image build, Trivy scan and smoke test.
Successful pushes to `master` publish the Docker image. `make ci` runs the same
build and test checks locally in containers; vulnerability checks run separately
with `make vuln`, `npm audit` and Trivy.

## Make targets (summary)

| Target | Description |
|--------|-------------|
| `make help` | List all targets |
| `make setup` | Copy dev `.env` if missing |
| `make dev` | Start dev stack (`docker compose up --build`) |
| `make build` | Build production Docker image (`warehouse06:local`) |
| `make test` / `make ci` | Go tests / CI pipeline (in containers) |
| `make up` / `make prod` | Docker dev / production compose |
| `make vendor-emulator` | Refresh emulator sources (in container) |
| `make clean` | Remove build artifacts on bind mounts |

## Third-party licenses

- **vector06js** — BSD-style, Viacheslav Slavinsky ([svofski/vector06js](https://github.com/svofski/vector06js), [README](frontend/emulator-src/README.md))
- **i8080-js** — Alexander Demin ([begoon/i8080-js](https://github.com/begoon/i8080-js))
- **zip.js** — Gildas Lormeau ([gildas-lormeau/zip.js](https://github.com/gildas-lormeau/zip.js))
- **bin2wav** — Viacheslav Slavinsky ([svofski/bin2wav](https://github.com/svofski/bin2wav))
