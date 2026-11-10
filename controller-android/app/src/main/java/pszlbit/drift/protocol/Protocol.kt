package pszlbit.drift.protocol

import org.json.JSONException
import org.json.JSONObject

/** Thrown when a frame does not match the version 1 protocol contract. */
class ProtocolException(message: String, cause: Throwable? = null) : Exception(message, cause)

/** One task shown on the controller. No map geometry is sent (FR-C06). */
data class Task(
    val taskId: Long,
    val spotLabel: String,
    val spotColor: String,
    val hint: String,
    val deadlineMs: Long,
)

data class RoundScore(
    val slot: Int,
    val score: Long,
    val tasks: Int,
    val collisions: Int,
)

/** Parsed server-to-controller messages from protocol.md. */
sealed class ServerMessage {
    data class Welcome(
        val slot: Int,
        val label: String,
        val color: String,
        val phase: String,
    ) : ServerMessage()

    data class Phase(
        val phase: String,
        val timeLeftMs: Long,
        val roundId: Long,
    ) : ServerMessage()

    data class Feedback(
        val task: Task?,
        val score: Long,
        val state: String,
    ) : ServerMessage()

    data class TaskResult(
        val taskId: Long,
        val points: Long,
        val timeScore: Long,
        val precisionScore: Long,
        val collisionPenalty: Long,
        val rating: String,
        val totalScore: Long,
    ) : ServerMessage()

    data class RoundResult(
        val roundId: Long,
        val winnerSlot: Int,
        val scores: List<RoundScore>,
    ) : ServerMessage()

    data class ServerError(
        val code: String,
        val message: String,
    ) : ServerMessage()

    data class Pong(
        val clientTime: Long,
        val serverTime: Long,
    ) : ServerMessage()

    /** Forward-compatible drop: additive message types are ignored. */
    data class Unknown(val type: String) : ServerMessage()
}

/**
 * Version 1 JSON protocol (documentation/06-development/protocol.md).
 *
 * Pure Kotlin: it has no Android dependencies so the encoder and parser can be
 * unit-tested on the JVM.
 */
object Protocol {
    const val VERSION = 1

    fun hello(deviceToken: String, appVersion: String): String =
        JSONObject()
            .put(FIELD_VERSION, VERSION)
            .put(FIELD_TYPE, "hello")
            .put(
                FIELD_DATA,
                JSONObject()
                    .put("deviceToken", deviceToken)
                    .put("appVersion", appVersion),
            )
            .toString()

    fun input(seq: Long, steer: Double, throttle: Double, ts: Long): String =
        JSONObject()
            .put(FIELD_VERSION, VERSION)
            .put(FIELD_TYPE, "input")
            .put(FIELD_SEQ, seq)
            .put(
                FIELD_DATA,
                JSONObject()
                    .put("steer", steer)
                    .put("throttle", throttle)
                    .put("ts", ts),
            )
            .toString()

    fun ping(clientTime: Long): String =
        JSONObject()
            .put(FIELD_VERSION, VERSION)
            .put(FIELD_TYPE, "ping")
            .put(FIELD_DATA, JSONObject().put("clientTime", clientTime))
            .toString()

    fun leave(): String =
        JSONObject()
            .put(FIELD_VERSION, VERSION)
            .put(FIELD_TYPE, "leave")
            .put(FIELD_DATA, JSONObject())
            .toString()

    fun parse(text: String): ServerMessage {
        val root = try {
            JSONObject(text)
        } catch (e: JSONException) {
            throw ProtocolException("Malformed JSON frame.", e)
        }
        val version = root.optInt(FIELD_VERSION, -1)
        if (version != VERSION) {
            throw ProtocolException("Unsupported protocol version: $version.")
        }
        val type = root.optString(FIELD_TYPE)
        if (type.isEmpty()) {
            throw ProtocolException("Frame has no type.")
        }
        val data = root.optJSONObject(FIELD_DATA) ?: JSONObject()
        return when (type) {
            "welcome" -> ServerMessage.Welcome(
                slot = data.optInt("slot"),
                label = data.optString("label"),
                color = data.optString("color"),
                phase = data.optString("phase", "attract"),
            )

            "phase" -> ServerMessage.Phase(
                phase = data.optString("phase", "attract"),
                timeLeftMs = data.optLong("timeLeftMs"),
                roundId = data.optLong("roundId"),
            )

            // 2026-10 protocol: per-player feedback. Accept the older `task`
            // message too so a stale server build still shows the task.
            "player_feedback" -> ServerMessage.Feedback(
                task = parseTask(data.optJSONObject("task")),
                score = data.optLong("score"),
                state = data.optString("state", ""),
            )

            "task" -> ServerMessage.Feedback(
                task = parseTask(data),
                score = 0,
                state = "",
            )

            "task_result" -> ServerMessage.TaskResult(
                taskId = data.optLong("taskId"),
                points = data.optLong("points"),
                timeScore = data.optLong("timeScore"),
                precisionScore = data.optLong("precisionScore"),
                collisionPenalty = data.optLong("collisionPenalty"),
                rating = data.optString("rating"),
                totalScore = data.optLong("totalScore"),
            )

            "round_result" -> ServerMessage.RoundResult(
                roundId = data.optLong("roundId"),
                winnerSlot = data.optInt("winnerSlot"),
                scores = parseScores(data),
            )

            "error" -> ServerMessage.ServerError(
                code = data.optString("code", "internal_error"),
                message = data.optString("message"),
            )

            "pong" -> ServerMessage.Pong(
                clientTime = data.optLong("clientTime"),
                serverTime = data.optLong("serverTime"),
            )

            else -> ServerMessage.Unknown(type)
        }
    }

    private fun parseTask(task: JSONObject?): Task? {
        if (task == null) return null
        return Task(
            taskId = task.optLong("taskId"),
            spotLabel = task.optString("spotLabel"),
            spotColor = task.optString("spotColor"),
            hint = task.optString("hint"),
            deadlineMs = task.optLong("deadlineMs"),
        )
    }

    private fun parseScores(data: JSONObject): List<RoundScore> {
        val array = data.optJSONArray("scores") ?: return emptyList()
        val result = ArrayList<RoundScore>(array.length())
        for (index in 0 until array.length()) {
            val score = array.optJSONObject(index) ?: continue
            result.add(
                RoundScore(
                    slot = score.optInt("slot"),
                    score = score.optLong("score"),
                    tasks = score.optInt("tasks"),
                    collisions = score.optInt("collisions"),
                ),
            )
        }
        return result
    }

    private const val FIELD_VERSION = "v"
    private const val FIELD_TYPE = "type"
    private const val FIELD_SEQ = "seq"
    private const val FIELD_DATA = "data"
}
