package pszlbit.drift.controller

import java.util.concurrent.Executors
import java.util.concurrent.ScheduledExecutorService
import java.util.concurrent.ScheduledFuture
import java.util.concurrent.TimeUnit

/**
 * WebSocket controller client for the Go server (protocol v1).
 *
 * Responsibilities in the spike scope:
 * - open the transport to the controller endpoint,
 * - authenticate with the device token via `hello` / `welcome` / `error`,
 * - send `input` envelopes at a fixed rate while the phase is `playing`,
 * - reconnect with exponential backoff after an unexpected close.
 *
 * The client holds no Android dependencies so the connection logic can be unit
 * tested against a fake transport. The UI layer only observes
 * [ControllerListener]; input samples are produced by the touch controls.
 */
class ControllerClient(
    private val transport: Transport,
    private val deviceToken: String,
    private val listener: ControllerListener,
    private val appVersion: String = DEFAULT_APP_VERSION,
    private val inputIntervalMs: Long = DEFAULT_INPUT_INTERVAL_MS,
    private val backoffPolicy: ReconnectPolicy = ReconnectPolicy(),
    private val executor: ScheduledExecutorService = Executors.newSingleThreadScheduledExecutor(),
    private val nowMillis: () -> Long = { System.currentTimeMillis() },
) : AutoCloseable {

    var state: ConnectionState = ConnectionState.DISCONNECTED
        private set

    val slot: Int? get() = assignedSlot

    val phase: String? get() = currentPhase

    val currentInput: InputSample? get() = currentSample

    fun start() {
        state = ConnectionState.CONNECTING
        assignedSlot = null
        currentPhase = null
        currentSample = null
        nextSeq = 1
        stopInputLoop()
        transport.open(transportListener)
    }

    override fun close() {
        stopInputLoop()
        reconnectTask?.cancel(false)
        reconnectTask = null
        state = ConnectionState.DISCONNECTED
        transport.close()
        executor.shutdownNow()
    }

    /** Stores the latest steering/throttle sample produced by the controls. */
    fun updateInput(steer: Double, throttle: Double) {
        currentSample = InputSample(steer = steer, throttle = throttle)
    }

    private fun sendHello() {
        transport.send(Envelope.create(TYPE_HELLO, null, HelloData(deviceToken = deviceToken, appVersion = appVersion)))
    }

    private fun startInputLoop() {
        stopInputLoop()
        inputTask = executor.scheduleWithFixedDelay(
            { sendCurrentInput() },
            inputIntervalMs,
            inputIntervalMs,
            TimeUnit.MILLISECONDS,
        )
    }

    private fun stopInputLoop() {
        inputTask?.cancel(false)
        inputTask = null
    }

    private fun sendCurrentInput() {
        val sample = currentSample ?: InputSample(steer = 0.0, throttle = 0.0)
        val seq = nextSeq
        nextSeq += 1
        transport.send(
            Envelope(
                v = PROTOCOL_VERSION,
                type = TYPE_INPUT,
                seq = seq,
                data = InputData(seq = seq, steer = sample.steer, throttle = sample.throttle, ts = nowMillis()),
            ),
        )
    }

    private fun handleEnvelope(envelope: Envelope) {
        when (envelope.type) {
            TYPE_WELCOME -> {
                val data = envelope.data as? WelcomeData
                if (data == null) {
                    handleErrorEnvelope(ErrorData(CODE_INTERNAL_ERROR, "Malformed welcome message."))
                    return
                }
                assignedSlot = data.slot
                currentPhase = data.phase
                state = ConnectionState.CONNECTED
                listener.onWelcome(data.slot, data.label, data.color, data.phase)
                if (data.phase == PHASE_PLAYING) {
                    startInputLoop()
                }
            }
            TYPE_PHASE -> {
                val data = envelope.data as? PhaseData
                if (data == null) {
                    handleErrorEnvelope(ErrorData(CODE_INTERNAL_ERROR, "Malformed phase message."))
                    return
                }
                currentPhase = data.phase
                listener.onPhaseUpdate(data.phase, data.timeLeftMs, data.roundId)
                if (data.phase == PHASE_PLAYING) {
                    startInputLoop()
                } else {
                    stopInputLoop()
                }
            }
            TYPE_ERROR -> handleErrorEnvelope(envelope.data as? ErrorData ?: ErrorData(CODE_INTERNAL_ERROR, "Malformed error message."))
            else -> Unit
        }
    }

    private fun handleErrorEnvelope(data: ErrorData?) {
        if (data == null) return
        listener.onError(data.code, data.message)
        if (data.code in TERMINAL_ERROR_CODES) {
            stopInputLoop()
            state = ConnectionState.FAILED
        }
    }

    private fun handleClosed(code: Int, reason: String) {
        stopInputLoop()
        if (state == ConnectionState.CONNECTED || state == ConnectionState.CONNECTING) {
            state = ConnectionState.RECONNECTING
            val delayMs = backoffPolicy.nextDelayMs()
            reconnectTask = executor.schedule({ start() }, delayMs, TimeUnit.MILLISECONDS)
        }
    }

    private val transportListener =
        object : TransportListener {
            override fun onOpened() {
                sendHello()
            }

            override fun onEnvelope(envelope: Envelope) = handleEnvelope(envelope)

            override fun onClosed(code: Int, reason: String) = handleClosed(code, reason)
        }

    private var assignedSlot: Int? = null

    private var currentPhase: String? = null

    private var currentSample: InputSample? = null

    private var nextSeq: Int = 1

    private var inputTask: ScheduledFuture<*>? = null

    private var reconnectTask: ScheduledFuture<*>? = null

    companion object {
        const val DEFAULT_APP_VERSION: String = "0.1.0"
        const val DEFAULT_INPUT_INTERVAL_MS: Long = 40L
        const val CODE_INTERNAL_ERROR: String = "internal_error"
    }
}
