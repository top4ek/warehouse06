# vector06js (vendored)

JavaScript emulator for the [Vector-06C](https://en.wikipedia.org/wiki/Vector-06C) computer.
Upstream: [svofski/vector06js](https://github.com/svofski/vector06js).

Pinned versions are recorded in [`vendor.lock.json`](vendor.lock.json).

## Layout

| Path | Origin | Used at runtime |
|------|--------|-----------------|
| `src/` | vector06js | bundled |
| `i8080-js/` | [svofski/i8080-js](https://github.com/svofski/i8080-js) | bundled |
| `wav.js/` | vector06js | bundled |
| `zip.js/` | [gildas-lormeau/zip.js](https://github.com/gildas-lormeau/zip.js) | copied as-is (Web Workers) |
| `*.png` | vector06js | copied as-is |
| `index.html` | vector06js (adapted) | copied to `public/emulator/` |

Build output lands in `frontend/public/emulator/` (gitignored) via `npm run build:emulator`.

## Update upstream

```bash
cd frontend
npm run vendor:emulator              # refresh to refs in vendor.lock.json
npm run vendor:emulator -- --ref <sha>   # bump vector06js ref, then re-vendor all
npm run build:emulator
```

Local integration changes are applied from `../scripts/emulator.patch`; bin2wav types
are generated from `../scripts/bin2wav-tape.d.ts`. Update these adaptations when
upstream changes. `npm run vendor:emulator -- --check` compares generated contents
with the pinned sources and adaptations, including added or removed files.

Commit vendored sources, lock and any adaptation changes together.
