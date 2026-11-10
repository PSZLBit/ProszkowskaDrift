package pszlbit.drift.net

import java.util.concurrent.CountDownLatch
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject

/**
 * One side of a fake controller server, used with OkHttp MockWebServer.
 *
 * It accepts a single WebSocket upgrade, records every client frame, and lets a
 * test push `welcome`, `phase`, and `error` frames back to the controller.
 */
class FakeControllerServer : WebSocketListener() {
    private val opened = CountDownLatch(1)
    private val closed = CountDownLatch(1)
    private val received = LinkedBlockingQueue<String>()

    @Volatile
    var socket: WebSocket? = null
        private set

    override fun onOpen(webSocket: WebSocket, response: Response) {
        socket = webSocket
        opened.countDown()
    }

    override fun onMessage(webSocket: WebSocket, text: String) {
        received.add(text)
    }

    override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
        webSocket.close(NORMAL_CLOSE, null)
    }

    override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
        closed.countDown()
    }

    override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
        closed.countDown()
    }

    fun awaitOpen(timeoutMs: Long = TIMEOUT_MS) {
        if (!opened.await(timeoutMs, TimeUnit.MILLISECONDS)) {
            throw AssertionError("Fake server did not receive a WebSocket connection.")
        }
    }

    fun awaitClosed(timeoutMs: Long = TIMEOUT_MS): Boolean =
        closed.await(timeoutMs, TimeUnit.MILLISECONDS)

    fun send(json: String) {
        val current = socket ?: throw AssertionError("Fake server socket is not open.")
        if (!current.send(json)) {
            throw AssertionError("Fake server could not send: $json")
        }
    }

    fun sendWelcome(slot: Int = 1, phase: String = "ready") {
        send(
            envelope(
                "welcome",
                JSONObject()
                    .put("slot", slot)
                    .put("label", "P$slot")
                    .put("color", "blue")
                    .put("phase", phase),
            ),
        )
    }

    fun sendPhase(phase: String, timeLeftMs: Long = 0L) {
        send(
            envelope(
                "phase",
                JSONObject()
                    .put("phase", phase)
                    .put("timeLeftMs", timeLeftMs)
                    .put("roundId", 1),
            ),
        )
    }

    fun sendError(code: String, message: String = "test error") {
        send(
            envelope(
                "error",
                JSONObject().put("code", code).put("message", message),
            ),
        )
    }

    fun sendPong(clientTime: Long, serverTime: Long = 1L) {
        send(
            envelope(
                "pong",
                JSONObject().put("clientTime", clientTime).put("serverTime", serverTime),
            ),
        )
    }

    fun sendRaw(text: String) {
        val current = socket ?: throw AssertionError("Fake server socket is not open.")
        if (!current.send(text)) {
            throw AssertionError("Fake server could not send raw frame: $text")
        }
    }

    fun awaitMessage(type: String, timeoutMs: Long = TIMEOUT_MS): JSONObject {
        val deadline = System.currentTimeMillis() + timeoutMs
        while (System.currentTimeMillis() < deadline) {
            val remaining = (deadline - System.currentTimeMillis()).coerceAtLeast(1L)
            val text = received.poll(remaining, TimeUnit.MILLISECONDS) ?: break
            val json = JSONObject(text)
            if (json.optString("type") == type) return json
        }
        throw AssertionError("No '$type' message received within ${timeoutMs}ms.")
    }

    fun awaitMessages(type: String, count: Int, timeoutMs: Long = TIMEOUT_MS): List<JSONObject> {
        val deadline = System.currentTimeMillis() + timeoutMs
        val result = mutableListOf<JSONObject>()
        while (result.size < count && System.currentTimeMillis() < deadline) {
            val remaining = (deadline - System.currentTimeMillis()).coerceAtLeast(1L)
            val text = received.poll(remaining, TimeUnit.MILLISECONDS) ?: break
            val json = JSONObject(text)
            if (json.optString("type") == type) result.add(json)
        }
        if (result.size < count) {
            throw AssertionError("Expected $count '$type' messages, got ${result.size}.")
        }
        return result
    }

    fun drainMessages(type: String): List<JSONObject> {
        val result = mutableListOf<JSONObject>()
        while (true) {
            val text = received.poll() ?: break
            val json = JSONObject(text)
            if (json.optString("type") == type) result.add(json)
        }
        return result
    }

    private fun envelope(type: String, data: JSONObject): String =
        JSONObject()
            .put("v", 1)
            .put("type", type)
            .put("data", data)
            .toString()

    private companion object {
        const val NORMAL_CLOSE = 1000
        const val TIMEOUT_MS = 5_000L
    }
}
