# tep-kotlin

Kotlin/JVM implementation of TEP (Teno Event Protocol).
Binary frame transport binding for **meshnet_app** (offline P2P mesh,
BLE + Wi-Fi Direct): `spec/transport-mesh.md`.

## Contents

- `TepEnvelope` — kernel envelope (data class)
- `Signature` — HMAC-SHA256 signature + constant-time check (JVM
  `MessageDigest.isEqual`)
- `FrameCodec` — MeshNet binary frame encode/decode/verify

## Frame format (transport-mesh.md)

```
0      1    version (0x01)
1      2    header_len (big-endian)
3      4    payload_len (big-endian)
7      16   event_id UUID (raw bytes)
23     8    timestamp (unix ms, big-endian)
31     1    flags (bit0 correlation_id, bit1 idempotency_key)
32     ...  header blob (varstring source, type, [correlation_id], [idempotency_key])
...    n    payload (raw bytes)
...    32   HMAC-SHA256 signature
```

Signature per the kernel: `tep\n1.0\n<event_id>\n<ISO timestamp>\n<source>\n<type>\n<payload_bytes>`.

## Run

```bash
./build.sh run        # if kotlinc is on PATH
KOTLINC=/path/kotlinc ./build.sh run
```

Or add `src/main/kotlin` and `src/test/kotlin` to a Gradle project.

## How to hook it into meshnet_app

Relay your frames as MessageType 41 (`TEP_EVENT`) in `RoutingEngine`.
`hopLimit`/`ttl` do not break the kernel — the TEP envelope lives inside the
MeshFrame payload.

```kotlin
val frame = FrameCodec.encode(envelope, secret)
transportManager.broadcast(MessageType.TEP_EVENT, frame)
```