# The Algorithm <-> Backend integration contract

The Algorithm is a private, stateless ranking service. The Backend product services are the only callers: the legacy Go `cmd/api` and the Appwrite-facing `cmd/fresh-api`. Each supplies a bounded batch (1-200) of already-authorized candidates plus ranking features. The Algorithm never reads Appwrite, Neon, object storage, user sessions or browser traffic and cannot decide canonical visibility, blocks, retention or deletion.

`POST /v1/feed/rank` is authenticated with a dedicated high-entropy `INTERNAL_TOKEN` over private Docker networks. `GET /healthz` and `/readyz` are infrastructure probes only. Keep the token server-side, rotate it independently from other service credentials, and never ship it to Flutter/web clients. Backend validates that returned IDs are unique and belong to the submitted candidate set, then reapplies authorization and visibility filters before responding to clients.

## Runtime wiring

- Algorithm container: `HTTP_ADDR=:8092`, `INTERNAL_TOKEN`.
- Legacy Backend `cmd/api`: `ALGORITHM_URL=http://algorithm:8092` and `ALGORITHM_INTERNAL_TOKEN`.
- Appwrite-facing Backend `cmd/fresh-api`: `ALGORITHM_ENDPOINT=http://algorithm:8092` and `ALGORITHM_INTERNAL_TOKEN`.
- The two endpoint variable names are process-specific; both clients call the same `POST /v1/feed/rank` contract and use the same independently generated Algorithm service token.
- In the Fresh Data Centre Compose topology, Algorithm joins the private `ranker` network for Fresh API and the Backend-owned `testagram-platform` network for `cmd/api`. It has no public route or published port.
- Ranking is optional for authentication and defaults to chronological fallback when unavailable. Keep `FEED_RANKING_REQUIRED=false` until live readiness and outage behavior are accepted.

The current rank endpoint and Backend `internal/algorithmclient` implement the service-to-service path. Feed candidate retrieval, cursor pagination and end-to-end Appwrite-to-ranker tests remain Backend work. Keep ranking policy and tests in this repository; do not create a separate API repository.
