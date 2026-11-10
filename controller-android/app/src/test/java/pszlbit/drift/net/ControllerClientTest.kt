package pszlbit.drift.net

import java.util.concurrent.TimeUnit
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Before
import org.junit.Test
import pszlbit.drift.input.InputState
import pszlbit.drift.protocol.ServerMessage

/**
 * Fake-server tests for the controller client.
 *
 * MockWebServer owns a real local socket and plays the Go server side: it
 * accepts `/ws/controller`, receives `hello` and `input`, and can push
 * `welcome`, `phase`, `pong`, and `error` frames. Timer-driven behavior uses
 * [ManualScheduler] and its fake monotonic clock, so input cadence and the pong
 * watchdog do not depend on wall-clock sleeps. No venue network or tablet is
 * needed.
 */
class ControllerClientTest {
    private lateinit var server: MockWebServer
    private lateinit var listener: RecordingControllerListener
    private lateinit var scheduler: ManualScheduler
    private var client: ControllerClient? = null

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        listener = RecordingControllerListener()
        scheduler = ManualScheduler()
    }

    @After
    fun tearDown() {
        client?.stop()
        server.shutdown()
    }

    @Test
    fun connectsToControllerEndpointAndSendsHelloWithDeviceToken() {
        val fake = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(fake))

        startClient()

        val request = server.takeRequest(5, TimeUnit.SECONDS)
        assertNotNull("Client did not open a WebSocket.", request)
        assertEquals("/ws/controller", request!!.path)

        fake.awaitOpen()
        val hello = fake.awaitMessage("hello")
        assertEquals(1, hello.getInt("v"))
        assertEquals("device-token-1", hello.getJSONObject("data").getString("deviceToken"))
        assertEquals("0.1.0-test", hello.getJSONObject("data").getString("appVersion"))

        fake.sendWelcome(slot = 2, phase = "ready")
        awaitState(ConnectionState.CONNECTED)
        assertEquals(2, listener.welcome?.slot)
        assertEquals("P2", listener.welcome?.label)
    }

    @Test
    fun sendsInputOnlyWhilePlaying() {
        val fake = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(fake))

        startClient()
        fake.awaitOpen()
        fake.awaitMessage("hello")
        fake.sendWelcome(slot = 1, phase = "ready")
        awaitState(ConnectionState.CONNECTED)

        scheduler.advanceBy(1_200L)
        assertEquals(0L, listener.lastStats?.sentTotal)

        fake.sendPhase("playing")
        awaitServerMessage({ it is ServerMessage.Phase && it.phase == "playing" }, "playing phase")
        scheduler.advanceBy(1_000L)
        val sentWhilePlaying = listener.lastStats?.sentTotal ?: 0L
        assertTrue("Expected input while playing.", sentWhilePlaying > 0L)

        val inputs = fake.awaitMessages("input", count = 2, timeoutMs = 3_000L)
        val first = inputs[0]
        assertEquals(1.0, first.getJSONObject("data").getDouble("steer"), 1e-9)
        assertEquals(-1.0, first.getJSONObject("data").getDouble("throttle"), 1e-9)
        assertTrue("Sequence numbers must increase.", first.getLong("seq") < inputs[1].getLong("seq"))

        fake.sendPhase("paused")
        awaitServerMessage({ it is ServerMessage.Phase && it.phase == "paused" }, "paused phase")
        fake.drainMessages("input")
        scheduler.advanceBy(1_000L)
        val sentAfterPause = listener.lastStats?.sentTotal ?: 0L
        scheduler.advanceBy(1_000L)
        assertEquals(sentAfterPause, listener.lastStats?.sentTotal)
        assertEquals(0, fake.drainMessages("input").size)
    }

    @Test
    fun reconnectsWithTheSameDeviceTokenAndResetsSequence() {
        val first = FakeControllerServer()
        val second = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(first))
        server.enqueue(MockResponse().withWebSocketUpgrade(second))

        startClient(backoffMs = 20L)
        first.awaitOpen()
        first.awaitMessage("hello")
        first.sendWelcome(slot = 4, phase = "playing")
        awaitState(ConnectionState.CONNECTED)

        scheduler.advanceBy(200L)
        val beforeDrop = first.awaitMessages("input", count = 2)
        assertTrue(beforeDrop[1].getLong("seq") > beforeDrop[0].getLong("seq"))

        first.socket?.close(1001, "server restart")
        awaitState(ConnectionState.RECONNECTING)
        scheduler.advanceBy(20L)

        second.awaitOpen()
        val hello = second.awaitMessage("hello")
        assertEquals("device-token-1", hello.getJSONObject("data").getString("deviceToken"))
        assertEquals(2, server.requestCount)
        assertTrue(listener.states.contains(ConnectionState.RECONNECTING))

        second.sendWelcome(slot = 4, phase = "playing")
        awaitState(ConnectionState.CONNECTED)
        assertEquals(4, listener.welcome?.slot)

        scheduler.advanceBy(200L)
        val afterReconnect = second.awaitMessages("input", count = 2)
        assertEquals("Sequence must restart after welcome.", 0L, afterReconnect[0].getLong("seq"))
    }

    @Test
    fun authFailureIsFatalAndDoesNotReconnect() = assertFatalError("auth_failed")

    @Test
    fun kickedIsFatalAndDoesNotReconnect() = assertFatalError("kicked")

    @Test
    fun protocolMismatchIsFatalAndDoesNotReconnect() = assertFatalError("protocol_mismatch")

    @Test
    fun sessionFullRetriesInsteadOfSticking() {
        val first = FakeControllerServer()
        val second = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(first))
        server.enqueue(MockResponse().withWebSocketUpgrade(second))

        startClient(backoffMs = 20L)
        first.awaitOpen()
        first.awaitMessage("hello")
        first.sendError("session_full", "No free player slot.")

        awaitState(ConnectionState.RECONNECTING)
        scheduler.advanceBy(20L)

        second.awaitOpen()
        val hello = second.awaitMessage("hello")
        assertEquals("device-token-1", hello.getJSONObject("data").getString("deviceToken"))
        assertEquals(2, server.requestCount)
        assertTrue(listener.errors.contains("session_full"))
    }

    @Test
    fun duplicateTokenRetriesWithTheSameTokenAndBackoff() {
        val first = FakeControllerServer()
        val second = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(first))
        server.enqueue(MockResponse().withWebSocketUpgrade(second))

        startClient(backoffMs = 20L)
        first.awaitOpen()
        first.awaitMessage("hello")
        first.sendError("token_in_use", "This device token is already connected.")

        awaitState(ConnectionState.RECONNECTING)
        assertTrue("Duplicate tokens must not become fatal.", listener.state != ConnectionState.FATAL)
        scheduler.advanceBy(20L)

        second.awaitOpen()
        val hello = second.awaitMessage("hello")
        assertEquals("device-token-1", hello.getJSONObject("data").getString("deviceToken"))
        assertEquals(2, server.requestCount)
        assertTrue(listener.errors.contains("token_in_use"))
    }

    @Test
    fun gameUnavailableKeepsConnectionAndGatesInput() {
        val fake = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(fake))

        startClient(backoffMs = 20L)
        fake.awaitOpen()
        fake.awaitMessage("hello")
        fake.sendWelcome(slot = 1, phase = "playing")
        awaitState(ConnectionState.CONNECTED)
        scheduler.advanceBy(1_000L)
        val sentBeforeLoss = listener.lastStats?.sentTotal ?: 0L
        assertTrue(sentBeforeLoss > 0L)

        fake.sendError("game_unavailable", "Game is unavailable. Waiting for restart.")
        awaitServerMessage(
            { it is ServerMessage.ServerError && it.code == "game_unavailable" },
            "game_unavailable error",
        )
        fake.sendPhase("attract")
        awaitServerMessage({ it is ServerMessage.Phase && it.phase == "attract" }, "attract phase")

        fake.drainMessages("input")
        scheduler.advanceBy(5_000L)
        assertEquals("No reconnect after game loss.", 1, server.requestCount)
        assertEquals(ConnectionState.CONNECTED, listener.state)
        assertEquals(sentBeforeLoss, listener.lastStats?.sentTotal)
        assertEquals(0, fake.drainMessages("input").size)

        fake.sendPhase("playing")
        awaitServerMessage({ it is ServerMessage.Phase && it.phase == "playing" }, "playing phase")
        scheduler.advanceBy(200L)
        assertTrue(fake.awaitMessages("input", count = 1).isNotEmpty())
        assertEquals("The same socket must stay in use.", 1, server.requestCount)
    }

    @Test
    fun malformedServerFrameIsToleratedAndConnectionStaysOpen() {
        val fake = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(fake))

        startClient()
        fake.awaitOpen()
        fake.awaitMessage("hello")
        fake.sendWelcome(slot = 1, phase = "ready")
        awaitState(ConnectionState.CONNECTED)

        fake.sendRaw("{not valid json")
        fake.sendPhase("attract")
        awaitServerMessage({ it is ServerMessage.Phase && it.phase == "attract" }, "attract phase")
        scheduler.advanceBy(1_000L)

        assertEquals(ConnectionState.CONNECTED, listener.state)
        assertEquals(1, server.requestCount)
        assertTrue(
            "Parse failure should be reported in diagnostics.",
            listener.lastStats?.lastError?.contains("ignored frame") == true,
        )
    }

    @Test
    fun pongWatchdogReconnectsAfterSilence() {
        val first = FakeControllerServer()
        val second = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(first))
        server.enqueue(MockResponse().withWebSocketUpgrade(second))

        startClient(backoffMs = 20L, pongTimeoutMs = 6_000L)
        first.awaitOpen()
        first.awaitMessage("hello")
        first.sendWelcome(slot = 1, phase = "ready")
        awaitState(ConnectionState.CONNECTED)

        scheduler.advanceBy(2_000L)
        val ping = first.awaitMessage("ping")
        val clientTime = ping.getJSONObject("data").getLong("clientTime")
        first.sendPong(clientTime)
        awaitServerMessage({ it is ServerMessage.Pong }, "pong")

        scheduler.advanceBy(4_000L)
        assertEquals("Fresh pong must keep the socket.", 1, server.requestCount)

        scheduler.advanceBy(10_000L)
        second.awaitOpen()
        assertEquals(2, server.requestCount)
    }

    @Test
    fun stopSendsLeaveAndAllowsRestart() {
        val first = FakeControllerServer()
        val second = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(first))
        server.enqueue(MockResponse().withWebSocketUpgrade(second))

        startClient()
        first.awaitOpen()
        first.awaitMessage("hello")
        first.sendWelcome(slot = 1, phase = "ready")
        awaitState(ConnectionState.CONNECTED)

        client!!.stop()
        first.awaitMessage("leave", timeoutMs = 2_000L)
        assertTrue(first.awaitClosed(2_000L))
        assertEquals(ConnectionState.IDLE, listener.state)

        client!!.start()
        second.awaitOpen()
        second.awaitMessage("hello")
        second.sendWelcome(slot = 1, phase = "ready")
        awaitState(ConnectionState.CONNECTED)
    }

    @Test
    fun baseAddressIsExpandedToTheControllerEndpoint() {
        assertEquals(
            "ws://192.168.1.10:8080/ws/controller",
            controllerEndpoint("ws://192.168.1.10:8080"),
        )
        assertEquals(
            "ws://192.168.1.10:8080/ws/controller",
            controllerEndpoint("ws://192.168.1.10:8080/"),
        )
        assertEquals(
            "ws://192.168.1.10:8080/ws/controller",
            controllerEndpoint("192.168.1.10:8080"),
        )
        assertEquals(
            "ws://192.168.1.10:8080/ws/controller",
            controllerEndpoint("WS://192.168.1.10:8080"),
        )
        assertEquals(
            "wss://example.test/ws/controller",
            controllerEndpoint("wss://example.test"),
        )
        assertEquals(
            "ws://192.168.1.10:8080/custom",
            controllerEndpoint("ws://192.168.1.10:8080/custom"),
        )
        assertEquals(
            "ws://192.168.1.10:8080/ws/controller",
            controllerEndpoint("http://192.168.1.10:8080"),
        )
    }

    @Test
    fun invalidAddressIsRejectedBeforeConnecting() {
        try {
            controllerEndpoint("not a server address")
            fail("Expected an IllegalArgumentException.")
        } catch (expected: IllegalArgumentException) {
            // Expected: connectNow reports this as a fatal state.
        }
    }

    private fun assertFatalError(code: String) {
        val fake = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(fake))

        startClient(backoffMs = 20L)
        fake.awaitOpen()
        fake.awaitMessage("hello")
        fake.sendError(code, "fatal test error")

        awaitState(ConnectionState.FATAL)
        assertTrue(listener.errors.contains(code))
        assertTrue("Fatal errors must close the socket.", fake.awaitClosed(2_000L))

        scheduler.advanceBy(1_000L)
        assertEquals(1, server.requestCount)
    }

    private fun startClient(
        backoffMs: Long = 500L,
        pongTimeoutMs: Long = 6_000L,
    ) {
        val newClient = ControllerClient(
            serverUrl = "ws://${server.hostName}:${server.port}",
            deviceToken = "device-token-1",
            appVersion = "0.1.0-test",
            inputProvider = { InputState(steer = 1, throttle = -1) },
            listener = listener,
            timeSource = { scheduler.nowMs },
            inputHz = ControllerClient.DEFAULT_INPUT_HZ,
            pongTimeoutMs = pongTimeoutMs,
            backoffBaseMs = backoffMs,
            backoffCapMs = backoffMs,
            schedulerFactory = { scheduler },
        )
        client = newClient
        newClient.start()
    }

    private fun awaitState(expected: ConnectionState, timeoutMs: Long = 5_000L) {
        val deadline = System.currentTimeMillis() + timeoutMs
        while (listener.state != expected) {
            if (System.currentTimeMillis() > deadline) {
                throw AssertionError(
                    "Expected state $expected, last state was ${listener.state} " +
                        "(history: ${listener.states}).",
                )
            }
            Thread.sleep(10L)
        }
    }

    private fun awaitServerMessage(
        predicate: (ServerMessage) -> Boolean,
        description: String,
        timeoutMs: Long = 5_000L,
    ) {
        val deadline = System.currentTimeMillis() + timeoutMs
        while (listener.messages.none(predicate)) {
            if (System.currentTimeMillis() > deadline) {
                throw AssertionError("Timed out waiting for $description; got ${listener.messages}.")
            }
            Thread.sleep(10L)
        }
    }
}
