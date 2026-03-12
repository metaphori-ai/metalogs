# metalogs

Local dev logging system backed by SQLite. Aggregate logs from multiple apps and layers into a single queryable store.

Designed for multi-site architectures where a shared API backend serves several frontends and BFFs — so when something breaks, you can trace errors across the entire stack.

## Install

```bash
go install github.com/metaphori-ai/metalogs/cmd/metalogs@latest
```

Or from source:

```bash
git clone git@github.com:metaphori-ai/metalogs.git
cd metalogs
./install.sh
```

## Concepts

Every log entry has a **site** (product identity) and a **layer** (deployment tier):

| Site | Layers |
|------|--------|
| `metaphori-ai` | `api` |
| `contextmax-ai` | `cm-bff`, `cm-fe` |
| `truememory-ai` | `tm-bff`, `tm-fe`, `tm-mobile` |
| `truearchitect-ai` | `ta-bff`, `ta-fe`, `ta-dev` |

**Collections** are named groups of site+layer pairs for cross-cutting queries.

All site, layer, and collection names are case-insensitive (normalized to lowercase).

## Quick Start

Start the server:

```bash
metalogs serve
# or with options
metalogs serve --port=9999 --ttl=14d
```

Send a log:

```bash
curl -X POST http://localhost:9999/ingest \
  -H "Content-Type: application/json" \
  -d '{"site":"truememory-ai","layer":"tm-bff","level":"error","message":"db timeout","details":"connection refused on port 5432"}'
```

Query:

```bash
metalogs query --site=truememory-ai --level=error --since=1h
```

## Collections

Group site+layer pairs for common queries:

```bash
# Create a collection for the TrueMemory full stack
metalogs collections create truememory \
  metaphori-ai:api,truememory-ai:tm-bff,truememory-ai:tm-fe,truememory-ai:tm-mobile

# Query errors across the whole stack
metalogs query --collection=truememory --level=error --since=1h

# List collections
metalogs collections list

# Delete
metalogs collections delete truememory
```

## CLI Reference

```
metalogs serve    [--port=9999] [--ttl=7d] [--db=PATH]
metalogs query    [--site] [--layer] [--collection] [--level] [--since] [--contains] [--limit] [--offset] [--json]
metalogs cleanup  [--older-than=7d]
metalogs sites    [--site=NAME]
metalogs collections list
metalogs collections create <name> <site:layer,site:layer,...>
metalogs collections delete <name>
```

Global flag: `--db` overrides the default database path (`~/.metalogs/metalogs.db`).

## HTTP API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/ingest` | Ingest a single log entry |
| `POST` | `/ingest/batch` | Ingest an array of log entries |
| `GET` | `/query` | Query logs (params: `site`, `layer`, `sites`, `layers`, `collection`, `level`, `since`, `until`, `contains`, `limit`, `offset`) |
| `GET` | `/sites` | List all site+layer pairs |
| `GET` | `/collections` | List all collections |
| `POST` | `/collections` | Create a collection (`{"name":"...","members":[{"site":"...","layer":"..."}]}`) |
| `DELETE` | `/collections/{name}` | Delete a collection |
| `POST` | `/cleanup` | Trigger cleanup (optional `older_than` param) |
| `GET` | `/health` | Health check |

## Log Entry Format

```json
{
  "site": "truememory-ai",
  "layer": "tm-bff",
  "level": "error",
  "message": "request failed",
  "details": "stack trace or extended info",
  "source": "handlers.go:42",
  "timestamp": "2025-03-12T10:30:00Z",
  "metadata": {"request_id": "abc-123", "user_id": "u-456"}
}
```

Required: `site`, `layer`, `level`, `message`. Everything else is optional. Timestamp defaults to server time if omitted.

## Go Library

Import `pkg/metalogs` directly for zero-network-hop integration:

```go
import "github.com/metaphori-ai/metalogs/pkg/metalogs"

store, err := metalogs.NewStore("") // defaults to ~/.metalogs/metalogs.db
defer store.Close()

// Ingest
store.Ingest(metalogs.LogEntry{
    Site:    "metaphori-ai",
    Layer:   "api",
    Level:   metalogs.LevelError,
    Message: "something broke",
})

// Query
since := time.Now().Add(-1 * time.Hour)
results, err := store.Query(metalogs.QueryOpts{
    Collection: "truememory",
    Levels:     []metalogs.LogLevel{metalogs.LevelError},
    Since:      &since,
})

// Collections
store.CreateCollection("truememory", []metalogs.SiteLayer{
    {Site: "metaphori-ai", Layer: "api"},
    {Site: "truememory-ai", Layer: "tm-bff"},
    {Site: "truememory-ai", Layer: "tm-fe"},
})
```

## Auto-Cleanup

When the server is running, a background goroutine purges logs older than the configured TTL (default 7 days) every hour. Use `metalogs cleanup --older-than=7d` for manual cleanup.

## License

MIT
