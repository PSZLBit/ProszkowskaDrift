package pszlbit.drift.controller

import java.util.concurrent.Executors
import java.util.concurrent.ScheduledExecutorService
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test

/**
 * Fake server used by the controller client unit tests.
 *
 * The fake transport mimics a minimal WebSocket server: it accepts the
 * connection, exposes reply helpers for the hello/welcome handshake and phase
 * pushes, and records every sent envelope.
 */
class FakeTransport : Transport {
    var listener: TransportListener? = null
    var shouldAccept: Boolean = true

    val sentEnvelopes: MutableList<Envelope> = mutableListOf()
    var openCount: Int = 0

    override fun open(listener: TransportListener) {
        this.listener = listener
        openCount += 1
        if (shouldAccept) {
            listener.onOpened()
        }
    }

    override fun send(envelope: Envelope) {
        sentEnvelopes.add(envelope)
    }

    override fun close() {
        listener?.onClosed(1000, "fake-close")
    }

    fun replyWelcome(envelope: Envelope) {
        listener?.onEnvelope(envelope)
    }

    fun replyError(envelope: Envelope) {
        listener?.onEnvelope(envelope)
    }

    fun replyPhase(phase: String) {
        listener?.onEnvelope(
            Envelope(
                v = PROTOCOL_VERSION,
                type = TYPE_PHASE,
                seq = null,
                data = PhaseData(phase = phase, timeLeftMs = 0L, roundId = 1),
            ),
        )
    }
}
