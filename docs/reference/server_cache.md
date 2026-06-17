# Server Cache

> **Experimental:** Server caching is experimental, breaking changes may occur within minor releases. We believe the implementation is safe in practice — that is why it ships enabled by default (`all`). Set `HCLOUD_SERVER_CACHE_MODE=off` to opt out.

The server cache reduces calls to the Hetzner Cloud API made by the `InstancesV2` and routes controllers, which look up Servers by ID or name to reconcile Node and route state. The cache sits between the controllers and the Hetzner Cloud API; behavior is controlled by the environment variables below.

## Environment Variables

| Name                          | Type                | Default | Description                                                                                                                                                                                                                                                              |
| ----------------------------- | ------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `HCLOUD_SERVER_CACHE_MODE`    | `all \| one \| off` | `all`   | Selects the caching strategy. See [Modes](#modes) below.                                                                                                                                                                                                                 |
| `HCLOUD_SERVER_CACHE_MAX_AGE` | `duration`          | `10s`   | Default lifetime of cached entries. Individual controllers may override the default max age for specific lookups (e.g. the routes controller uses a longer max age). Accepts any Go `time.Duration` string (e.g. `30s`, `2m`). We don't recommend values above a minute. |

## Modes

### `all`

Fetches every Server in the project with paginated calls to `GET /servers` and serves all subsequent `ByID` / `ByName` lookups from the resulting snapshot until the max age is exceeded. The snapshot is refreshed on the next lookup after expiry.

### `one`

Caches each Server individually with its own expiration. A `ByID` / `ByName` lookup either returns a non-expired entry or issues a `GET /servers/{id}` (or `GET /servers?name=`) call and stores the result. Expired entries are evicted lazily when other entries are inserted.

### `off`

Disables caching entirely. Every lookup goes directly to the API.
