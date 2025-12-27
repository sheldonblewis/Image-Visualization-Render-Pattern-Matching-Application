# Image Grid Viewer

An interactive web UI for exploring Google Cloud Storage datasets by pattern. Provide a `gs://bucket/path/%capture%/...` template or named-regex and browse the matching images in a responsive grid with keyboard navigation and a full-screen viewer.

## Quick Start

### Requirements

- Go 1.21+
- Node.js 18+ (npm)
- Access to a GCS bucket (defaults to `wlt-public-sandbox`)

### Install

```bash
go mod tidy ./backend
npm install --prefix frontend
```

### Run (single command)

```bash
./scripts/dev.sh
```

The script boots the Go API on `:8080` and the Vite dev server on `:5173`. Visit http://localhost:5173 to use the app.

> Need custom config? Set env vars (e.g. `PORT`, `GCS_BUCKET`, `ALLOWED_ORIGINS`) before running the script.

## Using the App

1. **Enter a pattern** – examples:
   - Percent tokens: `gs://bucket/imgrid/%exp%/%class%_%idx%.jpg`
   - Regex mode: `gs://bucket/(?P<exp>[^/]+)/(?P<class>\d+)_00.jpg`
2. **Choose mode** – *percent* (default) or *regex*.
3. **Run query** – results stream into the grid with infinite scroll.
4. **Group & layout** – select any capture name to group rows; adjust column count.
5. **Inspect images** – click a tile for the viewer, use ←/→/Esc for keyboard navigation. Viewer blocks wrap-around until the full dataset loads and shows a “Loading more…” sentinel while fetching the next batch.

### Pattern Tips

- `%token%` matches any non-slash characters.
- `%%` escapes a literal `%`.
- Regex mode follows Go-style named capture groups (`?P<name>`).
- The literal prefix of your pattern is used to minimize GCS listings—add as much concrete pathing as possible for best performance.

### Keyboard Shortcuts

- **Grid**: Page scroll via mouse/trackpad; fetch more as you near the bottom.
- **Viewer**: `←` / `→` switch images, `Esc` closes. Wrap-around unlocks once all pages are loaded.

## Troubleshooting

- **No results**: Double-check the bucket/path and ensure your captures align with actual filenames.
- **Slow scans**: Long-running listings are expected on unbounded prefixes. Narrow the pattern or increase backend worker count (`WORKER_COUNT`) if needed.
- **CORS**: Update `ALLOWED_ORIGINS` when hosting the frontend separately.

## API Reference

### `POST /api/query`

```jsonc
{
  "pattern": "gs://bucket/%exp%/%class%_00.jpg",
  "mode": "percent",
  "pageSize": 120,
  "cursor": null
}
```

Response includes the capture names, an array of items, cursor for pagination, and scan stats.

### `POST /api/count`

Returns `{ "total": <int>, "stats": { ... } }` for the same pattern parameters. Used by the UI to display total match count without hydrating every page.

## Development Reference

- **Backend tests**
  ```bash
  cd backend && go test ./...
  ```
- **Frontend checks**
  ```bash
  cd frontend
  npm run lint
  npm run build
  ```
- **Production build**
  - Backend: `go build -o bin/server ./backend/cmd/server`
  - Frontend: `npm run build --prefix frontend` (outputs to `frontend/dist`)

## Deployment Notes

- Backend is a stateless Go HTTP service; deploy to Cloud Run, App Runner, or any VM.
- Frontend is a static Vite bundle; host on Vercel, Netlify, Cloud Storage, etc.
- Configure environment variables for bucket access, request timeouts, worker counts, and allowed origins per environment.

## License

MIT
