# TEP over MeshNet Frame — transport binding v1.0

Binary frame transport for meshnet_app. Offline P2P: BLE (GATT) +
Wi-Fi Direct. Kernel: `spec/TEP.md`.

## Differences vs HTTP

- Transport encryption: ChaCha20-Poly1305 (meshnit's own layer).
- A **compact binary header** replaces JSON for the envelope.
- Canonical signature per the kernel: `tep\n<version>\n<event_id>\n<timestamp>\n<source>\n<type>\n<payload_bytes>`.
  Here `payload_bytes` are the raw payload bytes inside the frame.

## Frame format

We place TEP inside a MeshNet `MeshFrame` (type, hopLimit, ttl, payload).
TEP mesh envelope (binary):

```
Offset  Size      Field
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
...     payload_len   payload (raw bytes, consumer app payload)
...     32        signature (HMAC-SHA256 32 bytes)
```

### varstring

1 byte length (0-255) + UTF-8 bytes.

### timestamp

Unix epoch milliseconds, not IEEE 754 — plain uint64 big-endian (8 bytes).

## Signature

```
tep\n1.0\n<event_id_uuid_string>\n<ISO8601 timestamp>\n<source>\n<type>\n<payload_bytes>
```

- `event_id_uuid_string` — the UUID's canonical text form (lowercase, with
  dashes).
- timestamp ISO8601 — even though the frame stores unix ms, the signature
  uses the ISO string.

The signature is a 32-byte HMAC-SHA256 appended at the end of the frame
(16 lanes). No `v1.` prefix is needed because the frame carries a version
field.

## Kotlin implementation example domains

- `TepMeshEnvelope` — header + payload + signature via frame decode/encode.
- `TepMeshCrypto.sign(frame)` / `.verify(frame)` — HMAC with a shared
  secret derived from the X25519 session key or an app-level `TEP_SECRET`.
- `RoutingEngine` routes the frame: relayed as `MessageType` `TEP_EVENT`
  (41), without violating the kernel's `hopLimit`/`ttl`.

## Event types in MeshNet

| Type | Description |
|---|---|
| `peer.joined` | a new peer joined, topology updated |
| `message.relayed` | a multi-hop message relay occurred |
| `file.transferred` | a chunked file transfer completed |
| `group.updated` | group membership/keys updated |

## Idempotency (mesh)

- Per-node key: `event_id` → `MessageStore` with 24h TTL.
- Duplicate `TEP_EVENT` frames are not relayed again (loop prevention).

## Retry (mesh)

- If BLE drops: the frame is retried over `WifiDirectTransport`.
- `store-and-forward`: frames are stored for offline nodes and delivered on
  reconnection.
- Max 5 attempts; beyond that the status becomes `failed`.