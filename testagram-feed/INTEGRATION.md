# The Algorithm <-> Backend integration contract

The Algorithm is a private, stateless ranking service. The current Testagram Backend `cmd/api` service is the only supported caller. Each supplies a bounded batch (1-200) of already-authorized candidates plus ranking features. The Algorithm never reads product databases, object storage, user sessions or browser traffic and cannot decide canonical visibility, blocks, retention or deletion.

`POST /v1/feed/rank` is authenticated with a dedicated high-entropy `INTERNAL_TOKEN` over private Docker networks. `GET /healthz` and `/readyz` are infrastructure probes only. Keep the token server-side, rotate it independently from other service credentials, and never ship it to Flutter/web clients. Backend validates that returned IDs are unique and belong to the submitted candidate set, then reapplies authorization and visibility filters before responding to clients.

## Runtime wiring

- Algorithm container: `HTTP_ADDR=:8092`, `INTERNAL_TOKEN`.
- Backend `cmd/api`: `ALGORITHM_URL=http://algorithm:8092` and `ALGORITHM_INTERNAL_TOKEN`.
- The ranker joins only the Backend-owned private `testagram-platform` network. It has no public route or published port.
- The retired Fresh API is not a supported caller and must not be reintroduced as a compatibility fallback.
- Ranking is optional for authentication and defaults to chronological fallback when unavailable. Keep `FEED_RANKING_REQUIRED=false` until live readiness and outage behavior are accepted.

The current rank endpoint and Backend `internal/algorithmclient` implement the service-to-service path. Feed candidate retrieval, cursor pagination and end-to-end staging acceptance remain Backend responsibilities. The source-level client/OpenAPI contract is not proof of a deployed private service call. Keep ranking policy and tests in this repository; do not create a separate API repository.
