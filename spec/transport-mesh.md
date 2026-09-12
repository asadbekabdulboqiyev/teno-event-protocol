# TEP over MeshNet Frame — transport binding v1.0

meshnet_app uchun binary frame transport. Offline P2P: BLE (GATT) +
Wi-Fi Direct. Kernel: `spec/TEP.md`.

## Farqi vs HTTP

- Transport sifri: ChaCha20-Poly1305 (meshnitning o'z qatlami).
- Envelope uchun JSON o'rniga **compact binary header**.
- Canonical imzo kernel bo'yicha: `tep\n<version>\n<event_id>\n<timestamp>\n<source>\n<type>\n<payload_bytes>`.
  Bunda `payload_bytes` frame ichidagi xom payload baytlari.

## Frame formati

MeshNet `MeshFrame` (type, hopLimit, ttl, payload) ichiga TEP`ni joylashtiramiz.
TEP mesh envelope (binary):

```
Offset  Size      Maydon
0       1         version (0x01)
1       2         header_len (big-endian)
3       4         payload_len (big-endian)
7       16        event_id UUID (16 raw bytes)
23      8         timestamp (unix ms, big-endian)
31      1         flags
                  bit0 = has correlation_id
                  bit1 = has idempotency_key
32      ...       header blob (varstring source, varstring type,
                  optional correlation_id, optional idempotency_key)
...     payload_len   payload (xom baytlar, quidor ilova yuklashi)
...     32        imzo (HMAC-SHA256 32 bayt)
```

### varstring

1 bayt uzunlik (0-255) + UTF-8 baytlar.

### timestamp

Unix epoch millisekund, IEEE 754 emas — oddiy uint64 big-endian (8 bayt).

## Imzo

```
tep\n1.0\n<event_id_uuid_string>\n<ISO8601 timestamp>\n<source>\n<type>\n<payload_bytes>
```

- `event_id_uuid_string` — UUID ning standart matn shakli (lowercase, dash'lar bilan).
- timestamp ISO8601 — frame ichida unix ms bo'lsa ham, imzoda ISO satr.

Imzo 32 bayt HMAC-SHA256 — frame oxirida 32 bayt qilib qo'yiladi (16 lane).
Hech qanday `v1.` prefiksi shart emas, chunki frame ichida version maydoni bor.

## Kotlin implementatsiya namunalari domainlari

- `TepMeshEnvelope` — frame dekod/keycode yordamida header+payload+signature.
- `TepMeshCrypto.sign(frame)` / `.verify(frame)` — X25519 sessiya kalitdan
  olingan shared secret yoki app-level `TEP_SECRET` bilan HMAC.
- `RoutingEngine` frame'ni TESH-tovush: `MessageType` `TEP_EVENT` (41) sifatida
  relay qiladi, `hopLimit`, `ttl` kernelni buzmaydi.

## MeshNet da hodisa turlari

| Type | Izoh |
|---|---|
| `peer.joined` | yangi peer qo'shildi, topology yangilandi |
| `message.relayed` | multi-hop message relay ro'y berdi |
| `file.transferred` | chunked file transfer tugadi |
| `group.updated` | guruh a'zolari/kalit yangilandi |

## Idempotency (mesh)

- Har bir node kalit: `event_id` → `MessageStore`'da 24 soat TTL.
- Takroriy `TEP_EVENT` frame'lar qayta relay qilinmaydi (loop prevention).

## Retry (mesh)

- BLE tushib qolsa: frame `WifiDirectTransport` orqali qayta uriniladi.
- `store-and-forward`: offline node uchun frame saqlanadi, ulanganda
  yuboriladi.
- Max 5 urinish; oshsa `failed` status.