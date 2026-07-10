# Warehouse06 frontend

React 19 + TypeScript + Vite + Ant Design SPA for the retro software catalog.

## Scripts

```bash
npm run dev          # emulator bundle + Vite dev server
npm run build        # production build (emulator + tsc + vite)
npm test             # Vitest unit tests
npm run lint         # ESLint
```

API requests are validated with Zod schemas in `src/api/schemas.ts`; TypeScript types are inferred from those schemas.

Data fetching uses TanStack Query (`src/hooks/useApiQueries.ts`) with shared cache keys in `src/api/queryKeys.ts`.

## Layout

- `src/pages/` — route pages
- `src/components/` — UI components
- `src/hooks/` — React hooks (`useCatalog`, query hooks, list restore)
- `src/api/` — HTTP client and schemas
- `src/lib/` — utilities (browse filters, emulator routes, HTML sanitization)
