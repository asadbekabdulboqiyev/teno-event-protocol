# TEP over HTTP/REST — transport binding v1.0

nescom (Next.js/TS) va nesto-codebase uchun HTTP/REST transport.

Kernel: `spec/TEP.md`.

## Transport xususiyatlari

- HTTPS (ishlab chiqarishda HSTS bilan).
- `Content-Type: application/json` (UTF-8).
- Imzo `X-TEP-Signature` header'ida.

## Endpointlar

### 1. Hodisa yuborish

```
POST /v1/events/push
Content-Type: application/json
X-TEP-Version: 1.0
X-TEP-Key: <consumer_key>
X-TEP-Signature: v1.<base64url>
```

Body — to'liq envelope JSON (kernel bo'yicha).

Javob:

```json
{ "event_id": "...", "status": "delivered", "error": null }
```

202 qaytadigan holatlar (async queue):
```json
{ "event_id": "...", "status": "accepted", "error": null }
```

### 2. Status so'rovi

```
GET /v1/events/:id/status
```

Javob:
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

## Imzo tekshiruvi (consumer)

1. `X-TEP-Version`, `X-TEP-Key`, `X-TEP-Signature` mavjudligi.
2. Body **raw baytlari** saqlanadi (express: `express.raw({type:'application/json'})`
   yoki Next.js Route Handler `request.text()` → `Buffer`).
3. Envelope JSON-dan parse qilinadi.
4. Canonical string: `tep\n<version>\n<event_id>\n<timestamp>\n<source>\n<type>\n<raw_payload_bytes>`.
   Bunda `raw_payload_bytes` = asl body dagi `payload` qiymatining aniq baytlari.
   JSON parse/re-serialize **qilinmaydi** (key tartibi va son formatiga
   bog'liq farqlar tufayli).

> Amalda, n-til bir xilligi uchun: yuboruvchi `payload`'ni o'zi tuzgan JSON
> satr sifatida signaturega kiritadi. Qabul qiluvchi body'ni raw saqlab,
> payload qiymatini raw body ichidan (offset bilan) oladi yoki shartnomada
> "payload faqat yopiq ob'ekt va u `\n` o'z ichiga olmaydi" deb belgilaydi.
> Eng ishonchli — `body`'ning o'zi `{envelope-fieldlar + payload}` sifatida
> yuboriladi va signature butun body raw baytlari ustida hisoblanadi:

```
canonical = "tep\n" + version + "\n" + event_id + "\n" + timestamp + "\n"
          + source + "\n" + type + "\n" + rawBodyBytes
```

`rawBodyBytes` — butun so'rov body baytlari. Bu yondashuv cross-language
100% mos va soddalikni ta'minlaydi. Bu kernel spec'dagi tavsifning HTTP
uchun konkret talqini.

## Consumer (Express namuna)

```ts
import { TepHttpConsumer } from "tep-typescript";

const consumer = new TepHttpConsumer({
  secret: process.env.TEP_SECRET!,
  handler: async (env) => {
    // biznes logikasi — masalan audit event saqlash
    await db.audit.create({ data: env.payload });
    return { status: 200, code: "OK" };
  },
});

app.use("/v1/events/push", consumer.middleware());
app.get("/v1/events/:id/status", consumer.statusHandler());
```

## Producer (client namuna)

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
- Takroriy urinish aniq shu `event_id` bilan — consumer idempotent.

## Xato kodlari → HTTP status

| Kod | HTTP |
|---|---|
| `OK`/`DUPLICATE_EVENT` | 200 |
| `EVENT_ACCEPTED` | 202 |
| `BAD_REQUEST` | 400 |
| `SIG_INVALID` | 401 |
| `VERSION_UNSUPPORTED` | 415 |
| `EVENT_NOT_FOUND` | 404 |
| `UPSTREAM_ERROR` | 502 |