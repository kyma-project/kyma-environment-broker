<!--{"metadata":{"publish":false}}-->

# KEB Parameter Usage Analytics

The `keb-analytics` binary provides a self-contained analytics UI that shows which provisioning and update parameters were used across all active Kyma instances.

## Architecture

`keb-analytics` is a separate Go binary deployed alongside KEB in the same Helm chart. It connects directly to the KEB PostgreSQL database, aggregates parameter usage statistics, and caches them in memory. It exposes a web UI and a JSON API — both without authentication, as network access is restricted to the cluster.

```
Browser
  └─► GET /   → embedded HTML+JS UI
  └─► GET /api/stats → cached aggregation JSON
                            └─► every 60 min: queries PostgreSQL directly
                                    └─► active instances only
```

## Configuration

`keb-analytics` is configured via environment variables (prefix `APP_`):

| Variable | Default | Description |
|---|---|---|
| `APP_DATABASE_HOST` | `localhost` | PostgreSQL host |
| `APP_DATABASE_PORT` | `5432` | PostgreSQL port |
| `APP_DATABASE_USER` | `postgres` | PostgreSQL user |
| `APP_DATABASE_PASSWORD` | `password` | PostgreSQL password |
| `APP_DATABASE_NAME` | `broker` | PostgreSQL database name |
| `APP_DATABASE_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `APP_PORT` | `8080` | HTTP port for the analytics server |
| `APP_REFRESHINTERVAL` | `1h` | How often to refresh the in-memory stats cache |

## HTTP Endpoints

### `GET /`

Serves the embedded single-page analytics UI. No authentication required.

### `GET /api/stats`

Returns a JSON object with aggregated parameter usage statistics.

**Query parameters:**

| Parameter | Format | Description |
|---|---|---|
| `from` | `YYYY-MM-DD` | Start of time range (filters by provisioning/update operation creation date) |
| `to` | `YYYY-MM-DD` | End of time range |
| `plan` | string | Filter by plan name (e.g. `aws`, `azure`, `gcp`, `trial`) |
| `region` | string | Filter by provisioning region |

All parameters are optional. Omitting `from`/`to` returns stats for all active instances from the in-memory cache. Providing a time range triggers a live DB query.

**Response schema:**

```json
{
  "total_instances": 1234,
  "provisioning": {
    "parameters": [
      { "parameter": "region",      "set_count": 1200, "total": 1234 },
      { "parameter": "machineType", "set_count":  950, "total": 1234 }
    ]
  },
  "updates": {
    "parameters": [
      { "parameter": "machineType", "set_count": 320, "total": 410 }
    ]
  },
  "distributions": [
    {
      "parameter": "machineType",
      "values": { "m6i.xlarge": 410, "Standard_D4_v3": 280 }
    }
  ],
  "plans": ["aws", "azure", "gcp", "trial"],
  "regions_by_plan": {
    "aws":   ["eu-central-1", "us-east-1"],
    "azure": ["westeurope", "eastus"]
  }
}
```

`set_count` is the number of instances that had the parameter explicitly set. Parameters are sorted by `set_count` descending.

### `POST /api/refresh`

Triggers an immediate out-of-band refresh of the in-memory cache by re-querying the database. Returns `204 No Content`.

## UI Views

The UI is a single-page application with four tabs:

| Tab | Description |
|---|---|
| **Provisioning** | Ranked bar chart of provisioning parameter usage (% of instances with each parameter set) |
| **Update** | Same chart scoped to update operations |
| **Combined** | Provisioning and update stats merged into one chart |
| **Value Distribution** | Bar chart of distinct values for a selected parameter (e.g. `machineType` breakdown) |

Global filters (Period, Plan, Region) apply to all tabs.
