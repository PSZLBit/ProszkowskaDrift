package pszlbit.drift.net

import okhttp3.HttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import pszlbit.drift.input.InputState
import pszlbit.drift.protocol.Protocol
import pszlbit.drift.protocol.ProtocolException
import pszlbit.drift.protocol.ServerMessage
import java.util.ArrayDeque
import java.util.Locale
import java.util.concurrent.Executors
import java.util.concurrent.RejectedExecutionException
import java.util.concurrent.ScheduledExecutorService
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean

enum class ConnectionState {
    IDLE,
    CONNECTING,
    CONNECTED,
    RECONNECTING,
    FATAL,
}

/** Snapshot for the diagnostics line; measured on the client. */
data class ClientStats(
    val state: ConnectionState,
    val phase: String,
    val sentTotal: Long,
    val measuredHz: Double,
    val lastRttMs: Long,
    val averageRttMs: Long,
    val reconnects: Int,
    val sequence: Long,
    val lastError: String?,
)

/**
 * Controller WebSocket client.
 *
 * Responsibilities: handshake with `hello`, input at [DEFAULT_INPUT_HZ] while
 * the server phase is `playing`, a 2 s application `ping`, and reconnect with
 * exponential backoff using the same device token (protocol.md, Reconnect).
 *
 * The server counts input outside `playing` as an invalid message and closes
 * the connection after repeated violations, so input is gated on phase.
 *
 * No Android APIs are used here: the client is exercised by JVM tests against
 * OkHttp's MockWebServer, which plays the Go server side.
 */
class ControllerClient(
    private val serverUrl: String,
    private val deviceToken: String,
    private val appVersion: String,
    private val inputProvider: () -> InputState,
    private val listener: Listener,
    private val timeSource: () -> Long = { System.nanoTime() / 1_000_000L },
    private val inputHz: Int = DEFAULT_INPUT_HZ,
    private val pingIntervalMs: Long = 2_000L,
    private val backoffBaseMs: Long = 500L,
    private val backoffCapMs: Long = 10_000L,
) {
    interface Listener {
        fun onStateChanged(state: ConnectionState, detail: String?)
        fun onMessage(message: ServerMessage)
        fun onStats(stats: ClientStats)
    }

    private val scheduler: ScheduledExecutorService =
        Executors.newSingleThreadScheduledExecutor { runnable ->
            Thread(runnable, "controller-client").apply { isDaemon = true }
        }

    private val httpClient: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(5, TimeUnit.SECONDS)
        .readTimeout(0, TimeUnit.MILLISECONDS)
        .build()

    private val running = AtomicBoolean(false)
    private val started = AtomicBoolean(false)
    private val reconnectScheduled = AtomicBoolean(false)

    @Volatile
    private var socket: WebSocket? = null

    @Volatile
    private var state = ConnectionState.IDLE

    @Volatile
    private var phase = "attract"

    @Volatile
    private var sequence = 0L

    @Volatile
    private var fatal = false

    @Volatile
    private var attempt = 0

    @Volatile
    private var reconnects = 0

    @Volatile
    private var lastError: String? = null

    private var sentTotal = 0L
    private var sentThisSecond = 0L
    private var measuredHz = 0.0
    private var lastRttMs = -1L
    private val rttSamples = ArrayDeque<Long>()

    fun start() {
        if (!started.compareAndSet(false, true)) return
        running.set(true)
        connectNow()
        // Fixed delay, not fixed rate: after a GC or scheduling pause the client
        // must not emit a catch-up burst above the 20–30 Hz input budget.
        scheduler.scheduleWithFixedDelay(
            { sendInput() },
            0L,
            maxOf(1L, 1_000L / inputHz),
            TimeUnit.MILLISECONDS,
        )
        scheduler.scheduleAtFixedRate(
            { sendPing() },
            0L,
            pingIntervalMs,
            TimeUnit.MILLISECONDS,
        )
        scheduler.scheduleAtFixedRate(
            { publishStats() },
            1_000L,
            1_000L,
            TimeUnit.MILLISECONDS,
        )
    }

    fun stop() {
        if (!running.compareAndSet(true, false)) return
        fatal = true
        reconnectScheduled.set(false)
        closeSocket(CLOSE_NORMAL, "client stopped")
        scheduler.shutdownNow()
        httpClient.dispatcher().executorService().shutdown()
        httpClient.connectionPool().evictAll()
        state = ConnectionState.IDLE
        listener.onStateChanged(state, "stopped")
    }

    private fun connectNow() {
        if (!running.get()) return
        val request = try {
            Request.Builder().url(controllerEndpoint(serverUrl)).build()
        } catch (e: IllegalArgumentException) {
            fatal = true
            lastError = "Invalid server URL: $serverUrl"
            setState(ConnectionState.FATAL, lastError)
            return
        }
        setState(ConnectionState.CONNECTING, null)
        httpClient.newWebSocket(request, SocketListener())
    }

    private fun sendInput() {
        if (!running.get()) return
        val ws = socket ?: return
        if (state != ConnectionState.CONNECTED || phase != PHASE_PLAYING) return
        val input = inputProvider()
        val seq = sequence
        sequence = seq + 1
        val sent = ws.send(
            Protocol.input(
                seq = seq,
                steer = input.steer.toDouble(),
                throttle = input.throttle.toDouble(),
                ts = timeSource(),
            ),
        )
        if (sent) {
            sentTotal++
            sentThisSecond++
        }
    }

    private fun sendPing() {
        if (!running.get()) return
        val ws = socket ?: return
        if (state != ConnectionState.CONNECTED) return
        ws.send(Protocol.ping(timeSource()))
    }

    private fun publishStats() {
        if (!running.get()) return
        measuredHz = sentThisSecond.toDouble()
        sentThisSecond = 0L
        listener.onStats(stats())
    }

    private fun stats(): ClientStats {
        val average = if (rttSamples.isEmpty()) -1L else rttSamples.average().toLong()
        return ClientStats(
            state = state,
            phase = phase,
            sentTotal = sentTotal,
            measuredHz = measuredHz,
            lastRttMs = lastRttMs,
            averageRttMs = average,
            reconnects = reconnects,
            sequence = sequence,
            lastError = lastError,
        )
    }

    private fun setState(newState: ConnectionState, detail: String?) {
        state = newState
        listener.onStateChanged(newState, detail)
    }

    private fun closeSocket(code: Int, reason: String) {
        val current = socket
        socket = null
        current?.close(code, reason)
    }

    private fun scheduleReconnect(reason: String) {
        if (!running.get() || fatal) return
        // OkHttp can report a failed connection through both onClosed and
        // onFailure; schedule only one retry per drop.
        if (!reconnectScheduled.compareAndSet(false, true)) return
        lastError = reason
        val delay = Backoff.delayMs(attempt, backoffBaseMs, backoffCapMs)
        attempt++
        reconnects++
        setState(
            ConnectionState.RECONNECTING,
            String.format(Locale.US, "retry in %d ms: %s", delay, reason),
        )
        try {
            scheduler.schedule(
                {
                    reconnectScheduled.set(false)
                    connectNow()
                },
                delay,
                TimeUnit.MILLISECONDS,
            )
        } catch (e: RejectedExecutionException) {
            reconnectScheduled.set(false)
        }
    }

    private fun handleMessage(text: String) {
        val message = try {
            Protocol.parse(text)
        } catch (e: ProtocolException) {
            lastError = e.message
            return
        }
        when (message) {
            is ServerMessage.Welcome -> {
                phase = message.phase
                sequence = 0L
                attempt = 0
                reconnectScheduled.set(false)
                fatal = false
                lastError = null
                setState(ConnectionState.CONNECTED, null)
            }

            is ServerMessage.Phase -> {
                phase = message.phase
            }

            is ServerMessage.Pong -> {
                val rtt = (timeSource() - message.clientTime).coerceAtLeast(0L)
                lastRttMs = rtt
                rttSamples.addLast(rtt)
                while (rttSamples.size > RTT_SAMPLES) rttSamples.removeFirst()
            }

            is ServerMessage.ServerError -> {
                lastError = "${message.code}: ${message.message}"
                if (message.code in FATAL_ERRORS) {
                    fatal = true
                    reconnectScheduled.set(false)
                    closeSocket(CLOSE_NORMAL, "fatal: ${message.code}")
                    setState(ConnectionState.FATAL, lastError)
                }
            }

            is ServerMessage.Unknown -> Unit
            else -> Unit
        }
        listener.onMessage(message)
    }

    private inner class SocketListener : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
            if (!running.get()) {
                webSocket.close(CLOSE_NORMAL, "client stopped")
                return
            }
            socket = webSocket
            setState(ConnectionState.CONNECTING, "handshake")
            webSocket.send(Protocol.hello(deviceToken, appVersion))
        }

        override fun onMessage(webSocket: WebSocket, text: String) {
            if (!running.get()) return
            handleMessage(text)
        }

        override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
            webSocket.close(CLOSE_NORMAL, null)
        }

        override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
            if (socket === webSocket) socket = null
            if (!running.get()) return
            scheduleReconnect("connection closed ($code ${reason.trim()})")
        }

        override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
            if (socket === webSocket) socket = null
            if (!running.get()) return
            scheduleReconnect(t.message ?: t.javaClass.simpleName)
        }
    }

    companion object {
        /** D-23 allows 20–30 Hz; 25 Hz leaves headroom on the old tablets. */
        const val DEFAULT_INPUT_HZ = 25

        private const val CLOSE_NORMAL = 1000
        private const val RTT_SAMPLES = 10
        private const val PHASE_PLAYING = "playing"
        private val FATAL_ERRORS = setOf("auth_failed", "kicked", "protocol_mismatch")
    }
}

/**
 * Accepts a base address such as `ws://192.168.1.10:8080` or a full endpoint.
 * A base address is expanded to the controller endpoint `/ws/controller`.
 * `http(s)` input is accepted for tests and converted back to `ws(s)`.
 */
internal fun controllerEndpoint(serverUrl: String): String {
    val trimmed = serverUrl.trim()
    val secure = trimmed.startsWith("wss:", ignoreCase = true) ||
        trimmed.startsWith("https:", ignoreCase = true)
    val httpUrl = when {
        trimmed.startsWith("ws:", ignoreCase = true) -> "http:" + trimmed.substring(3)
        trimmed.startsWith("wss:", ignoreCase = true) -> "https:" + trimmed.substring(4)
        else -> trimmed
    }
    val parsed = HttpUrl.parse(httpUrl)
        ?: throw IllegalArgumentException("Invalid server URL: $serverUrl")
    val path = if (parsed.encodedPath() == "/") CONTROLLER_PATH else parsed.encodedPath()
    val normalized = parsed.newBuilder().encodedPath(path).build().toString()
    return if (secure) {
        "wss" + normalized.substring("https".length)
    } else {
        "ws" + normalized.substring("http".length)
    }
}

private const val CONTROLLER_PATH = "/ws/controller"
