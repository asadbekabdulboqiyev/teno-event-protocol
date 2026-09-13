# Teno Event Protocol (TEP) — Kernel v1.0

A transport-agnostic protocol for server-to-server event delivery. The kernel
is identical everywhere; only the transport adapters differ: HTTP/REST
(nescom, nesto-codebase) and the MeshNet binary frame (meshnet_app).

## Origin

A single event protocol for the 4 pinned projects:

| Project | Stack | Transport |
|---|---|---|
| nescom | Next.js / TS / Prisma | HTTP/REST |
| nesto-codebase | JS web app engine | HTTP/REST |
| stress-strike | Go load-tester | test tool |
| meshnet_app | Kotlin + Flutter, offline P2P | binary frame (BLE/Wi-Fi Direct) |

## Core principles

1. **Unique `event_id`** — a duplicate delivery is not processed a second
   time (idempotency). Key: `idempotency_key` (falls back to `event_id`).
2. **HMAC-SHA256 signature** — shared secret between producer and consumer.
   The receiver verifies the signature; forged events are rejected.
3. **Versioning** — `version` field, `1.0`.
4. **Observability** — event delivery status is tracked (`received`,
   `processing`, `delivered`, `failed`).
5. **Transport-agnostic** — the kernel is not coupled to any transport; HTTP
   and mesh use the same envelope and signature. Signing uses only the **raw
   bytes** of the payload, which enables an identical canonical computation in
   all three languages (TS/Go/Kotlin).

## Envelope (kernel)

```json
{
  "protocol": "tep",
  "version": "1.0",
  "event_id": "3f9b4d2e-1c2a-4f5a-9b8c-000000000001",
  "type": "order.created",
  "source": "nescom",
  "timestamp": "2026-09-12T10:00:00.000Z",
  "correlation_id": "req-abc123",
  "idempotency_key": "order-1-created",
  "payload": {}
}
```

### Fields

| Field | Required | Description |
|---|---|---|
| `protocol` | yes | `"tep"` |
| `version` | yes | `"1.0"` |
| `event_id` | yes | UUID v4, unique |
| `type` | yes | `{domain}.{action}` — `order.created`, `job.matched` |
| `source` | yes | sender identifier |
| `timestamp` | yes | ISO8601 UTC (%Y-%m-%dT%H:%M:%S.%LZ) |
| `correlation_id` | no | request tracing |
| `idempotency_key` | no | deduplication key (defaults to `event_id`) |
| `payload` | yes | business payload (object or bytes) |

## Signing

Algorithm: `HMAC-SHA256`, key length ≥ 32 bytes.

The **canonical string** used to compute the signature:

```
tep\n
<version>\n
<event_id>\n
<timestamp>\n
<source>\n
<type>\n
<raw_payload_bytes>
```

Notes:
- `raw_payload_bytes` are **exactly** the bytes of the payload transmitted
  over the transport. Over HTTP these are the JSON payload bytes; over mesh
  they are the payload bytes inside the frame. This approach guarantees
  byte-level compatibility across all three languages (no JSON key-order or
  float-formatting mistakes).
- Signature: `v1.<base64url(HMAC-SHA256)>`.
- Comparison: constant-time (`timingSafeEqual` / `ConstantTimeCompare` / JVM
  `MessageDigest.isEqual`).

The `secret` is kept private by both producer and consumer. It is never
logged and never derived from transport headers.

## Idempotency

1. On the first sight of an `event_id` → `received`, the work is executed.
2. If the same id arrives again → **not processed again**, the first result
   is returned (`DUPLICATE_EVENT`).
3. Idempotency records have a TTL (24 hours recommended).

## Statuses

| Status | Description |
|---|---|
| `received` | accepted and queued |
| `processing` | being processed |
| `delivered` | processed successfully |
| `failed` | error (max retries exceeded) |

## Error codes (transport-agnostic)

| Code | Description |
|---|---|
| `OK` | success |
| `EVENT_ACCEPTED` | accepted asynchronously |
| `BAD_REQUEST` | frame (format) invalid |
| `SIG_INVALID` | signature or key invalid |
| `VERSION_UNSUPPORTED` | version not supported |
| `DUPLICATE_EVENT` | duplicate event — returns the previous result |
| `EVENT_NOT_FOUND` | not found |
| `UPSTREAM_ERROR` | internal error |

## Sample event types (per project)

| Project | Types |
|---|---|
| nescom | `company.updated`, `employee.invited`, `audit.logged` |
| nesto-codebase | `theme.published`, `preview.generated` |
| stress-strike | `loadtest.finished`, `threshold.breached` |
| meshnet | `peer.joined`, `message.relayed`, `file.transferred` |

## Transports

- **`tep-http`** — HTTP/REST binding (nescom, nesto-codebase) →
  `transport-http.md`
- **`tep-mesh`** — MeshNet binary frame binding (meshnet_app) →
  `transport-mesh.md`

Both adapters use this kernel's envelope and canonical signature.

## Security

1. The secret lives only in an environment variable (`TEP_SECRET`).
2. Signatures, keys, and secrets are never logged.
3. Strict validation — malformed envelopes are rejected.
4. HTTPS is required on HTTP transports; on mesh, protection is provided by
   the transport encryption layer (ChaCha20-Poly1305).