package pszlbit.drift.controller

/**
 * Protocol version implemented by the controller and expected by the Go server.
 * See `06-development/protocol.md`.
 */
const val PROTOCOL_VERSION: Int = 1

const val TYPE_HELLO: String = "hello"
const val TYPE_INPUT: String = "input"
const val TYPE_PING: String = "ping"
const val TYPE_PONG: String = "pong"
const val TYPE_LEAVE: String = "leave"
const val TYPE_WELCOME: String = "welcome"
const val TYPE_PHASE: String = "phase"
const val TYPE_TASK: String = "task"
const val TYPE_TASK_RESULT: String = "task_result"
const val TYPE_ROUND_RESULT: String = "round_result"
const val TYPE_ERROR: String = "error"

/** Phase names used by the server in `phase` messages. */
const val PHASE_ATTRACT: String = "attract"
const val PHASE_READY: String = "ready"
const val PHASE_COUNTDOWN: String = "countdown"
const val PHASE_PLAYING: String = "playing"
const val PHASE_PAUSED: String = "paused"
const val PHASE_RESULTS: String = "results"
const val PHASE_ENDED: String = "ended"

/**
 * Error codes that end the session from the controller's point of view.
 * Transient codes (rate limits, protocol issues) stay non-terminal and are
 * handled by the server or a reconnect.
 */
val TERMINAL_ERROR_CODES: Set<String> = setOf(
    "auth_failed",
    "session_full",
    "game_unavailable",
    "kicked",
)

/** Envelope wrapper for every protocol message. Unknown fields are ignored by design. */
class Envelope(val v: Int, val type: String, val seq: Int?, val data: Any?) {
    companion object {
        fun create(type: String, seq: Int?, data: Any?): Envelope = Envelope(PROTOCOL_VERSION, type, seq, data)
    }
}

/** `hello` payload: device token and app version. */
data class HelloData(val deviceToken: String, val appVersion: String)

/** `input` payload: steering and throttle of the current player sample. */
data class InputData(
    val seq: Int,
    val steer: Double,
    val throttle: Double,
    val ts: Long,
)

/** One steering/throttle sample produced by the player controls. */
data class InputSample(
    val steer: Double,
    val throttle: Double,
)

/** `ping` payload. */
data class PingData(val clientTime: Long)

/** `pong` payload. */
data class PongData(val clientTime: Long, val serverTime: Long)

/** `leave` payload. */
data class LeaveData(val reason: String? = null)

/** `welcome` payload: slot assignment after a successful device token auth. */
data class WelcomeData(
    val slot: Int,
    val label: String,
    val color: String,
    val phase: String,
)

/** `phase` payload: authoritative round phase broadcast by the server. */
data class PhaseData(
    val phase: String,
    val timeLeftMs: Long,
    val roundId: Int,
)

/** `task` payload: assigned parking task for the slot. */
data class TaskData(
    val taskId: Int,
    val spotLabel: String,
    val spotColor: String,
    val hint: String,
    val deadlineMs: Long,
)

/** `task_result` payload. */
data class TaskResultData(
    val taskId: Int,
    val points: Int,
    val timeScore: Int,
    val precisionScore: Int,
    val collisionPenalty: Int,
    val rating: String,
    val totalScore: Int,
)

/** `round_result` payload. */
data class RoundResultData(
    val roundId: Int,
    val winnerSlot: Int,
    val scores: List<ScoreEntry>,
)

/** One score entry in a round result. */
data class ScoreEntry(
    val slot: Int,
    val score: Int,
    val tasks: Int,
    val collisions: Int,
)

/** `error` payload. */
data class ErrorData(
    val code: String,
    val message: String,
)

/** Connection lifecycle states of the controller client. */
enum class ConnectionState {
    DISCONNECTED,
    CONNECTING,
    CONNECTED,
    RECONNECTING,
    FAILED,
}

/** Callback interface implemented by the controller UI layer. */
interface ControllerListener {
    fun onStateChange(state: ConnectionState)
    fun onWelcome(slot: Int, label: String, color: String, phase: String)
    fun onPhaseUpdate(phase: String, timeLeftMs: Long, roundId: Int)
    fun onError(code: String, message: String)
}
