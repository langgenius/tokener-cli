# Module `Tokener.ai Console Management API`

## Source

- Backend: `openapi3`
- Repository: `unknown`
- Pinned tag: ``unknown``
- Files: `v1.yaml`

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

- Summary: Create a Tokener.ai API key (returns the plaintext secret once)
- HTTP: `POST /api/v1/keys`
- Auth: required
- Body: required; media type `application/json`
- Flags: none

### `tokener keys list`

- Summary: List Tokener.ai API keys with 30-day spend
- HTTP: `GET /api/v1/keys`
- Auth: required
- Body: none
- Flags: none
- Output: list path `keys`; columns `name`, `id`, `createdAt`, `lastUsedAt`, `prefix`, `spend30d`; response media `application/json`

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

## usage

### `tokener usage breakdown`

- Summary: Get usage aggregated by model
- HTTP: `GET /api/v1/usage/breakdown`
- Auth: required
- Body: none
- Flags:
  - `--range` (query, default `7`, one of: 7|30): range
  - `--by` (query, default `model`, one of: model): by
- Output: list path `rows`; columns `key`, `requests`, `spendUsdMicro`, `tokens`; response media `application/json`

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
