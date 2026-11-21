# Testagram Algorithm integration boundary

The Testagram Backend is the sole identity, session, authorization and canonical product-data authority. It validates native user sessions against PostgreSQL before constructing a bounded ranking request. Algorithm is a private ranking service called only by the Go Backend.

Algorithm receives only Backend-authorized candidate batches. It never receives user credentials, provider credentials, raw identity evidence, or unrestricted product records. It does not query identity or product databases, accept browser requests, or make authorization decisions.

Backend applies blocks, visibility, moderation, privacy and follow rules before ranking, then rechecks current authorization before returning a feed. Algorithm may reorder only IDs from the submitted candidate set. No external identity-provider or legacy data-provider fallback is permitted. Federation remains governed by its own private service contract.
