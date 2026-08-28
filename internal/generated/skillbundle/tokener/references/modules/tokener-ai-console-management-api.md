# Module `Tokener.ai Console Management API`

## Source

- Backend: `openapi3`
- Default hostname: `console.tokener.dev`
- Repository: https://github.com/langgenius/ai-gateway.git
- Pinned tag: `v0.0.2-beta3`
- Files: `apps/console/openapi/v1.yaml`
- Resolved SHA: `780454479bdd398f3ade7f96d07c535b4095722d`

## billing

### `tokener billing balance`

- Summary: Get the organization's available credit balance
- HTTP: `GET /api/v1/billing/balance`
- Auth: required
- Body: none
- Flags: none
- Output: response media `application/json`

### `tokener billing ledger`

- Summary: List append-only ledger entries (paginated)
- HTTP: `GET /api/v1/billing/ledger`
- Auth: required
- Body: none
- Flags:
  - `--cursor` (query): cursor
  - `--limit` (query): limit
- Output: list path `items`; columns `type`, `id`, `amountUsdMicro`, `description`, `occurredAt`, `sourceType`; response media `application/json`; pagination `cursor`

### `tokener billing purchases`

- Summary: List credit purchases (paginated)
- HTTP: `GET /api/v1/billing/purchases`
- Auth: required
- Body: none
- Flags:
  - `--cursor` (query): cursor
  - `--limit` (query): limit
- Output: list path `items`; columns `id`, `amountCents`, `createdAt`, `creditAmountUsdMicro`, `currency`, `invoiceUrl`; response media `application/json`; pagination `cursor`

## identity

### `tokener identity whoami`

- Summary: Resolve the caller identity for the presented management token
- HTTP: `GET /api/v1/whoami`
- Auth: required
- Body: none
- Flags: none
- Output: response media `application/json`

## keys

### `tokener keys create`

- Summary: Create a Tokener.ai API key
- HTTP: `POST /api/v1/keys`
- Auth: required
- Body: required; media type `application/json`
- Flags: none
- Output: response media `application/json`

### `tokener keys list`

- Summary: List Tokener.ai API keys with 30-day spend
- HTTP: `GET /api/v1/keys`
- Auth: required
- Body: none
- Flags: none
- Output: list path `keys`; columns `name`, `id`, `createdAt`, `lastUsedAt`, `prefix`, `spend30d`; response media `application/json`

### `tokener keys reveal`

- Summary: Reveal an existing non-revoked key
- HTTP: `POST /api/v1/keys/{id}/reveal`
- Auth: required
- Body: none
- Flags:
  - `--id` (path, required): id
- Output: response media `application/json`

### `tokener keys revoke`

- Summary: Revoke a key permanently
- HTTP: `POST /api/v1/keys/{id}/revoke`
- Auth: required
- Body: none
- Flags:
  - `--id` (path, required): id
- Output: response media `application/json`

### `tokener keys set-status`

- Summary: Enable or disable a key
- HTTP: `POST /api/v1/keys/{id}/status`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - `--id` (path, required): id
- Output: response media `application/json`

### `tokener keys update-limits`

- Summary: Replace budget, rate, model, and expiry limits for a key (full replacement; omitted/null fields are cleared)
- HTTP: `PATCH /api/v1/keys/{id}/limits`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - `--id` (path, required): id
- Output: response media `application/json`

## models

### `tokener models catalog`

- Summary: List the public managed model catalog
- HTTP: `GET /api/v1/models`
- Auth: required
- Body: none
- Flags: none
- Output: list path `models`; columns `name`, `id`, `cachePer1m`, `category`, `context`, `inputPer1m`; response media `application/json`

### `tokener models public-model-catalog`

- Summary: Get the anonymous versioned machine model catalog
- HTTP: `GET /api/public/v1/models`
- Auth: public
- Body: none
- Flags: none
- Output: list path `models`; response media `application/json`

## partners

### `tokener partners correct-partner-allowance-window`

- Summary: Correct a Partner allowance window
- HTTP: `POST /api/partner/v1/allowance-windows/{windowId}/corrections`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - `--window-id` (path, required): windowId
- Output: response media `application/json`

### `tokener partners create-partner-allowance-window`

- Summary: Schedule an idempotent Partner allowance window
- HTTP: `POST /api/partner/v1/allowance-windows`
- Auth: required
- Body: required; media type `application/json`
- Flags: none
- Output: response media `application/json`

### `tokener partners get-partner-allowance-window`

- Summary: Read an accepted Partner allowance window
- HTTP: `GET /api/partner/v1/allowance-windows/{windowId}`
- Auth: required
- Body: none
- Flags:
  - `--window-id` (path, required): windowId
  - `--external-ref` (query, required): externalRef
- Output: response media `application/json`

### `tokener partners get-partner-credit-event`

- Summary: Read a posted Partner credit event
- HTTP: `GET /api/partner/v1/credit-events/{eventId}`
- Auth: required
- Body: none
- Flags:
  - `--event-id` (path, required): eventId
  - `--external-ref` (query, required): externalRef
- Output: response media `application/json`

### `tokener partners get-partner-organization`

- Summary: Read Partner organization provisioning status
- HTTP: `GET /api/partner/v1/organizations/by-external-ref/{externalRef}`
- Auth: required
- Body: none
- Flags:
  - `--external-ref` (path, required): externalRef
- Output: response media `application/json`

### `tokener partners get-partner-organization-balance`

- Summary: Read a Partner organization's credit balances
- HTTP: `GET /api/partner/v1/organizations/by-external-ref/{externalRef}/balance`
- Auth: required
- Body: none
- Flags:
  - `--external-ref` (path, required): externalRef
- Output: response media `application/json`

### `tokener partners get-partner-organization-models`

- Summary: List models callable by a Partner organization
- HTTP: `GET /api/partner/v1/organizations/by-external-ref/{externalRef}/models`
- Auth: required
- Body: none
- Flags:
  - `--external-ref` (path, required): externalRef
- Output: list path `models`; response media `application/json`

### `tokener partners get-partner-organization-usage`

- Summary: Read model usage for a Partner organization
- HTTP: `GET /api/partner/v1/organizations/by-external-ref/{externalRef}/usage`
- Auth: required
- Body: none
- Flags:
  - `--external-ref` (path, required): externalRef
  - `--start-date` (query, required, date): First UTC date in the usage range.
  - `--end-date` (query, required, date): Last UTC date; cannot be in the future.
- Output: list path `models`; response media `application/json`

### `tokener partners post-partner-credit-event`

- Summary: Post an idempotent Partner credit event
- HTTP: `POST /api/partner/v1/credit-events`
- Auth: required
- Body: required; media type `application/json`
- Flags: none
- Output: response media `application/json`

### `tokener partners provision-partner-organization`

- Summary: Provision a Partner organization and its initial API key
- HTTP: `POST /api/partner/v1/organizations`
- Auth: required
- Body: required; media type `application/json`
- Flags: none
- Output: response media `application/json`

### `tokener partners revoke-partner-organization-key`

- Summary: Revoke a Partner organization's model API key
- HTTP: `DELETE /api/partner/v1/organizations/by-external-ref/{externalRef}/key`
- Auth: required
- Body: none
- Flags:
  - `--external-ref` (path, required): externalRef
- Output: response media `application/json`

### `tokener partners rotate-partner-organization-key`

- Summary: Rotate a Partner organization's model API key
- HTTP: `POST /api/partner/v1/organizations/by-external-ref/{externalRef}/key`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - `--external-ref` (path, required): externalRef
- Output: response media `application/json`

### `tokener partners set-partner-organization-status`

- Summary: Suspend or resume a Partner organization
- HTTP: `PATCH /api/partner/v1/organizations/by-external-ref/{externalRef}`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - `--external-ref` (path, required): externalRef
- Output: response media `application/json`

## usage

### `tokener usage breakdown`

- Summary: Get usage aggregated by model
- HTTP: `GET /api/v1/usage/breakdown`
- Auth: required
- Body: none
- Flags:
  - `--range` (query, default `7`, one of: 7|30): range
- Output: list path `rows`; columns `cacheReadInputTokens`, `inputTokens`, `key`, `outputTokens`, `requests`, `spendUsdMicro`; response media `application/json`

### `tokener usage series`

- Summary: Get per-day usage points with per-model segments
- HTTP: `GET /api/v1/usage/series`
- Auth: required
- Body: none
- Flags:
  - `--range` (query, default `7`, one of: 7|30): range
- Output: list path `points`; columns `day`, `requests`, `spendUsdMicro`, `tokens`; response media `application/json`

### `tokener usage summary`

- Summary: Get usage totals for a time range
- HTTP: `GET /api/v1/usage/summary`
- Auth: required
- Body: none
- Flags:
  - `--range` (query, default `7`, one of: 7|30): range
- Output: response media `application/json`
