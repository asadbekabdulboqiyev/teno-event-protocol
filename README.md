# Teno Event Protocol (TEP)

A transport-agnostic server-to-server event protocol. One kernel, three
language implementations, three transports.

Used by the 4 pinned projects:

- **nescom** (TS, HTTP) — `packages/tep-typescript`
- **nesto-codebase** (TS, HTTP) — `packages/tep-typescript`
- **stress-strike** (Go) — `packages/tep-go`
- **meshnet_app** (Kotlin, binary frame) — `packages/tep-kotlin`

## Layout

```
spec/                    — protocol specification
  TEP.md                 — kernel (envelope, signature, idempotency)
  transport-http.md      — HTTP/REST binding
  transport-mesh.md      — MeshNet binary frame binding
packages/
  tep-typescript/        — TS library (nescom, nesto-codebase)
  tep-go/                — Go library (stress-strike testing + usage)
  tep-kotlin/            — Kotlin library (meshnet frame)
```

## Signature (at a glance)

```
canonical = "tep\n1.0\n<event_id>\n<timestamp>\n<source>\n<type>\n<payload_bytes>"
signature = v1.<base64url(HMAC-SHA256(secret, canonical))>
```

`payload_bytes` is the raw payload exactly as sent over the transport. This
yields byte-identical signatures across all three languages.

## Getting started

```bash
# TS
cd packages/tep-typescript && npm install && npm test

# Go
cd packages/tep-go && go test ./...

# Kotlin (requires Gradle)
cd packages/tep-kotlin && ./gradlew test
```