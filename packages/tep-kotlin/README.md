# tep-kotlin

TEP (Teno Event Protocol) — Kotlin/JVM implementatsiyasi.
**meshnet_app** (offline P2P mesh, BLE + Wi-Fi Direct) uchun binary frame
transport binding: `spec/transport-mesh.md`.

## Tarkib

- `TepEnvelope` — kernel envelope (data class)
- `Signature` — HMAC-SHA256 imzo + constant-time tekshiruv (JVM `MessageDigest.isEqual`)
- `FrameCodec` — MeshNet binary frame encode/decode/verify

## Frame formati (transport-mesh.md)

```
0      1    version (0x01)
1      2    header_len (big-endian)
3      4    payload_len (big-endian)
7      16   event_id UUID (raw bytes)
23     8    timestamp (unix ms, big-endian)
31     1    flags (bit0 correlation_id, bit1 idempotency_key)
32     ...  header blob (varstring source, type, [correlation_id], [idempotency_key])
...    n    payload (raw bytes)
...    32   HMAC-SHA256 imzo
```

Imzo kernel bo'yicha: `tep\n1.0\n<event_id>\n<ISO timestamp>\n<source>\n<type>\n<payload_bytes>`.

## Ishga tushirish

```bash
./build.sh run        # yangi kotlinc PATH'da bo'lsa
KOTLINC=/path/kotlinc ./build.sh run
```

Yoki Gradle loyihaga `src/main/kotlin` va `src/test/kotlin`ni qo'shing.

## meshnet_app'ga qanday ulanish

`RoutingEngine`'dagi 41 chi MessageType (`TEP_EVENT`) sifatida frame'laringizni
relay qiling. `hopLimit`/`ttl` kernel'ni buzmaydi — MeshFrame o'zida qoladi,
TEP uning payload'iga joylashadi.

```kotlin
val frame = FrameCodec.encode(envelope, secret)
transportManager.broadcast(MessageType.TEP_EVENT, frame)
```