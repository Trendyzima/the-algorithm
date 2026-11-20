# Testagram Recommendation and Feed Ranking

This repository ([Trendyzima/the-algorithm](https://github.com/Trendyzima/the-algorithm)) contains the private Testagram ranking service in `testagram-feed/`. The deployable service is built from `testagram-feed/Dockerfile`, listens on the private ranking network, and is called only by Testagram Backend.

## Product ownership

Testagram Backend is authoritative for identity, sessions, candidate retrieval, canonical posts, follows, blocks, mutes, visibility, moderation, pagination, and final authorization. This repository ranks a bounded set of Backend-authorized candidate IDs and returns ordering, scores, and bounded reason codes only. It cannot authorize, hydrate, create, or delete product records.

The broader repository also retains non-deployed research/reference source outside `testagram-feed/`. That material is not a Testagram runtime, is not built into the Testagram ranking container, and must not be imported as a runtime dependency or treated as a product service. The Testagram runtime source of truth is the standalone Go module under `testagram-feed/`.

## Runtime request flow

1. Flutter calls the public Testagram Backend API; it never calls this private ranker.
2. Backend authenticates the viewer and retrieves candidates under current product policy.
3. Backend sends at most 200 authorized candidates to `POST /v1/feed/rank` using a dedicated service bearer, API version, and request ID.
4. The ranker returns unique IDs from that submitted candidate set only.
5. Backend validates the response, rechecks current visibility and authorization, hydrates canonical records, and returns the feed to Flutter.
6. If ranking is unavailable, Backend uses its safe chronological fallback without widening access.

## Seven-repository integration

| Repository | Owns | Boundary with ranking |
|---|---|---|
| [Flutter Client](https://github.com/Trendyzima/testagram-flutter-client-) | Android/web UI and public API transport | Never receives or calls the ranker service credential |
| [Backend](https://github.com/Trendyzima/Testagram-backend-) | Canonical identity, sessions, PostgreSQL records, authorization, candidate retrieval and feed orchestration | Sole ranker caller; rechecks access after ranking |
| [Data Centre](https://github.com/Trendyzima/Testagram-data-centre-) | Private infrastructure, storage lifecycle, processing, backups and recovery | Supplies private runtime networking only |
| [CDN](https://github.com/Trendyzima/cdn-) | Authorized media byte delivery, caching, ranges and streaming | Not a ranking or authorization service |
| [Federated Content](https://github.com/Trendyzima/federated-content) | ActivityPub/WebFinger protocol state and remote delivery | Supplies eligible remote candidates only through Backend-owned authorization |
| **The Algorithm (this repository)** | Private candidate ranking | Returns ordering only; never grants access |
| [Testagram API](https://github.com/Trendyzima/testagram-api) | OpenAPI, private contracts, SDKs, immutable source pins and compatibility CI | Build/release governance only; never a runtime hop |

The canonical ownership map, host routing, service contracts, and locked release gates are maintained in the [Testagram API control plane](https://github.com/Trendyzima/testagram-api).

## Security invariants

- The ranker is reachable only on the private service network.
- The dedicated internal bearer is high entropy, independently rotated, and server-side only.
- User access and refresh tokens are not ranker credentials.
- Candidate provenance does not prove account linking, ownership, or visibility.
- The ranker cannot read product databases, session stores, or object storage.
- Backend rejects duplicate or foreign IDs and rechecks authorization before response.
- No service token, storage key, database credential, or private service URL is shipped in the client.

## Build and test

```sh
cd testagram-feed
go test -race ./...
INTERNAL_TOKEN='replace-with-a-long-random-secret' go run ./cmd/feed-api
```

See [the service contract](testagram-feed/API-CONTRACT.md) and [integration runbook](testagram-feed/INTEGRATION.md). The standalone container listens on port 8092 and must not publish that port to the public Internet.

## Shared integration references

- [Seven-repository ownership and request flow](https://github.com/Trendyzima/testagram-api/blob/main/docs/seven-repository-navigation.md)
- [Locked end-to-end acceptance gates](https://github.com/Trendyzima/testagram-api/blob/main/docs/integration/locked-seven-repository-acceptance.md)
- [Machine-readable service registry](https://github.com/Trendyzima/testagram-api/blob/main/contracts/registry/services.json)
- [Algorithm OpenAPI contract](https://github.com/Trendyzima/testagram-api/blob/main/contracts/upstream/algorithm/openapi.yaml)