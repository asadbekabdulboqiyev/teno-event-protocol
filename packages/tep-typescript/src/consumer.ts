import type { IncomingMessage, ServerResponse } from 'node:http';
import { parseEnvelope, serializeEnvelope } from './envelope.js';
import { verify } from './signature.js';
import { MemoryIdempotencyStore, type IdempotencyRecord, type IdempotencyStore } from './idempotency.js';
import { HTTP_STATUS } from './errors.js';
import type { TepEnvelope, TepResult } from './types.js';

export type EventHandler = (envelope: TepEnvelope) => Promise<Partial<TepResult> | void>;

export interface TepConsumerOptions {
  secret: string;
  handler: EventHandler;
  store?: IdempotencyStore;
}

async function readBody(req: IncomingMessage): Promise<Buffer> {
  const chunks: Buffer[] = [];
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return Buffer.concat(chunks);
}

function sendJson(res: ServerResponse, status: number, body: unknown): void {
  res.statusCode = status;
  res.setHeader('content-type', 'application/json');
  res.end(JSON.stringify(body));
}

function statusToResult(record: IdempotencyRecord, eventId: string): TepResult {
  return {
    code: record.status === 'failed' ? 'UPSTREAM_ERROR' : 'DUPLICATE_EVENT',
    event_id: eventId,
    status: record.status,
    error: record.error,
    httpStatus: record.status === 'failed' ? 502 : 200,
  };
}

export class TepHttpConsumer {
  private readonly secret: string;
  private readonly handler: EventHandler;
  private readonly store: IdempotencyStore;

  constructor(options: TepConsumerOptions) {
    if (options.secret.length < 32) throw new Error('TEP secret must be at least 32 bytes');
    this.secret = options.secret;
    this.handler = options.handler;
    this.store = options.store ?? new MemoryIdempotencyStore();
  }

  async pushEndpoint(req: IncomingMessage, res: ServerResponse): Promise<void> {
    const version = req.headers['x-tep-version'];
    const key = req.headers['x-tep-key'];
    const signature = req.headers['x-tep-signature'];

    if (!signature || typeof signature !== 'string') {
      return sendJson(res, HTTP_STATUS.SIG_INVALID, { code: 'SIG_INVALID', error: 'missing signature header' });
    }

    let rawBody: Buffer;
    try {
      rawBody = await readBody(req);
      if (rawBody.length === 0) throw new Error('empty body');
    } catch (err) {
      return sendJson(res, HTTP_STATUS.BAD_REQUEST, { code: 'BAD_REQUEST', error: String(err) });
    }

    let envelope: TepEnvelope;
    try {
      envelope = parseEnvelope(rawBody);
    } catch (err) {
      return sendJson(res, HTTP_STATUS.BAD_REQUEST, { code: 'BAD_REQUEST', error: String(err) });
    }

    if (!version || typeof version !== 'string' || version !== envelope.version) {
      return sendJson(res, HTTP_STATUS.VERSION_UNSUPPORTED, {
        code: 'VERSION_UNSUPPORTED',
        error: `expected version ${envelope.version}`,
      });
    }
    if (!key || typeof key !== 'string' || key.length === 0) {
      return sendJson(res, HTTP_STATUS.SIG_INVALID, { code: 'SIG_INVALID', error: 'missing key header' });
    }

    const signatureOk = verify(envelope, rawBody, signature, this.secret);
    if (!signatureOk) {
      return sendJson(res, HTTP_STATUS.SIG_INVALID, { code: 'SIG_INVALID', error: 'signature verification failed' });
    }

    const dedupeKey = envelope.idempotency_key ?? envelope.event_id;
    const existing = this.store.get(dedupeKey);
    if (existing) {
      return sendJson(res, existing.status === 'failed' ? 502 : 200, statusToResult(existing, envelope.event_id));
    }

    this.store.set(dedupeKey, { status: 'received' });

    try {
      const result = await this.handler(envelope);
      const record: IdempotencyRecord = {
        status: result?.code === 'EVENT_ACCEPTED' ? 'processing' : 'delivered',
        processedAt: new Date().toISOString(),
        error: result?.error ?? undefined,
      };
      this.store.set(dedupeKey, record);
      const code = result?.code ?? 'OK';
      const status = HTTP_STATUS[code] ?? 200;
      return sendJson(res, status, {
        code,
        event_id: envelope.event_id,
        status: record.status,
        error: record.error,
      });
    } catch (err) {
      const record: IdempotencyRecord = { status: 'failed', error: String(err) };
      this.store.set(dedupeKey, record);
      return sendJson(res, HTTP_STATUS.UPSTREAM_ERROR, statusToResult(record, envelope.event_id));
    }
  }

  async statusEndpoint(req: IncomingMessage, res: ServerResponse, eventId: string): Promise<void> {
    const record = this.store.get(eventId);
    if (!record) {
      return sendJson(res, HTTP_STATUS.EVENT_NOT_FOUND, { code: 'EVENT_NOT_FOUND', event_id: eventId, error: null });
    }
    return sendJson(res, 200, {
      code: record.status === 'failed' ? 'UPSTREAM_ERROR' : 'OK',
      event_id: eventId,
      status: record.status,
      processed_at: record.processedAt,
      error: record.error,
    });
  }
}

export { serializeEnvelope };