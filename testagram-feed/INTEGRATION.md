# The Algorithm <-> Backend integration contract

The Algorithm is a private, stateless ranking service. The current Backend `cmd/api` runtime is the supported caller and supplies a bounded batch (1-200) of already-authorized candidates plus ranking features. The historical Appwrite-facing `cmd/fresh-api` runtime is not present in current Backend mainline and is not a supported native-auth caller. The Algorithm never reads Testagram identities, PostgreSQL session state, object storage or browser traffic and cannot decide canonical visibility, blocks, retention or deletion. Testagram user access/refresh tokens are never sent to the ranker.

`POST /v1/feed/rank` is authenticated with a dedicated high-entropy `INTERNAL_TOKEN` over private Docker networks. `GET /healthz` and `/readyz` are infrastructure probes only. Keep the token server-side, rotate it independently from other service credentials, and never ship it to Flutter/web clients. Backend validates that returned IDs are unique and belong to the submitted candidate set, then reapplies authorization and visibility filters before responding to clients.

## Runtime wiring

- Algorithm container: `HTTP_ADDR=:8092`, `INTERNAL_TOKEN`.
- Current Backend `cmd/api`: `ALGORITHM_URL=http://algorithm:8092` and `ALGORITHM_INTERNAL_TOKEN`, matching Algorithm `INTERNAL_TOKEN`.
- The Backend service token is independent from Testagram user access JWTs, Federation credentials and CDN credentials. Do not retain a second caller contract for the removed `cmd/fresh-api` source path.
- In the Fresh Data Centre Compose topology, Algorithm joins the private `ranker` network for Fresh API and the Backend-owned `testagram-platform` network for `cmd/api`. It has no public route or published port.
- Ranking is optional for authentication and defaults to chronological fallback when unavailable. Keep `FEED_RANKING_REQUIRED=false` until live readiness and outage behavior are accepted.

The current rank endpoint and Backend `internal/algorithmclient` implement the service-to-service path. Feed candidate retrieval, cursor pagination and end-to-end native-authenticated Backend-to-ranker tests remain Backend work. Keep ranking policy and tests in this repository; do not create a separate API repository.
