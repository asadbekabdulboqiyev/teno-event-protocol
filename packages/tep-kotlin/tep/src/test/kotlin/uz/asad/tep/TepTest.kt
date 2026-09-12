package uz.asad.tep

import java.nio.charset.StandardCharsets

fun main() {
    val secret = "0123456789abcdef0123456789abcdef".toCharArray()
        .joinToString("")
        .toByteArray(StandardCharsets.UTF_8)

    val payload = "{\"relay\":true,\"hops\":2}".toByteArray(StandardCharsets.UTF_8)
    val envelope = TepEnvelope.build(
        type = "message.relayed",
        source = "node-a",
        payload = payload,
        correlationId = "corr-1",
        idempotencyKey = "msg-42",
        eventId = "3f9b4d2e-1c2a-4f5a-9b8c-000000000001",
    )

    val frame = FrameCodec.encode(envelope, secret)
    check(frame.size > 32) { "frame unexpectedly short" }

    val decoded = FrameCodec.decode(frame)
    check(decoded.envelope.eventId == envelope.eventId) { "event_id mismatch" }
    check(decoded.envelope.type == envelope.type) { "type mismatch" }
    check(decoded.envelope.source == envelope.source) { "source mismatch" }
    check(decoded.envelope.correlationId == envelope.correlationId) { "correlation_id mismatch" }
    check(decoded.envelope.idempotencyKey == envelope.idempotencyKey) { "idempotency_key mismatch" }
    check(decoded.envelope.payload.contentEquals(payload)) { "payload mismatch" }

    check(FrameCodec.verify(frame, secret)) { "verify should pass" }

    val tampered = frame.copyOf()
    tampered[tampered.lastIndex - 1] = (tampered[tampered.lastIndex - 1].toInt() xor 0xff).toByte()
    check(!FrameCodec.verify(tampered, secret)) { "tampered frame accepted" }

    val otherSecret = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx".toByteArray(StandardCharsets.UTF_8)
    check(!FrameCodec.verify(frame, otherSecret)) { "wrong secret accepted" }

    try {
        FrameCodec.decode(ByteArray(5))
        error("short frame should have thrown")
    } catch (e: IllegalArgumentException) {
        check(true)
    }

    val sig = Signature.sign(envelope, payload, secret)
    check(sig.startsWith(Signature.SIG_PREFIX)) { "bad signature prefix" }
    check(Signature.verify(envelope, payload, sig, secret)) { "signature verify failed" }
    check(!Signature.verify(envelope, "x".toByteArray(), sig, secret)) { "altered payload accepted" }
    check(!Signature.verify(envelope, payload, sig, otherSecret)) { "wrong secret signature accepted" }

    println("ALL KOTLIN TESTS PASSED")
}