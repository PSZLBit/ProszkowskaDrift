package pszlbit.drift.controller

import com.squareup.okhttp3.OkHttpClient
import com.squareup.okhttp3.Request
import com.squareup.okhttp3.Response
import com.squareup.okhttp3.WebSocket
import com.squareup.okhttp3.WebSocketListener
import org.json.JSONObject

/**
 * OkHttp 3 based [Transport] for the controller endpoint.
 *
 * OkHttp 3.12 is the last line that supports Android API 19; OkHttp 4 requires
 * API 21 (ADR-0002, D-20). JSON is encoded/decoded with the platform
 * `org.json` parser.
 *
 * @param endpoint WebSocket endpoint, for example `ws://192.168.1.50:8080/ws/controller`.
 */
class OkHttpTransport(private val endpoint: String) : Transport {

    private val client: OkHttpClient = OkHttpClient()
    private var webSocket: WebSocket? = null
    private var listener: TransportListener? = null

    override fun open(listener: TransportListener) {
        this.listener = listener
        client.newWebSocket(
            Request.Builder().url(endpoint).build(),
            object : WebSocketListener() {
                override fun onOpened(webSocket: WebSocket, response: Response) {
                    this@OkHttpTransport.webSocket = webSocket
                    listener.onOpened()
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    listener?.onEnvelope(decode(text))
                }

                override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                    if (this@OkHttpTransport.webSocket === webSocket) {
                        this@OkHttpTransport.webSocket = null
                    }
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    if (this@OkHttpTransport.webSocket === webSocket) {
                        this@OkHttpTransport.webSocket = null
                    }
                    listener?.onClosed(code, reason)
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    if (this@OkHttpTransport.webSocket === webSocket) {
                        this@OkHttpTransport.webSocket = null
                    }
                    listener?.onClosed(1000, t.message ?: "connection failed")
                }
            },
        )
    }

    override fun send(envelope: Envelope) {
        val webSocket = this.webSocket ?: return
        val body = JSONObject()
            .put("v", envelope.v)
            .put("type", envelope.type)
        if (envelope.seq != null) {
            body.put("seq", envelope.seq)
        }
        body.put("data", encodeData(envelope))
        webSocket.send(body.toString(), { _, _ -> Unit })
    }

    override fun close() {
        val webSocket = this.webSocket ?: return
        this.webSocket = null
        webSocket.close(1000, "closing")
        listener = null
    }

    private fun decode(text: String): Envelope {
        val json = JSONObject(text)
        val type = json.optString("type")
        val seq = if (json.has("seq") && !json.isNull("seq")) json.optInt("seq") else null
        val data = when (type) {
            TYPE_WELCOME -> WelcomeData(
                slot = json.optInt("slot"),
                label = json.optString("label"),
                color = json.optString("color"),
                phase = json.optString("phase"),
            )
            TYPE_PHASE -> PhaseData(
                phase = json.optString("phase"),
                timeLeftMs = json.optLong("timeLeftMs"),
                roundId = json.optInt("roundId"),
            )
            TYPE_ERROR -> ErrorData(code = json.optString("code"), message = json.optString("message"))
            TYPE_TASK -> TaskData(
                taskId = json.optInt("taskId"),
                spotLabel = json.optString("spotLabel"),
                spotColor = json.optString("spotColor"),
                hint = json.optString("hint"),
                deadlineMs = json.optLong("deadlineMs"),
            )
            TYPE_TASK_RESULT -> TaskResultData(
                taskId = json.optInt("taskId"),
                points = json.optInt("points"),
                timeScore = json.optInt("timeScore"),
                precisionScore = json.optInt("precisionScore"),
                collisionPenalty = json.optInt("collisionPenalty"),
                rating = json.optString("rating"),
                totalScore = json.optInt("totalScore"),
            )
            TYPE_ROUND_RESULT -> RoundResultData(
                roundId = json.optInt("roundId"),
                winnerSlot = json.optInt("winnerSlot"),
                scores = emptyList(),
            )
            TYPE_PONG -> PongData(clientTime = json.optLong("clientTime"), serverTime = json.optLong("serverTime"))
            else -> JSONObject()
        }
        return Envelope(v = json.optInt("v"), type = type, seq = seq, data = data)
    }

    private fun encodeData(envelope: Envelope): JSONObject {
        val data = envelope.data
        return when (envelope.type) {
            TYPE_HELLO -> JSONObject()
                .put("deviceToken", (data as HelloData).deviceToken)
                .put("appVersion", (data as HelloData).appVersion)
            TYPE_INPUT -> JSONObject()
                .put("seq", (data as InputData).seq)
                .put("steer", (data as InputData).steer)
                .put("throttle", (data as InputData).throttle)
                .put("ts", (data as InputData).ts)
            TYPE_PING -> JSONObject().put("clientTime", (data as PingData).clientTime)
            TYPE_LEAVE -> JSONObject()
            else -> JSONObject()
        }
    }
}
