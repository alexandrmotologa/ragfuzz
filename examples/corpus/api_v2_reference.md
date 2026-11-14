# API v2 Reference Specification

The API v2 architecture exposes RESTful endpoints for programmatic workspace management, data ingestion, and webhook subscriptions.

## Authentication and API Keys
All requests must include an HTTP Authorization header containing a Bearer API token: `Authorization: Bearer <api_token>`.
Tokens without the `Bearer ` prefix return HTTP 401 Unauthorized. API keys do not expire automatically unless configured with a custom TTL during generation.

## Webhook Delivery and Retries
Webhooks are dispatched via HTTPS POST requests signed with an HMAC-SHA256 signature passed in the `X-Signature-SHA256` header.
Failed deliveries (HTTP status >= 500 or timeout exceeding 5000ms) are retried using exponential backoff up to 5 attempts over 24 hours.

## Deprecated Endpoints and Version Collisions
The legacy endpoint `/api/v1/workspace/export` is deprecated as of December 2024.
Clients must migrate to `/api/v2/async-exports` which returns a job ID rather than synchronous binary payloads.
The parameter `include_metadata` was renamed to `with_provenance` in API v2.0.
