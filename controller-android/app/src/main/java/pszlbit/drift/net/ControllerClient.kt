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
import java.util.concurrent.ScheduledFuture
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger

enum class ConnectionState {
    IDLE,
    CONNECTING,
    CONNECTED,
    RECONNECTING,
    RETRY_EXHAUSTED,
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

/** Minimal scheduler boundary so host-JVM tests can drive a fake clock. */
internal interface TaskScheduler {
    fun schedule(delayMs: Long, task: () -> Unit): ScheduledTask

    fun scheduleWithFixedDelay(
        initialDelayMs: Long,
        delayMs: Long,
        task: () -> Unit,
    ): ScheduledTask

    fun shutdownNow()
}

internal interface ScheduledTask {
    fun cancel()
}

private class ExecutorScheduler : TaskScheduler {
    private val executor =
        Executors.newSingleThreadScheduledExecutor { runnable ->
            Thread(runnable, "controller-client").apply { isDaemon = true }
        }

    override fun schedule(delayMs: Long, task: () -> Unit): ScheduledTask =
        Handle(executor.schedule({ task() }, delayMs, TimeUnit.MILLISECONDS))

    override fun scheduleWithFixedDelay(
        initialDelayMs: Long,
        delayMs: Long,
        task: () -> Unit,
    ): ScheduledTask =
        Handle(
            executor.scheduleWithFixedDelay(
                { task() },
                initialDelayMs,
                delayMs,
                TimeUnit.MILLISECONDS,
            ),
        )

    override fun shutdownNow() {
        executor.shutdownNow()
    }

    private class Handle(private val future: ScheduledFuture<*>) : ScheduledTask {
        override fun cancel() {
            future.cancel(false)
        }
    }
}

/**
 * Controller WebSocket client.
 *
 * Responsibilities: handshake with `hello`, input at [DEFAULT_INPUT_HZ] while
 * the server phase is `playing`, a 2 s application `ping`, a bounded pong
 * watchdog, and bounded reconnect with exponential backoff using the same
 * device token (protocol.md, Reconnect). After
 * [DEFAULT_MAX_RECONNECT_ATTEMPTS] failures the client emits
 * [ConnectionState.RETRY_EXHAUSTED] and stops.
 *
 * The server counts input outside `playing` as an invalid message and closes
 * the connection after repeated violations, so input is gated on phase. A
 * `game_unavailable` error keeps the current socket: the server follows it with
 * `phase: attract` and input stays gated until the game returns. Duplicate
 * tokens use `token_in_use` and retry; `auth_failed` remains permanent.
 *
 * Malformed or unsupported-version server frames are tolerated: they are
 * recorded in `lastError` and ignored. The server owns connection policy, so
 * reconnecting after one bad frame would add avoidable churn.
 *
 * No Android APIs are used here: the client is exercised by JVM tests against
 * OkHttp's MockWebServer, which plays the Go server side.
 */
class ControllerClient internal constructor(
    private val serverUrl: String,
    private val deviceToken: String,
    private val appVersion: String,
    private val inputProvider: () -> InputState,
    private val listener: Listener,
    private val timeSource: () -> Long = { System.nanoTime() / 1_000_000L },
    private val inputHz: Int = DEFAULT_INPUT_HZ,
    private val pingIntervalMs: Long = 2_000L,
    private val pongTimeoutMs: Long = 6_000L,
    private val backoffBaseMs: Long = 500L,
    private val backoffCapMs: Long = 10_000L,
    private val maxReconnectAttempts: Int = DEFAULT_MAX_RECONNECT_ATTEMPTS,
    private val schedulerFactory: () -> TaskScheduler = { ExecutorScheduler() },
) {
    interface Listener {
        fun onStateChanged(state: ConnectionState, detail: String?)
        fun onMessage(message: ServerMessage)
        fun onStats(stats: ClientStats)
    }

    private var scheduler: TaskScheduler? = null
    private var httpClient: OkHttpClient? = null

    private val running = AtomicBoolean(false)
    private val started = AtomicBoolean(false)
    private val reconnectScheduled = AtomicBoolean(false)
    private val generation = AtomicInteger(0)

    @Volatile
    private var pendingReconnect: ScheduledTask? = null

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

    @Volatile
    private var lastPongAtMs = 0L

    @Volatile
    private var lastRttMs = -1L

    private val rttLock = Any()
    private val rttSamples = ArrayDeque<Long>()

    private var sentTotal = 0L
    private var sentThisSecond = 0L
    private var measuredHz = 0.0

    fun start() {
        if (!started.compareAndSet(false, true)) return
        running.set(true)
        fatal = false
        attempt = 0
        reconnects = 0
        sequence = 0L
        phase = "attract"
        lastError = null
        lastPongAtMs = timeSource()
        lastRttMs = -1L
        sentTotal = 0L
        sentThisSecond = 0L
        measuredHz = 0.0
        synchronized(rttLock) { rttSamples.clear() }
        reconnectScheduled.set(false)
        pendingReconnect?.cancel()
        pendingReconnect = null

        scheduler = schedulerFactory()
        httpClient = OkHttpClient.Builder()
            .connectTimeout(5, TimeUnit.SECONDS)
            .readTimeout(0, TimeUnit.MILLISECONDS)
            .build()
        connectNow()

        val activeScheduler = scheduler ?: return
        // Fixed delay, not fixed rate: after a GC or scheduling pause the client
        // must not emit a catch-up burst above the 20–30 Hz input budget.
        activeScheduler.scheduleWithFixedDelay(
            0L,
            maxOf(1L, 1_000L / inputHz),
        ) { runSafely { sendInput() } }
        activeScheduler.scheduleWithFixedDelay(
            0L,
            pingIntervalMs,
        ) { runSafely { sendPing() } }
        activeScheduler.scheduleWithFixedDelay(
            1_000L,
            1_000L,
        ) { runSafely { publishStats() } }
        activeScheduler.scheduleWithFixedDelay(
            maxOf(250L, pingIntervalMs / 2),
            maxOf(250L, pingIntervalMs / 2),
        ) { runSafely { watchPong() } }
    }

    fun stop() {
        if (!started.compareAndSet(true, false)) return
        running.set(false)

        val current = socket
        if (current != null) {
            current.send(Protocol.leave())
            socket = null
            current.close(CLOSE_NORMAL, "client stopped")
        }

        shutdownResources("client stopped")
        state = ConnectionState.IDLE
        listener.onStateChanged(state, "stopped")
    }

    private fun connectNow() {
        if (!running.get() || fatal) return
        val client = httpClient ?: return
        val request = try {
            Request.Builder().url(controllerEndpoint(serverUrl)).build()
        } catch (e: IllegalArgumentException) {
            fatal = true
            lastError = "Invalid server URL: $serverUrl"
            setState(ConnectionState.FATAL, lastError)
            return
        }
        val socketGeneration = generation.incrementAndGet()
        setState(ConnectionState.CONNECTING, null)
        try {
            client.newWebSocket(request, SocketListener(socketGeneration))
        } catch (e: RejectedExecutionException) {
            if (!running.get()) return
            lastError = "connect rejected: ${e.message}"
            scheduleReconnect(lastError ?: "connect rejected")
        }
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

    private fun watchPong() {
        if (!running.get() || state != ConnectionState.CONNECTED) return
        val silentForMs = timeSource() - lastPongAtMs
        if (silentForMs <= pongTimeoutMs) return
        val reason = "pong timeout after $silentForMs ms"
        lastError = reason
        closeSocket(CLOSE_GOING_AWAY, "pong timeout")
        scheduleReconnect(reason)
    }

    private fun publishStats() {
        if (!running.get()) return
        measuredHz = sentThisSecond.toDouble()
        sentThisSecond = 0L
        listener.onStats(stats())
    }

    private fun stats(): ClientStats {
        val average = synchronized(rttLock) {
            if (rttSamples.isEmpty()) -1L else rttSamples.average().toLong()
        }
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
        if (!running.get() && newState != ConnectionState.IDLE) return
        state = newState
        listener.onStateChanged(newState, detail)
    }

    private fun closeSocket(code: Int, reason: String) {
        val current = socket
        socket = null
        current?.close(code, reason)
    }

    private fun shutdownResources(closeReason: String) {
        reconnectScheduled.set(false)
        pendingReconnect?.cancel()
        pendingReconnect = null
        generation.incrementAndGet()

        val current = socket
        if (current != null) {
            socket = null
            current.close(CLOSE_NORMAL, closeReason)
        }

        scheduler?.shutdownNow()
        scheduler = null
        httpClient?.let { client ->
            client.dispatcher().executorService().shutdown()
            client.connectionPool().evictAll()
        }
        httpClient = null
    }

    private fun cancelReconnect() {
        reconnectScheduled.set(false)
        pendingReconnect?.cancel()
        pendingReconnect = null
    }

    private fun scheduleReconnect(reason: String) {
        if (!running.get() || fatal) return
        // OkHttp can report a failed connection through both onClosed and
        // onFailure; schedule only one retry per drop.
        if (!reconnectScheduled.compareAndSet(false, true)) return
        lastError = reason
        if (attempt >= maxReconnectAttempts) {
            reconnectScheduled.set(false)
            failReconnects(reason)
            return
        }
        val delay = Backoff.delayMs(attempt, backoffBaseMs, backoffCapMs)
        attempt++
        reconnects++
        setState(
            ConnectionState.RECONNECTING,
            String.format(Locale.US, "retry in %d ms: %s", delay, reason),
        )
        val scheduledGeneration = generation.get()
        val activeScheduler = scheduler
        if (activeScheduler == null) {
            reconnectScheduled.set(false)
            return
        }
        try {
            pendingReconnect = activeScheduler.schedule(delay) {
                pendingReconnect = null
                reconnectScheduled.set(false)
                if (running.get() && !fatal && generation.get() == scheduledGeneration) {
                    connectNow()
                }
            }
        } catch (e: RejectedExecutionException) {
            pendingReconnect = null
            reconnectScheduled.set(false)
        }
    }

    private fun failReconnects(reason: String) {
        if (!started.compareAndSet(true, false)) return
        running.set(false)
        fatal = true
        lastError = reason
        shutdownResources("reconnect attempts exhausted")
        state = ConnectionState.RETRY_EXHAUSTED
        listener.onStateChanged(
            state,
            "$maxReconnectAttempts reconnect attempts failed",
        )
    }

    private fun handleMessage(text: String) {
        if (!running.get()) return
        val message = try {
            Protocol.parse(text)
        } catch (e: ProtocolException) {
            lastError = "ignored frame: ${e.message}"
            return
        }
        when (message) {
            is ServerMessage.Welcome -> {
                cancelReconnect()
                phase = message.phase
                sequence = 0L
                attempt = 0
                fatal = false
                lastError = null
                lastPongAtMs = timeSource()
                setState(ConnectionState.CONNECTED, null)
            }

            is ServerMessage.Phase -> {
                phase = message.phase
            }

            is ServerMessage.Pong -> {
                lastPongAtMs = timeSource()
                val rtt = (timeSource() - message.clientTime).coerceAtLeast(0L)
                synchronized(rttLock) {
                    lastRttMs = rtt
                    rttSamples.addLast(rtt)
                    while (rttSamples.size > RTT_SAMPLES) rttSamples.removeFirst()
                }
            }

            is ServerMessage.ServerError -> handleServerError(message)

            is ServerMessage.Unknown -> Unit
            else -> Unit
        }
        if (running.get()) listener.onMessage(message)
    }

    private fun handleServerError(message: ServerMessage.ServerError) {
        lastError = "${message.code}: ${message.message}"
        when {
            message.code in FATAL_ERRORS -> {
                fatal = true
                cancelReconnect()
                closeSocket(CLOSE_NORMAL, "fatal: ${message.code}")
                setState(ConnectionState.FATAL, lastError)
            }

            message.code == CODE_GAME_UNAVAILABLE -> {
                // The server keeps controllers connected while the game is
                // down. The following attract phase gates input; reconnecting
                // here would only race the server's slot bookkeeping.
                phase = "attract"
            }

            message.code in RETRYABLE_ERRORS -> {
                closeSocket(CLOSE_NORMAL, "retry: ${message.code}")
                scheduleReconnect(lastError ?: message.code)
            }
        }
    }

    private inner class SocketListener(
        private val socketGeneration: Int,
    ) : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
            if (!running.get() || generation.get() != socketGeneration) {
                webSocket.close(CLOSE_NORMAL, "stale socket")
                return
            }
            cancelReconnect()
            socket = webSocket
            lastPongAtMs = timeSource()
            setState(ConnectionState.CONNECTING, "handshake")
            if (!webSocket.send(Protocol.hello(deviceToken, appVersion))) {
                webSocket.close(CLOSE_GOING_AWAY, "hello send failed")
                scheduleReconnect("hello send failed")
            }
        }

        override fun onMessage(webSocket: WebSocket, text: String) {
            if (!running.get() || generation.get() != socketGeneration) return
            handleMessage(text)
        }

        override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
            if (generation.get() == socketGeneration) {
                webSocket.close(CLOSE_NORMAL, null)
            }
        }

        override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
            if (generation.get() != socketGeneration) return
            if (socket === webSocket) socket = null
            if (!running.get()) return
            scheduleReconnect("connection closed ($code ${reason.trim()})")
        }

        override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
            if (generation.get() != socketGeneration) return
            if (socket === webSocket) socket = null
            if (!running.get()) return
            scheduleReconnect(t.message ?: t.javaClass.simpleName)
        }
    }

    private fun runSafely(task: () -> Unit) {
        try {
            task()
        } catch (e: RuntimeException) {
            lastError = "client task failed: ${e.message ?: e.javaClass.simpleName}"
        }
    }

    companion object {
        /** D-23 allows 20–30 Hz; 25 Hz leaves headroom on the old tablets. */
        const val DEFAULT_INPUT_HZ = 25

        /** After this many failed reconnects the client stops and asks for setup. */
        const val DEFAULT_MAX_RECONNECT_ATTEMPTS = 3

        private const val CLOSE_NORMAL = 1000
        private const val CLOSE_GOING_AWAY = 1001
        private const val RTT_SAMPLES = 10
        private const val PHASE_PLAYING = "playing"
        private const val CODE_GAME_UNAVAILABLE = "game_unavailable"
        private val FATAL_ERRORS = setOf("auth_failed", "kicked", "protocol_mismatch")
        private val RETRYABLE_ERRORS = setOf("session_full", "token_in_use")
    }
}

/**
 * Accepts a base address such as `192.168.1.10:8080`, `ws://192.168.1.10:8080`,
 * or a full endpoint. A base address is expanded to `/ws/controller`.
 * `http(s)` input is accepted for tests and converted back to `ws(s)`.
 */
internal fun controllerEndpoint(serverUrl: String): String {
    val trimmed = serverUrl.trim()
    if (trimmed.isEmpty()) throw IllegalArgumentException("Invalid server URL: $serverUrl")
    val httpUrl = when {
        trimmed.startsWith("ws:", ignoreCase = true) -> "http:" + trimmed.substring(3)
        trimmed.startsWith("wss:", ignoreCase = true) -> "https:" + trimmed.substring(4)
        trimmed.startsWith("http:", ignoreCase = true) -> trimmed
        trimmed.startsWith("https:", ignoreCase = true) -> trimmed
        else -> "http://$trimmed"
    }
    val parsed = HttpUrl.parse(httpUrl)
        ?: throw IllegalArgumentException("Invalid server URL: $serverUrl")
    val path = if (parsed.encodedPath() == "/") CONTROLLER_PATH else parsed.encodedPath()
    val normalized = parsed.newBuilder().encodedPath(path).build().toString()
    val secure = parsed.scheme().equals("https", ignoreCase = true)
    return if (secure) {
        "wss" + normalized.substring("https".length)
    } else {
        "ws" + normalized.substring("http".length)
    }
}

private const val CONTROLLER_PATH = "/ws/controller"
