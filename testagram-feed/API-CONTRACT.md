# Testagram Algorithm API contract

The Testagram ranking service is private and stateless. It accepts only bounded candidates selected and authorized by Testagram Backend. It never owns product records, identity, visibility, or access decisions.

## Feed ranking

- Method/path: `POST /v1/feed/rank`
- Authentication: dedicated `Authorization: Bearer <INTERNAL_TOKEN>`, private service network only.
- Required headers: `Content-Type: application/json`, `X-Testagram-API-Version: 1`, and a bounded `X-Request-ID`.
- Request body: at most 1 MiB; exactly one JSON object; unknown fields rejected.
- Candidate count: 1–200; result limit: 1–50.
- Accepted feed modes: `for_you`, `following`, `media`, and `discover`. An omitted mode defaults to `for_you`. `latest` is handled by Backend chronological pagination and is not sent to the ranker. Federated candidates use the same server-selected feed modes.
- Candidate sources: `native`, `activitypub`, or `atproto`. Source identity is provenance only and never establishes account linking or authorization.
- Each candidate must have a non-empty bounded ID and author ID, a valid source, and `visible=true`. Candidate IDs must be unique.
- Ranking features are `following`, `mutualAffinity`, `interestAffinity`, `quality`, `conversation`, `negativeFeedback`, and `reportRisk`. Numeric signals are bounded to [0,1].
- The ranker returns at most the requested number of unique candidate IDs, finite scores in [0,1], and bounded reason codes. It must never return an ID outside the supplied candidate set.

Backend owns candidate retrieval, user/session validation, blocks, mutes, privacy, moderation, visibility, deletion, pagination, canonical hydration, and the final authorization pass after ranking. The ranker must not access Backend PostgreSQL, identity/session stores, object storage, or public client traffic.

## Topic ranking

- Method/path: `POST /v1/topics/rank`
- Same dedicated service bearer, API-version header, request-ID header, content type, body limit, and private-network restriction.
- Backend supplies 1–200 public hashtag candidates and a result limit of 1–50.
- The ranker returns only unique normalized tags from the supplied set with bounded scores and known reason codes. Backend owns candidate discovery and final display policy.

## Operations and failure behavior

- `GET /healthz` reports process liveness; `GET /readyz` reports ranker readiness. These probes do not establish Backend integration or production readiness.
- Calls have bounded deadlines. The Backend client rejects non-200 responses, redirects, duplicate/foreign IDs, invalid scores, and malformed responses.
- Ranking is optional until staging acceptance. On ranker outage, Backend uses its safe chronological fallback; it must not widen access or block authentication and publishing.
- The service bearer is high-entropy, independently rotated, server-side only, and never bundled into Flutter/web or logged.

The canonical product routes and OpenAPI are governed by `Trendyzima/testagram-api`; this repository owns ranking implementation and ranking-specific tests only.