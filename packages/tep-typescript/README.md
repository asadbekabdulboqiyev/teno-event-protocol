# tep-typescript

TEP (Teno Event Protocol) — TypeScript implementatsiyasi.
**nescom** va **nesto-codebase** uchun HTTP/REST transport binding:
`spec/transport-http.md`.

## Tarkib

- `buildEnvelope` / `parseEnvelope` / `serializeEnvelope` — envelope serde
- `canonical` / `sign` / `verify` — HMAC-SHA256 imzo (constant-time)
- `TepClient` — producer: imzolash + exponential backoff (5 ta urinish)
- `TepHttpConsumer` — consumer: Express/Node uchun `pushEndpoint` +
  `statusEndpoint` handlerlar
- `MemoryIdempotencyStore` — 24 soat TTL'li deduplikatsiya store

## Izazlash

```bash
npm install
npm test        # build + node --test
npm run typecheck
npm run build
```

## Producer (nescom)

```ts
import { TepClient } from "tep-typescript";

const producer = new TepClient({
  url: new URL("https://consumer.example.com/v1/events/push"),
  secret: process.env.TEP_SECRET!,
  key: process.env.TEP_CONSUMER_KEY!,
  source: "nescom",
});

await producer.push({
  type: "audit.logged",
  payload: { action: "user.login", userId: 42 },
  correlation_id: "req-abc",
});
```

## Consumer (Express)

```ts
import { TepHttpConsumer } from "tep-typescript";
import express from "express";

const consumer = new TepHttpConsumer({
  secret: process.env.TEP_SECRET!,
  handler: async (env) => {
    await db.audit.create({ data: env.payload });
  },
});

const app = express();
app.post("/v1/events/push", (req, res) => consumer.pushEndpoint(req, res));
app.get("/v1/events/:id/status", (req, res) =>
  consumer.statusEndpoint(req, res, req.params.id)
);
```