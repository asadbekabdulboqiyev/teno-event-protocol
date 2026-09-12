import { test } from 'node:test';
import assert from 'node:assert/strict';
import {
  buildEnvelope,
  serializeEnvelope,
  parseEnvelope,
  canonical,
  sign,
  verify,
  MemoryIdempotencyStore,
} from '../src/index.js';

const SECRET = '0123456789abcdef0123456789abcdef';

test('envelope serde round-trips without field drift', () => {
  const envelope = buildEnvelope({
    type: 'order.created',
    source: 'nescom',
    correlation_id: 'abc',
    idempotency_key: 'order-1',
    payload: { order_id: '1', total: 250000, ok: true },
  });
  const raw = serializeEnvelope(envelope);
  const parsed = parseEnvelope(raw);
  assert.equal(parsed.event_id, envelope.event_id);
  assert.equal(parsed.type, envelope.type);
  assert.equal(parsed.correlation_id, 'abc');
  assert.equal(parsed.idempotency_key, 'order-1');
  assert.deepEqual(parsed.payload, envelope.payload);
});

test('parseEnvelope rejects bad envelopes', () => {
  assert.throws(() => parseEnvelope(Buffer.from('{"x":1}')));
  const base = buildEnvelope({ type: 'a.b', source: 's', payload: {} });
  assert.throws(() => parseEnvelope(serializeEnvelope({ ...base, version: '9.0' } as unknown as typeof base)));
  assert.throws(() => parseEnvelope(serializeEnvelope({ ...base, payload: 'nope' } as unknown as typeof base)));
});

test('sign/verify accept the exact raw body', () => {
  const envelope = buildEnvelope({ type: 'order.created', source: 'nescom', payload: { x: 1 } });
  const raw = serializeEnvelope(envelope);
  const sig = sign(envelope, raw, SECRET);
  assert.ok(sig.startsWith('v1.'));
  assert.ok(verify(envelope, raw, sig, SECRET));
});

test('verify rejects a different body', () => {
  const env = buildEnvelope({ type: 'a.b', source: 's', payload: { x: 1 } });
  const raw = serializeEnvelope(env);
  const sig = sign(env, raw, SECRET);
  const tampered = Buffer.from(raw.toString().replace('"x":1', '"x":2'));
  assert.ok(!verify(env, tampered, sig, SECRET));
});

test('verify rejects a wrong secret and malformed signature', () => {
  const env = buildEnvelope({ type: 'a.b', source: 's', payload: { x: 1 } });
  const raw = serializeEnvelope(env);
  assert.ok(!verify(env, raw, sign(env, raw, 'ffffffffffffffffffffffffffffffff'), SECRET));
  assert.ok(!verify(env, raw, 'garbage', SECRET));
  assert.ok(!verify(env, raw, '', SECRET));
});

test('canonical is stable for same envelope', () => {
  const a = buildEnvelope({ type: 'a.b', source: 's', payload: { y: [1, 2, 3] } });
  const b = buildEnvelope({
    type: a.type,
    source: a.source,
    payload: a.payload,
    event_id: a.event_id,
  });
  b.timestamp = a.timestamp;
  assert.equal(canonical(a, serializeEnvelope(a)).toString('hex'), canonical(b, serializeEnvelope(b)).toString('hex'));
});

test('MemoryIdempotencyStore evicts expired entries', () => {
  const store = new MemoryIdempotencyStore(-1);
  store.set('k1', { status: 'received' });
  assert.equal(store.get('k1'), undefined);
});

test('MemoryIdempotencyStore returns existing records', () => {
  const store = new MemoryIdempotencyStore(60000);
  store.set('k1', { status: 'delivered', processedAt: new Date().toISOString() });
  const got = store.get('k1');
  assert.equal(got?.status, 'delivered');
});