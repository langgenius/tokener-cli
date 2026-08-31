# Module `Tokener.ai Console Management API`

## Source

- Backend: `openapi3`
- Default hostname: `console.tokener.dev`
- Repository: https://github.com/langgenius/ai-gateway.git
- Pinned tag: `101069be5d59fc9d03c0a54f24e7a42f640a4660`
- Files: `apps/console/openapi/v1.yaml`
- Resolved SHA: `101069be5d59fc9d03c0a54f24e7a42f640a4660`

## billing

### `tokener billing balance`

- Summary: Show the available Tokener.ai credit balance
- HTTP: `GET /api/v1/billing/balance`
- Auth: required
- Body: none
- Flags: none
- Output: columns `availableUsdMicro`, `purchasedAvailableUsdMicro`, `grantedAvailableUsdMicro`, `status`, `updatedAt`; response media `application/json`
- Search terms: `credit`, `credits`
- Example:

```
tokener billing balance
tokener billing balance -o json
```

### `tokener billing ledger`

- Summary: List credit ledger activity
- HTTP: `GET /api/v1/billing/ledger`
- Auth: required
- Body: none
- Flags:
  - `--cursor` (query): Opaque cursor returned by the previous page
  - `--limit` (query): Number of ledger entries per page
- Output: list path `items`; columns `occurredAt`, `amountUsdMicro`, `type`, `sourceType`, `description`, `id`; response media `application/json`; pagination `cursor`
- Search terms: `credit`, `credits`
- Example:

```
tokener billing ledger --limit 25
tokener billing ledger --all -o json
```

### `tokener billing purchases`

- Summary: List credit purchases
- HTTP: `GET /api/v1/billing/purchases`
- Auth: required
- Body: none
- Flags:
  - `--cursor` (query): Opaque cursor returned by the previous page
  - `--limit` (query): Number of purchases per page
- Output: list path `items`; columns `createdAt`, `status`, `amountCents`, `creditAmountUsdMicro`, `currency`, `id`; response media `application/json`; pagination `cursor`
- Example:

```
tokener billing purchases --limit 25
tokener billing purchases --all -o json
```

## identity

### `tokener identity whoami`

- Summary: Show the authenticated user and organization
- HTTP: `GET /api/v1/whoami`
- Auth: required
- Body: none
- Shortcuts:
  - `tokener whoami`
- Flags: none
- Output: columns `email`, `emailVerified`, `tokenName`, `organizationId`, `userId`; response media `application/json`
- Example:

```
tokener identity whoami
tokener identity whoami -o json
```

## keys

### `tokener keys create`

- Summary: Create an API key
- HTTP: `POST /api/v1/keys`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - `--name` (body): name
- Output: columns `record.name`, `record.prefix`, `record.status`, `record.id`, `record.createdAt`; response media `application/json`
- Example:

```
tokener keys create --name production -o json
tokener keys create \
  --name production \
  --set-str limits.maxBudgetUsd=100 \
  --set-str limits.budgetDuration=monthly \
  -o json
```

### `tokener keys list`

- Summary: List API keys and 30-day spend
- HTTP: `GET /api/v1/keys`
- Auth: required
- Body: none
- Flags: none
- Output: list path `keys`; columns `name`, `status`, `prefix`, `spend30d`, `limits.maxBudgetUsdMicro`, `lastUsedAt`, `id`; response media `application/json`
- Example:

```
tokener keys list
tokener keys list -o json
```

### `tokener keys replace-limits`

- Summary: Replace all limits for an API key
- HTTP: `PATCH /api/v1/keys/{id}/limits`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - argument 1 `[key-id]` or `--id` (path, required): API key ID
  - `--allowed-models` (body): allowedModels
  - `--budget-duration` (body, one of: daily|weekly|monthly): budgetDuration
  - `--expires-at` (body, date-time): expiresAt
  - `--max-budget-usd` (body): USD decimal string from 0 to 2000000000 with at most 6 fractional digits.
  - `--max-parallel-requests` (body): maxParallelRequests
  - `--rpm-limit` (body): rpmLimit
  - `--tpm-limit` (body): tpmLimit
- Output: response media `application/json`
- Notes:
  - This is a full replacement: omitted fields are cleared.
- Example:

```
tokener keys replace-limits <key-id> --file limits.json -o json
tokener keys replace-limits <key-id> \
  --max-budget-usd 100 \
  --budget-duration monthly \
  --rpm-limit 60 \
  --tpm-limit 100000 \
  --max-parallel-requests 4 \
  --expires-at 2027-01-01T00:00:00Z \
  --allowed-models gpt-5 \
  -o json
```

### `tokener keys reveal`

- Summary: Reveal an API key secret
- HTTP: `POST /api/v1/keys/{id}/reveal`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[key-id]` or `--id` (path, required): API key ID
- Output: response media `application/json`
- Notes:
  - The output is a credential. Do not write it to logs or shell history.
- Search terms: `secret`
- Known errors:
  - HTTP 400: The key is already revoked.
- Example: `tokener keys reveal <key-id> -o json`

### `tokener keys revoke`

- Summary: Permanently revoke an API key
- HTTP: `POST /api/v1/keys/{id}/revoke`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[key-id]` or `--id` (path, required): API key ID
- Output: response media `application/json`
- Known errors:
  - HTTP 400: The key is already revoked.
- Example: `tokener keys revoke <key-id> -o json`

### `tokener keys set-status`

- Summary: Enable or disable an API key
- HTTP: `POST /api/v1/keys/{id}/status`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - argument 1 `[key-id]` or `--id` (path, required): API key ID
  - `--status` (body, required, one of: active|disabled): status
- Output: response media `application/json`
- Known errors:
  - HTTP 400: The key is already revoked.
- Example:

```
tokener keys set-status <key-id> --status disabled -o json
tokener keys set-status <key-id> --status active -o json
```

## models

### `tokener models list`

- Summary: List models available to the organization
- HTTP: `GET /api/v1/models`
- Auth: required
- Body: none
- Flags: none
- Output: list path `models`; columns `id`, `name`, `provider`, `context`, `inputPer1m`, `outputPer1m`; response media `application/json`
- Example:

```
tokener models list
tokener models list -o json
```

## usage

### `tokener usage by-model`

- Summary: Show usage grouped by model
- HTTP: `GET /api/v1/usage/breakdown`
- Auth: required
- Body: none
- Flags:
  - `--range` (query, default `7`, one of: 7|30): Usage window in days
- Output: list path `rows`; columns `key`, `requests`, `inputTokens`, `outputTokens`, `cacheReadInputTokens`, `spendUsdMicro`; response media `application/json`
- Search terms: `spend`, `cost`
- Example:

```
tokener usage by-model
tokener usage by-model --range 30 -o json
```

### `tokener usage daily`

- Summary: Show daily usage
- HTTP: `GET /api/v1/usage/series`
- Auth: required
- Body: none
- Flags:
  - `--range` (query, default `7`, one of: 7|30): Usage window in days
- Output: list path `points`; columns `day`, `requests`, `tokens`, `spendUsdMicro`; response media `application/json`
- Search terms: `spend`, `cost`
- Example:

```
tokener usage daily
tokener usage daily --range 30 -o json
```

### `tokener usage summary`

- Summary: Show usage totals
- HTTP: `GET /api/v1/usage/summary`
- Auth: required
- Body: none
- Flags:
  - `--range` (query, default `7`, one of: 7|30): Usage window in days
- Output: columns `summary.requests`, `summary.tokens`, `summary.spendUsdMicro`, `summary.inputTokens`, `summary.outputTokens`, `summary.cacheReadInputTokens`; response media `application/json`
- Search terms: `spend`, `cost`
- Example:

```
tokener usage summary
tokener usage summary --range 30 -o json
```
