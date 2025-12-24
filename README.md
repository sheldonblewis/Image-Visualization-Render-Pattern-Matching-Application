# Image-Visualization-Render-Pattern-Matching-Application

A high-performance web application for visualizing and exploring images stored in Google Cloud Storage with pattern-based querying capabilities.

## Features

- Pattern-based image querying with `%variable%` syntax
- Efficient GCS object listing with minimal directory scanning
- Responsive grid layout with lazy loading
- Dynamic grouping of results by captured variables
- Infinite scroll pagination
- Full-screen image viewer
- Column count adjustment (2/4/6/8 columns)
- Keyboard navigation
- Optimized for large datasets (100,000+ images)

## Tech Stack

### Backend (Go)
- **Framework**: Standard Library + Gorilla Mux
- **GCP**: cloud.google.com/go/storage (gRPC-enabled)
- **Concurrency**: Native Go routines with worker pool
- **API**: RESTful JSON API

### Frontend (React + TypeScript)
- **UI**: React 18 + Vite
- **State Management**: TanStack Query + React Context
- **Virtualization**: react-window
- **Styling**: Tailwind CSS
- **Icons**: Lucide React

## Prerequisites

- Go 1.21+ (for backend)
- Node.js 18+ and npm/yarn (for frontend)
- Google Cloud SDK (for local development with GCS)

## Getting Started

### Backend Setup

1. Navigate to the backend directory:
   ```bash
   cd backend
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Set up environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. Run the server:
   ```bash
   go run cmd/server/main.go
   ```

### Frontend Setup

1. Navigate to the frontend directory:
   ```bash
   cd frontend
   ```

2. Install dependencies:
   ```bash
   npm install
   # or
   yarn
   ```

3. Start the development server:
   ```bash
   npm run dev
   # or
   yarn dev
   ```

### One-command dev loop

After installing backend (`go mod tidy`) and frontend (`npm install`) deps once, you can run both servers together:

```bash
./scripts/dev.sh
```

This script:

1. Finds your Go binary (respects `GO_BIN` env var if set).
2. Starts the backend on port `8080`.
3. Starts the frontend dev server on port `5173` with HMR.
4. Handles Ctrl+C cleanly by terminating both processes.

## Architecture

### Backend

- `config/` centralizes runtime defaults (port, CORS, page sizes) and parses env overrides.
- `service/` contains the core query pipeline:
  - `pattern_percent.go` and `pattern_regex.go` parse `%capture%` or named-regex tokens into compiled matchers while tracking literal prefixes to minimize listings.
  - `query_service.go` plans recursive listing jobs, performs cursor-based pagination, and converts matches into API responses.
  - `errors.go` propagates validation issues as HTTP 400 responses.
- `storage/` exposes two interchangeable clients:
  - `http_client.go` uses the public JSON API (no auth required for public buckets).
  - `gcs_client.go` wraps the official Go SDK for gRPC-enabled, authenticated deployments.
- `cmd/server/main.go` wires config, storage client, query service, Gorilla Mux routing, and CORS middleware.

### Frontend

- Built with Vite + React + TypeScript for instant dev feedback.
- `src/App.tsx` orchestrates TanStack Query, the pattern form, grouping controls, and modal viewer.
- `src/lib/transform.ts` enriches API results with grouping metadata for virtualization.
- `src/components/GridViewport.tsx` uses `react-window` for incremental rendering and lazy-loaded thumbnails with shimmer placeholders.
- `src/components/ViewerModal.tsx` provides full-screen previews with keyboard navigation.
- Tailwind CSS powers the theme; see `tailwind.config.cjs` for the palette and fonts.

### Testing

- Backend: `go test ./...`
- Frontend: `npm run build` ensures type-check + production bundle; optionally run `npm run lint`.

## API Documentation

### `POST /api/query`

Query images based on a pattern.

**Request:**
```json
{
  "pattern": "gs://wlt-public-sandbox/imgrid-takehome/%exp%/%class%_%idx%.jpg",
  "mode": "percent",
  "pageSize": 50,
  "cursor": "optional-cursor-from-previous-response"
}
```

**Response:**
```json
{
  "captureNames": ["exp", "class", "idx"],
  "items": [
    {
      "object": "imgrid-takehome/exp1/0000_00.jpg",
      "url": "https://storage.googleapis.com/wlt-public-sandbox/imgrid-takehome/exp1/0000_00.jpg",
      "captures": {
        "exp": "exp1",
        "class": "0000",
        "idx": "00"
      }
    }
  ],
  "nextCursor": "optional-next-page-cursor",
  "stats": {
    "scannedPrefixes": 5,
    "scannedObjects": 100,
    "matched": 50
  }
}
```

## Development

### Backend

- Run tests:
  ```bash
  go test ./...
  ```

- Build:
  ```bash
  go build -o bin/server cmd/server/main.go
  ```

### Frontend

- Run linter:
  ```bash
  npm run lint
  ```

- Build for production:
  ```bash
  npm run build
  ```

## Deployment

### Backend

The backend can be deployed to any platform that supports Go applications, such as:
- Google Cloud Run
- AWS App Runner
- Heroku
- Self-hosted server

### Frontend

The frontend can be deployed to any static hosting service, such as:
- Vercel
- Netlify
- GitHub Pages
- Firebase Hosting

## AI Assistance

Development leveraged an AI pair-programming assistant (Cascade) for scaffolding, code generation, and refactors. All AI-authored changes were reviewed, formatted, and tested (`go test ./...`, `npm run build`) before inclusion.

## License

MIT
