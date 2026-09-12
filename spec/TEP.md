# Teno Event Protocol (TEP) — Kernel v1.0

Server-server hodisalarni uzatish uchun transportdan mustaqil protokol.
Yadro (kernel) bir xil — transport adapterlari turlicha: HTTP/REST
(nescom, nesto-codebase) va MeshNet binary frame (meshnet_app).

## Qaerdan kelgan

4 ta pinned loyiha uchun yagona event protokoli:

| Loyiha | Stack | Transport |
|---|---|---|
| nescom | Next.js / TS / Prisma | HTTP/REST |
| nesto-codebase | JS web app engine | HTTP/REST |
| stress-strike | Go load-tester | sinov vositasi |
| meshnet_app | Kotlin + Flutter, offline P2P | binary frame (BLE/Wi-Fi Direct) |

## Asosiy tamoyillar

1. **Yagona `event_id`** — takroriy yuborilganda ikkinchi marta qayta
   ishlanmaydi (idempotency). Kalit: `idempotency_key` (bo'lmasa `event_id`).
2. **HMAC-SHA256 imzo** — ishlab chiqaruvchi uchun shared secret. Qabul
   qiluvchi imzoni tekshiradi, teskari soxta hodisalar rad etiladi.
3. **Versiyalash** — `version` maydoni, `1.0`.
4. **Kuzatuvchanlik** — yuborilgan hodisa statusi kuzatiladi
   (`received`, `processing`, `delivered`, `failed`).
5. **Transport-agnostik** — kernel transportga bog'lanmaydi; HTTP va mesh
   bir xil envelope + imzo ishlatadi. Imzo hisoblash uchun faqat `payload`
   ning **raw baytlari** ishlatiladi, bu esa har 3 tilda (TS/Go/Kotlin)
   o'zgarmas canonical hisoblash imkonini beradi.

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

### Maydonlar

| Maydon | Majburiy | Izoh |
|---|---|---|
| `protocol` | ha | `"tep"` |
| `version` | ha | `"1.0"` |
| `event_id` | ha | UUID v4, yagona |
| `type` | ha | `{domain}.{action}` — `order.created`, `job.matched` |
| `source` | ha | yuboruvchi identifikatori |
| `timestamp` | ha | ISO8601 UTC (%Y-%m-%dT%H:%M:%S.%LZ) |
| `correlation_id` | yo'q | so'rovni kuzatish |
| `idempotency_key` | yo'q | deduplikatsiya kaliti (event_id fallback) |
| `payload` | ha | biznes yuki (ob'ekt yoki baytlar) |

## Imzolash

Algoritm: `HMAC-SHA256`, kalit uzunligi ≥ 32 bayt.

Imzo hisoblash uchun **canonical string**:

```
tep\n
<version>\n
<event_id>\n
<timestamp>\n
<source>\n
<type>\n
<raw_payload_bytes>
```

Eslatmalar:
- `raw_payload_bytes` — transport'ga uzatiladigan payloadning **aynan o'sha**
  baytlari. HTTP'da bu JSON paydload baytlari; mesh'da frame ichidagi
  payload baytlari. Bu yondashuv 3 til o'rtasida byte-level moslikni
  kafolatlaydi (JSON key tartibi/flyot xatolari yuzaga kelmaydi).
- Imzo: `v1.<base64url(HMAC-SHA256)>`.
- Solishtirish: constant-time (`timingSafeEqual` / `ConstantTimeCompare` /
  JVM `MessageDigest.isEqual`).

Yuboruvchi va qabul qiluvchi `secret` mahfiy saqlaydi. Secret log'ga
tushmaydi, imzo hech qachon transport header bilan bog'liq emas.

## Idempotency

1. `event_id` birinchi marta kelganda → `received`, ish bajariladi.
2. Xuddi shu id qayta kelsa → **qayta ishlanmaydi**, birinchi natija
   qaytariladi (`DUPLICATE_EVENT`).
3. Idempotency yozuviqa TTL (24 soat tavsiya).

## Statuslar

| Status | Izoh |
|---|---|
| `received` | qabul qilindi, navbatga qo'yildi |
| `processing` | ishlov berilmoqda |
| `delivered` | muvaffaqiyatli ishlov berildi |
| `failed` | xatolik (max retry oshdi) |

## Xato kodlari (transport-agnostik)

| Kod | Izoh |
|---|---|
| `OK` | muvaffaqiyat |
| `EVENT_ACCEPTED` | asinxron qabul qilindi |
| `BAD_REQUEST` | franse (format) noto'g'ri |
| `SIG_INVALID` | imzo yoki kalit noto'g'ri |
| `VERSION_UNSUPPORTED` | versiya qo'llab-quvvatlanmaydi |
| `DUPLICATE_EVENT` | takroriy hodisa — ilgari natija qaytariladi |
| `EVENT_NOT_FOUND` | topilmadi |
| `UPSTREAM_ERROR` | ichki xatolik |

## Event turlari namunalari (4 loyiha bo'yicha)

| Loyiha | Turlar |
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

Ikkala adapter ham ushbu kernel'dagi envelope + canonical imzoni ishlatadi.

## Xavfsizlik

1. Secret faqat muhit o'zgaruvchisida (`TEP_SECRET`).
2. Imzo/kalit/secret log'ga yozilmaydi.
3. Qattiq validation — noto'g'ri envelope rad etiladi.
4. HTTP transport'da HTTPS shart; mesh'da transport sifri (ChaCha20-Poly1305)
   qatlamida himoya qilinadi.