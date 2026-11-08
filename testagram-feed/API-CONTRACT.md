# Testagram Algorithm API contract

The Algorithm exposes a private ranking API. It is not a public product API and is never called by XClone.

## Endpoint
`POST /v1/feed/rank`

Authentication: `Authorization: Bearer <INTERNAL_TOKEN>`, private service network only.

Required headers:
- `Content-Type: application/json`
- `X-Testagram-API-Version: 1`
- `X-Request-ID: <safe request correlation ID>`

Request: `now` is UTC RFC3339; `candidates` contains 1-200 candidates with unique `id`, `authorId`, `createdAt`, `visible=true`, sensitivity flag and bounded ranking features; `limit` is 1-50.

Response: `{ "items": [{ "id": "...", "score": 0.0, "reasons": [] }] }`. IDs must be unique and drawn exclusively from the submitted candidate set. Scores are finite and normalized to [0,1].

Backend owns identity, candidate discovery, pagination, blocks, mutes, privacy, visibility, deletion, canonical hydration and the final authorization pass after ranking. The ranker must never fetch user/post records from Appwrite, Neon or object storage and must never return arbitrary IDs.

## Operations
- `GET /healthz`: process liveness.
- `GET /readyz`: ranking service readiness.
- Maximum request body is 1 MiB; malformed/unknown JSON fields fail closed.
- Internal token must be high entropy, server-side only and independently rotated.
- Calls have bounded deadlines; no public CORS or browser access.

Canonical product routes and OpenAPI are owned by `Trendyzima/Testagram-backend-`. This repository owns ranking implementation and ranking-specific tests only.
