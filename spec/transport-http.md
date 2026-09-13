# TEP over HTTP/REST — transport binding v1.0

HTTP/REST transport for nescom (Next.js/TS) and nesto-codebase.

Kernel: `spec/TEP.md`.

## Transport characteristics

- HTTPS (with HSTS in production).
- `Content-Type: application/json` (UTF-8).
- Signature in the `X-TEP-Signature` header.

## Endpoints

### 1. Push an event

```
POST /v1/events/push
Content-Type: application/json
X-TEP-Version: 1.0
X-TEP-Key: <consumer_key>
X-TEP-Signature: v1.<base64url>
```

Body — the full envelope JSON (per the kernel).

Response:

```json
{ "event_id": "...", "status": "delivered", "error": null }
```

For async queueing (202):

```json
{ "event_id": "...", "status": "accepted", "error": null }
```

### 2. Status query

```
GET /v1/events/:id/status
```

Response:
```json
{ "event_id": "...", "status": "delivered", "processed_at": "...", "error": null }
```

### 3. Health check

```
GET /v1/health
```

```json
{ "status": "ok", "version": "1.0", "time": "<ISO8601>" }
```

## Signature verification (consumer)

1. `X-TEP-Version`, `X-TEP-Key`, `X-TEP-Signature` must be present.
2. The body is kept as **raw bytes** (`express.raw({type:'application/json'})`
   or Next.js Route Handler `request.text()` → `Buffer`).
3. The envelope is parsed from the JSON.
4. Canonical string: `tep\n<version>\n<event_id>\n<timestamp>\n<source>\n<type>\n<raw_payload_bytes>`.
   Here `raw_payload_bytes` are the exact bytes of the `payload` value in the
   original body. The JSON is **not** parsed and re-serialized (to avoid
   differences from key order and number formatting).

> In practice, for cross-language equivalence: the producer signs the
> `payload` it built itself as a JSON string. The consumer keeps the body
> raw and extracts the payload value from within the raw body (by offset),
> or the contract states "the payload is a closed object and must not
> contain `\n`". The most reliable approach is to send the `body` itself as
> `{envelope-fields + payload}` and compute the signature over the raw bytes
> of the whole body:

```
canonical = "tep\n" + version + "\n" + event_id + "\n" + timestamp + "\n"
          + source + "\n" + type + "\n" + rawBodyBytes
```

`rawBodyBytes` are the bytes of the entire request body. This approach is
100% cross-language consistent and simple. It is the concrete HTTP
interpretation of the kernel spec.

## Consumer (Express example)

```ts
import { TepHttpConsumer } from "tep-typescript";

const consumer = new TepHttpConsumer({
  secret: process.env.TEP_SECRET!,
  handler: async (env) => {
    // business logic — e.g. store an audit event
    await db.audit.create({ data: env.payload });
    return { status: 200, code: "OK" };
  },
});

app.use("/v1/events/push", consumer.middleware());
app.get("/v1/events/:id/status", consumer.statusHandler());
```

## Producer (client example)

```ts
import { TepClient } from "tep-typescript";

const producer = new TepClient({
  url: new URL("https://api.nescom.uz/v1/events/push"),
  secret: process.env.TEP_SECRET!,
  key: process.env.TEP_CONSUMER_KEY!,
  source: "nescom",
});

await producer.push({
  type: "audit.logged",
  payload: { action: "user.login", userId: 42 },
});
```

## Retry (producer)

- Exponential backoff: `delay = MIN(base * 2^n, max)`, `base=1s`, `max=60s`.
- `maxRetries = 5`.
- Retries reuse the exact same `event_id` — the consumer is idempotent.

## Error codes → HTTP status

| Code | HTTP |
|---|---|
| `OK`/`DUPLICATE_EVENT` | 200 |
| `EVENT_ACCEPTED` | 202 |
| `BAD_REQUEST` | 400 |
| `SIG_INVALID` | 401 |
| `VERSION_UNSUPPORTED` | 415 |
| `EVENT_NOT_FOUND` | 404 |
| `UPSTREAM_ERROR` | 502 |