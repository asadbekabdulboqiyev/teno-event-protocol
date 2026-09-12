# Teno Event Protocol (TEP)

Transportdan mustaqil server-server event protokoli. Bitta kernel, uch xil
transportga mos uch til (TypeScript / Go / Kotlin) implementatsiya.

4 ta pinned loyiha uchun:

- **nescom** (TS, HTTP) — `packages/tep-typescript`
- **nesto-codebase** (TS, HTTP) — `packages/tep-typescript`
- **stress-strike** (Go) — `packages/tep-go`
- **meshnet_app** (Kotlin, binary frame) — `packages/tep-kotlin`

## Struktura

```
spec/                    — protokol spetsifikatsiyasi
  TEP.md                 — kernel (envelope, imzo, idempotency)
  transport-http.md      — HTTP/REST binding
  transport-mesh.md      — MeshNet binary frame binding
packages/
  tep-typescript/        — TS kutubxonasi (nescom, nesto-codebase)
  tep-go/                — Go kutubxonasi (stress-strike sinovi + ish)
  tep-kotlin/            — Kotlin kutubxonasi (meshnet frame)
```

## Imzo (bir qarashda)

```
canonical = "tep\n1.0\n<event_id>\n<timestamp>\n<source>\n<type>\n<payload_bytes>"
signature = v1.<base64url(HMAC-SHA256(secret, canonical))>
```

`payload_bytes` — transport'ga yuborilgan xom paydload baytlari. Bu uch
til o'rtasida byte-level mos imzo beradi.

## Ishga tushirish

```bash
# TS
cd packages/tep-typescript && npm install && npm test

# Go
cd packages/tep-go && go test ./...

# Kotlin (Gradle bo'lsa)
cd packages/tep-kotlin && ./gradlew test
```