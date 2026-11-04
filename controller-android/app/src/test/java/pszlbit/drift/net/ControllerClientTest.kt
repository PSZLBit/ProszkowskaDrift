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

/**
 * Fake-server tests for the controller client.
 *
 * MockWebServer owns a real local socket and plays the Go server side: it
 * accepts `/ws/controller`, receives `hello` and `input`, and can push
 * `welcome`, `phase`, and `error` frames. No venue network or tablet is needed.
 */
class ControllerClientTest {
    private lateinit var server: MockWebServer
    private lateinit var listener: RecordingControllerListener
    private var client: ControllerClient? = null

    @Before
    fun setUp() {
        server = MockWebServer()
        server.start()
        listener = RecordingControllerListener()
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

        Thread.sleep(150L)
        assertEquals(0, fake.drainMessages("input").size)

        fake.sendPhase("playing")
        val inputs = fake.awaitMessages("input", count = 3, timeoutMs = 3_000L)
        val first = inputs[0]
        assertEquals(1.0, first.getJSONObject("data").getDouble("steer"), 1e-9)
        assertEquals(-1.0, first.getJSONObject("data").getDouble("throttle"), 1e-9)
        assertTrue("Sequence numbers must increase.", first.getLong("seq") < inputs[1].getLong("seq"))

        fake.sendPhase("paused")
        Thread.sleep(250L)
        fake.drainMessages("input")
        Thread.sleep(250L)
        assertEquals(0, fake.drainMessages("input").size)
    }

    @Test
    fun reconnectsWithTheSameDeviceTokenAfterServerDrop() {
        val first = FakeControllerServer()
        val second = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(first))
        server.enqueue(MockResponse().withWebSocketUpgrade(second))

        startClient(backoffMs = 20L)
        first.awaitOpen()
        first.awaitMessage("hello")
        first.sendWelcome(slot = 4, phase = "playing")
        awaitState(ConnectionState.CONNECTED)

        first.socket?.close(1001, "server restart")

        second.awaitOpen()
        val hello = second.awaitMessage("hello")
        assertEquals("device-token-1", hello.getJSONObject("data").getString("deviceToken"))
        assertEquals(2, server.requestCount)
        assertTrue(listener.states.contains(ConnectionState.RECONNECTING))

        second.sendWelcome(slot = 4, phase = "playing")
        awaitState(ConnectionState.CONNECTED)
        assertEquals(4, listener.welcome?.slot)
    }

    @Test
    fun authFailureIsFatalAndDoesNotReconnect() {
        val fake = FakeControllerServer()
        server.enqueue(MockResponse().withWebSocketUpgrade(fake))

        startClient(backoffMs = 20L)
        fake.awaitOpen()
        fake.awaitMessage("hello")
        fake.sendError("auth_failed", "Unknown device token.")

        awaitState(ConnectionState.FATAL)
        assertTrue(listener.errors.contains("auth_failed"))
        assertTrue("Fatal errors must close the socket.", fake.awaitClosed(2_000L))

        Thread.sleep(250L)
        assertEquals(1, server.requestCount)
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

    private fun startClient(backoffMs: Long = 500L) {
        val newClient = ControllerClient(
            serverUrl = "ws://${server.hostName}:${server.port}",
            deviceToken = "device-token-1",
            appVersion = "0.1.0-test",
            inputProvider = { InputState(steer = 1, throttle = -1) },
            listener = listener,
            inputHz = ControllerClient.DEFAULT_INPUT_HZ,
            backoffBaseMs = backoffMs,
            backoffCapMs = backoffMs,
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
}
