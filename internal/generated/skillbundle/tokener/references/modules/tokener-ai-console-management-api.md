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

- Summary: Show the available Tokener.ai credit balance
- HTTP: `GET /api/v1/billing/balance`
- Auth: required
- Body: none
- Flags: none
- Output: columns `availableUsdMicro`, `purchasedAvailableUsdMicro`, `grantedAvailableUsdMicro`, `status`, `updatedAt`; response media `application/json`
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
- Flags: none
- Output: response media `application/json`
- Example:

```
tokener keys create --set-str name=production -o json
tokener keys create \
  --set-str name=production \
  --set limits.maxBudgetUsd=100 \
  --set-str limits.budgetDuration=monthly \
  -o json
```

### `tokener keys list`

- Summary: List API keys and 30-day spend
- HTTP: `GET /api/v1/keys`
- Auth: required
- Body: none
- Flags: none
- Output: list path `keys`; columns `name`, `status`, `prefix`, `spend30d`, `lastUsedAt`, `id`; response media `application/json`
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
- Output: response media `application/json`
- Example: `tokener keys replace-limits <key-id> --file limits.json -o json`

### `tokener keys reveal`

- Summary: Reveal an API key secret
- HTTP: `POST /api/v1/keys/{id}/reveal`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[key-id]` or `--id` (path, required): API key ID
- Output: response media `application/json`
- Example: `tokener keys reveal <key-id> -o json`

### `tokener keys revoke`

- Summary: Permanently revoke an API key
- HTTP: `POST /api/v1/keys/{id}/revoke`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[key-id]` or `--id` (path, required): API key ID
- Output: response media `application/json`
- Example: `tokener keys revoke <key-id> -o json`

### `tokener keys set-status`

- Summary: Enable or disable an API key
- HTTP: `POST /api/v1/keys/{id}/status`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - argument 1 `[key-id]` or `--id` (path, required): API key ID
- Output: response media `application/json`
- Example:

```
tokener keys set-status <key-id> --set-str status=disabled -o json
tokener keys set-status <key-id> --set-str status=active -o json
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

## partners

### `tokener partners allowance`

- Summary: Show a Partner allowance window
- HTTP: `GET /api/partner/v1/allowance-windows/{windowId}`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[window-id]` or `--window-id` (path, required): Partner-owned allowance window ID
  - `--external-ref` (query, required): Partner-owned organization reference
- Output: columns `windowId`, `externalRef`, `status`, `effectiveAmountUsdMicro`, `startsAt`, `endsAt`; response media `application/json`
- Example:

```
tokener partners allowance <window-id> \
  --external-ref customer-123 \
  -o json
```

### `tokener partners balance`

- Summary: Show a Partner organization's credit balance
- HTTP: `GET /api/partner/v1/organizations/by-external-ref/{externalRef}/balance`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[external-ref]` or `--external-ref` (path, required): Partner-owned organization reference
- Output: columns `externalRef`, `availableUsdMicro`, `purchasedAvailableUsdMicro`, `promotionalAvailableUsdMicro`, `allowance.availableUsdMicro`, `generatedAt`; response media `application/json`
- Example:

```
tokener partners balance <external-ref>
tokener partners balance <external-ref> -o json
```

### `tokener partners correct-allowance`

- Summary: Correct a Partner allowance window
- HTTP: `POST /api/partner/v1/allowance-windows/{windowId}/corrections`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - argument 1 `[window-id]` or `--window-id` (path, required): Partner-owned allowance window ID
- Output: response media `application/json`
- Example:

```
tokener partners correct-allowance <window-id> \
  --set-str correctionId=correction-123 \
  --set-str externalRef=customer-123 \
  --set-str deltaUsdMicro=-5000000 \
  -o json
```

### `tokener partners credit-event`

- Summary: Show a Partner credit event
- HTTP: `GET /api/partner/v1/credit-events/{eventId}`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[event-id]` or `--event-id` (path, required): Partner-owned credit event ID
  - `--external-ref` (query, required): Partner-owned organization reference
- Output: columns `eventId`, `externalRef`, `type`, `amountUsdMicro`, `occurredAt`, `relatedEventId`; response media `application/json`
- Example:

```
tokener partners credit-event <event-id> \
  --external-ref customer-123 \
  -o json
```

### `tokener partners models`

- Summary: List models callable by a Partner organization
- HTTP: `GET /api/partner/v1/organizations/by-external-ref/{externalRef}/models`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[external-ref]` or `--external-ref` (path, required): Partner-owned organization reference
- Output: list path `models`; columns `id`, `label`, `provider`, `contextSize`, `pricing.inputPerMillion`, `pricing.outputPerMillion`; response media `application/json`
- Example:

```
tokener partners models <external-ref>
tokener partners models <external-ref> -o json
```

### `tokener partners organization`

- Summary: Show a Partner organization
- HTTP: `GET /api/partner/v1/organizations/by-external-ref/{externalRef}`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[external-ref]` or `--external-ref` (path, required): Partner-owned organization reference
- Output: columns `externalRef`, `status`; response media `application/json`
- Example: `tokener partners organization <external-ref> -o json`

### `tokener partners post-credit-event`

- Summary: Post a Partner credit event
- HTTP: `POST /api/partner/v1/credit-events`
- Auth: required
- Body: required; media type `application/json`
- Flags: none
- Output: response media `application/json`
- Example: `tokener partners post-credit-event --file credit-event.json -o json`

### `tokener partners provision`

- Summary: Provision a Partner organization
- HTTP: `POST /api/partner/v1/organizations`
- Auth: required
- Body: required; media type `application/json`
- Flags: none
- Output: response media `application/json`
- Example:

```
tokener partners provision \
  --set-str externalRef=customer-123 \
  --set-str displayName='Customer 123' \
  -o json
```

### `tokener partners revoke-key`

- Summary: Permanently revoke a Partner organization's API key
- HTTP: `DELETE /api/partner/v1/organizations/by-external-ref/{externalRef}/key`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[external-ref]` or `--external-ref` (path, required): Partner-owned organization reference
- Output: response media `application/json`
- Example: `tokener partners revoke-key <external-ref> -o json`

### `tokener partners rotate-key`

- Summary: Rotate a Partner organization's API key
- HTTP: `POST /api/partner/v1/organizations/by-external-ref/{externalRef}/key`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - argument 1 `[external-ref]` or `--external-ref` (path, required): Partner-owned organization reference
- Output: response media `application/json`
- Example:

```
tokener partners rotate-key <external-ref> \
  --set-str operationId=<uuid> \
  -o json
```

### `tokener partners schedule-allowance`

- Summary: Schedule a Partner allowance window
- HTTP: `POST /api/partner/v1/allowance-windows`
- Auth: required
- Body: required; media type `application/json`
- Flags: none
- Output: response media `application/json`
- Example: `tokener partners schedule-allowance --file allowance.json -o json`

### `tokener partners set-status`

- Summary: Suspend or resume a Partner organization
- HTTP: `PATCH /api/partner/v1/organizations/by-external-ref/{externalRef}`
- Auth: required
- Body: required; media type `application/json`
- Flags:
  - argument 1 `[external-ref]` or `--external-ref` (path, required): Partner-owned organization reference
- Output: response media `application/json`
- Example:

```
tokener partners set-status <external-ref> \
  --set-str desiredStatus=suspended \
  -o json
tokener partners set-status <external-ref> \
  --set-str desiredStatus=active \
  -o json
```

### `tokener partners usage`

- Summary: Show usage for a Partner organization
- HTTP: `GET /api/partner/v1/organizations/by-external-ref/{externalRef}/usage`
- Auth: required
- Body: none
- Flags:
  - argument 1 `[external-ref]` or `--external-ref` (path, required): Partner-owned organization reference
  - `--start-date` (query, required, date): Inclusive UTC start date
  - `--end-date` (query, required, date): Inclusive UTC end date
- Output: list path `models`; columns `model`, `requests`, `inputTokens`, `outputTokens`, `cacheReadInputTokens`, `billedUsdMicro`; response media `application/json`
- Example:

```
tokener partners usage <external-ref> \
  --start-date 2026-08-01 \
  --end-date 2026-08-28 \
  -o json
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
- Example:

```
tokener usage summary
tokener usage summary --range 30 -o json
```
