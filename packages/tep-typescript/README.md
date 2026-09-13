# tep-typescript

TypeScript implementation of TEP (Teno Event Protocol).
HTTP/REST transport binding for **nescom** and **nesto-codebase**:
`spec/transport-http.md`.

## Contents

- `buildEnvelope` / `parseEnvelope` / `serializeEnvelope` — envelope serde
- `canonical` / `sign` / `verify` — HMAC-SHA256 signature (constant-time)
- `TepClient` — producer: signing + exponential backoff (5 attempts)
- `TepHttpConsumer` — consumer: `pushEndpoint` + `statusEndpoint` handlers
  for Express/Node
- `MemoryIdempotencyStore` — deduplication store with 24h TTL

## Install

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