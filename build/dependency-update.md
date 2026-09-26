# Dependency update — 2026-09-26

## Versions

| Component | Before | After |
|---|---|---|
| Go | 1.27.1 | 1.27.1 (current stable) |
| Node.js | floating 24 | 24.21.0 LTS |
| Alpine runtime | 3.24 (3.24.1 digest) | 3.24.2 |
| E2E OS | Debian 12 Bookworm | Debian 13 Trixie |
| golangci-lint | 2.13.2 | 2.14.0 |
| golang.org/x/net | 0.58.0 | 0.59.0 |
| golang.org/x/text | 0.41.0 | 0.42.0 |
| zip.js | legacy commit 563fe1d | 2.18.2, commit dd9e5b4 |
| `@tanstack/react-query` | ^5.102.8 | ^5.104.0 |
| `antd` | ^6.6.2 | ^6.6.5 |
| `dompurify` | ^3.4.14 | ^3.4.16 |
| `react` | ^19.2.8 | ^19.3.0 |
| `react-dom` | ^19.2.8 | ^19.3.0 |
| `react-router-dom` | ^7.18.3 | ^7.18.4 |
| `zod` | ^4.5.4 | ^4.6.5 |
| `@types/node` | ^26.4.1 | ^24.19.0 |
| `@types/react` | ^19.2.18 | ^19.3.0 |
| `@types/react-dom` | ^19.2.7 | ^19.3.0 |
| `eslint` | ^10.10.0 | ^10.11.0 |
| `eslint-plugin-react-refresh` | ^0.5.6 | ^0.5.7 |
| `jsdom` | ^30.0.1 | ^30.1.1 |
| `typescript-eslint` | ^8.69.0 | ^8.70.1 |
| `vite` | ^8.2.2 | ^8.3.1 |
| `vitest` | ^5.0.0 | ^5.0.2 |

Other direct Go/npm dependencies, Playwright 1.63.0, reflex 0.3.2 and the vector06js,
i8080-js and bin2wav commits already match their latest applicable releases.
Transitive npm dependencies were refreshed, including patched browserslist and
baseline-browser-mapping. GitHub Actions were refreshed to stable release tags.

TypeScript remains 6.0.3: typescript-eslint 8.70.1 declares `>=4.8.4 <6.1.0`,
so TypeScript 7.0.2 is deferred. Node 26 is Current rather than Active LTS;
Node 24 and its matching type definitions are used.

## Pinned images

| Image | Digest |
|---|---|
| `golang:1.27.1-alpine3.24` | `sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414` |
| `golangci/golangci-lint:v2.14.0-alpine` | `sha256:25925c95ebdc7aee39ecc0e6f4f33ac27b9a91718bc65fdf18a1a5c4c9325ee4` |
| `node:24.21.0-trixie` | `sha256:be40f6a87b9b22215ddb20da0a2320a5c6d583fe3ee3b0024d9fa4f05b40c8fd` |
| `node:24.21.0-alpine3.24` | `sha256:ebfe2f90462722a7a4de65e91990e97fe0d401c70e0e762c5b53302f905ec1c1` |
| `alpine:3.24.2` | `sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6` |

## Emulator integration

Local iframe/input adaptations and the modern zip.js Promise API integration are
reproducible via `frontend/scripts/emulator.patch`. Vendoring now copies wildcard
files, preserves symlinks, removes obsolete files, preserves bin2wav type declarations,
and compares generated contents against pinned sources. ZIP workers and WASM are
shipped alongside the library. REST API, configuration and database schema are unchanged.

## Validation

Passed on the already built development images:

- Go formatting, vet, golangci-lint (0 issues), all tests with race and FTS5.
- Frontend typecheck, ESLint, 30 unit tests and production frontend build.
- 4 vendor script tests, including wildcard, content drift, stale-file and symlink checks.
- All 9 Playwright E2E tests: input controls, scrolling, ROM/FDD and deflated ZIP loading.
- Full vendor content verification; npm audit and govulncheck: 0 vulnerabilities.
- Running backend health check and live FTS5 search against the existing catalog.
- `git diff --check`.

The final user instruction selected the already built dev environment and tests.
The updated production image was not built or scanned with Trivy; its container smoke
test and persistent-database compatibility check remain unverified. The existing local
production image predates these changes and was not used as proof of their correctness.

Docker bridge downloads reset connections on this host. Network-dependent audit and
vendor checks used the already built E2E image with host networking. A temporary
Compose override added the missing localhost mapping for host-network E2E.

Development remains running at http://localhost:5173 and http://localhost:8080.
It started without image builds or npm ci; a temporary override uses the installed
node_modules and disables remote content sync. The storage working tree is unchanged.

For production builds on hosts with Docker bridge download issues,
`make build BUILD_NETWORK=host` selects host networking; the default remains unchanged.
