package pszlbit.drift.controller

import java.util.concurrent.Executors
import java.util.concurrent.ScheduledExecutorService
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test

class ControllerClientTest {

    private lateinit var transport: FakeTransport
    private lateinit var listener: FakeControllerListener
    private lateinit var client: ControllerClient
    private lateinit var executor: ScheduledExecutorService

    @Before
    fun setUp() {
        transport = FakeTransport()
        listener = FakeControllerListener()
        executor = Executors.newSingleThreadScheduledExecutor()
        client =
            ControllerClient(
                transport = transport,
                deviceToken = "device-token-1",
                listener = listener,
                inputIntervalMs = 5L,
                backoffPolicy = ReconnectPolicy(baseDelayMs = 10L, maxDelayMs = 100L),
                executor = executor,
            )
    }

    @After
    fun tearDown() {
        client.close()
        executor.shutdownNow()
    }

    @Test
    fun startSendsHelloWithDeviceTokenAndWaitsForWelcome() {
        client.start()

        val hello = transport.sentEnvelopes.single()
        assertEquals(TYPE_HELLO, hello.type)
        assertNull(hello.seq)
        val helloData = hello.data as HelloData
        assertEquals("device-token-1", helloData.deviceToken)
        assertEquals(ControllerClient.DEFAULT_APP_VERSION, helloData.appVersion)

        transport.replyWelcome(
            Envelope(
                v = PROTOCOL_VERSION,
                type = TYPE_WELCOME,
                seq = null,
                data = WelcomeData(slot = 2, label = "P2", color = "red", phase = PHASE_READY),
            ),
        )

        assertEquals(ConnectionState.CONNECTED, client.state)
        assertEquals(2, client.slot)
        assertEquals(PHASE_READY, client.phase)
    }

    @Test
    fun phasePlayingSendsInputLoopWithIncreasingSequence() {
        client.start()
        transport.replyWelcome(
            Envelope(
                v = PROTOCOL_VERSION,
                type = TYPE_WELCOME,
                seq = null,
                data = WelcomeData(slot = 1, label = "P1", color = "blue", phase = PHASE_PLAYING),
            ),
        )
        client.updateInput(steer = 0.25, throttle = 0.75)

        awaitUntil(timeoutMs = 2000) { transport.sentEnvelopes.count { it.type == TYPE_INPUT } >= 3 }

        val inputs = transport.sentEnvelopes.filter { it.type == TYPE_INPUT }.map { it.data as InputData }
        assertEquals(1, inputs[0].seq)
        assertEquals(0.25, inputs[0].steer, 0.0)
        assertEquals(0.75, inputs[0].throttle, 0.0)
        assertEquals(2, inputs[1].seq)
        assertEquals(3, inputs[2].seq)
    }

    @Test
    fun phaseChangeAwayFromPlayingStopsInputLoop() {
        client.start()
        transport.replyWelcome(
            Envelope(
                v = PROTOCOL_VERSION,
                type = TYPE_WELCOME,
                seq = null,
                data = WelcomeData(slot = 3, label = "P3", color = "green", phase = PHASE_PLAYING),
            ),
        )
        client.updateInput(steer = 0.1, throttle = 0.2)
        awaitUntil(timeoutMs = 2000) { transport.sentEnvelopes.count { it.type == TYPE_INPUT } >= 2 }
        val inputsWhilePlaying = transport.sentEnvelopes.count { it.type == TYPE_INPUT }

        transport.replyPhase(PHASE_ATTRACT)

        Thread.sleep(100)
        val countAfterPhaseChange = transport.sentEnvelopes.count { it.type == TYPE_INPUT }
        Thread.sleep(100)
        val countLater = transport.sentEnvelopes.count { it.type == TYPE_INPUT }
        assertEquals(countAfterPhaseChange, countLater)
    }

    @Test
    fun transportCloseTriggersReconnectWithBackoff() {
        client.start()
        transport.replyWelcome(
            Envelope(
                v = PROTOCOL_VERSION,
                type = TYPE_WELCOME,
                seq = null,
                data = WelcomeData(slot = 4, label = "P4", color = "orange", phase = PHASE_PLAYING),
            ),
        )
        client.updateInput(steer = 0.0, throttle = 1.0)
        awaitUntil(timeoutMs = 2000) { transport.sentEnvelopes.count { it.type == TYPE_INPUT } >= 2 }

        transport.close()

        awaitUntil(timeoutMs = 3000) { transport.sentEnvelopes.count { it.type == TYPE_HELLO } >= 2 }
        transport.replyWelcome(
            Envelope(
                v = PROTOCOL_VERSION,
                type = TYPE_WELCOME,
                seq = null,
                data = WelcomeData(slot = 4, label = "P4", color = "orange", phase = PHASE_PLAYING),
            ),
        )
        awaitUntil(timeoutMs = 3000) { client.state == ConnectionState.CONNECTED }

        val hellos = transport.sentEnvelopes.filter { it.type == TYPE_HELLO }
        assertEquals(2, hellos.size)
        assertEquals("device-token-1", (hellos[1].data as HelloData).deviceToken)
    }

    @Test
    fun errorAuthFailedMarksClientFailed() {
        client.start()
        transport.replyError(
            Envelope(
                v = PROTOCOL_VERSION,
                type = TYPE_ERROR,
                seq = null,
                data = ErrorData(code = "auth_failed", message = "Unknown device token."),
            ),
        )

        assertEquals(ConnectionState.FAILED, client.state)
        assertEquals(listOf("auth_failed"), listener.errors)
    }

    private fun awaitUntil(timeoutMs: Long, condition: () -> Boolean) {
        val deadline = System.currentTimeMillis() + timeoutMs
        while (!condition()) {
            if (System.currentTimeMillis() > deadline) {
                throw AssertionError("Condition was not met within ${timeoutMs}ms.")
            }
            Thread.sleep(5)
        }
    }
}
