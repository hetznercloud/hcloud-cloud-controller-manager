# Instance Cache

> **Experimental:** Instance caching is experimental, breaking changes may occur within minor releases. We believe the implementation is safe in practice — that is why it ships enabled by default (`all-server`). Set `HCLOUD_CACHE_MODE=off` to opt out.

The instance cache reduces calls to the Hetzner Cloud API made by the `InstancesV2` controller, which looks up Servers by ID or name to reconcile Node state. The cache sits between the controller and the Hetzner Cloud API; behavior is controlled by the environment variables below.

## Environment Variables

| Name                | Type                | Default | Description                                                                           |
| ------------------- | ------------------- | ------- | ------------------------------------------------------------------------------------- |
| `HCLOUD_CACHE_MODE` | `all \| one \| off` | `all`   | Selects the caching strategy. See [Modes](#modes) below.                              |
| `HCLOUD_CACHE_TTL`  | `duration`          | `10s`   | Lifetime of cached entries. Accepts any Go `time.Duration` string (e.g. `30s`, `2m`). |

## Modes

### `all`

Fetches every Server in the project with a single `GET /servers` call and serves all subsequent `ByID` / `ByName` lookups from the resulting snapshot until the TTL expires. The snapshot is refreshed on the next lookup after expiry. On a cache miss within the TTL (e.g. a freshly created Server), one rate-limited refresh per TTL window is allowed to pick up the new Server; further misses in the same window return without an API call.

### `one`

Caches each Server individually with its own expiration. A `ByID` / `ByName` lookup either returns a non-expired entry or issues a `GET /servers/{id}` (or `GET /servers?name=`) call and stores the result. Expired entries are evicted lazily when other entries are inserted.

### `off`

Disables caching entirely. Every lookup goes directly to the API.
| Default | Description |
| ------------------- | ------------------- | ------- | ------------------------------------------------------------------------------------- |
| `HCLOUD_CACHE_MODE` | `all \| one \| off` | `all` | Selects the caching strategy. See [Modes](#modes) below. |
| `HCLOUD_CACHE_TTL` | `duration`
