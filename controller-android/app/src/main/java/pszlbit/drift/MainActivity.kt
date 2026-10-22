package pszlbit.drift

import android.app.Activity
import android.app.AlertDialog
import android.os.Bundle
import android.text.InputType
import android.view.View
import android.view.WindowManager
import android.widget.EditText
import android.widget.TextView
import pszlbit.drift.input.InputPadView
import pszlbit.drift.input.InputState
import pszlbit.drift.net.ClientStats
import pszlbit.drift.net.ConnectionState
import pszlbit.drift.net.ControllerClient
import pszlbit.drift.protocol.ServerMessage
import pszlbit.drift.storage.DeviceTokenStore
import pszlbit.drift.storage.ServerSettings

/**
 * Single controller activity: feedback, reconnect overlay, and the spike touch
 * pad. Networking, protocol, and storage live in plain Kotlin classes; this
 * activity only wires them to platform Views.
 */
class MainActivity : Activity(), ControllerClient.Listener {
    private lateinit var deviceTokens: DeviceTokenStore
    private lateinit var settings: ServerSettings

    private var client: ControllerClient? = null

    @Volatile
    private var input = InputState.NEUTRAL

    private lateinit var statusView: TextView
    private lateinit var phaseView: TextView
    private lateinit var slotView: TextView
    private lateinit var serverView: TextView
    private lateinit var taskView: TextView
    private lateinit var scoreView: TextView
    private lateinit var resultView: TextView
    private lateinit var statsView: TextView
    private lateinit var overlayView: TextView
    private lateinit var inputPad: InputPadView

    private var state = ConnectionState.IDLE
    private var phase = "attract"
    private var timeLeftSeconds = 0L

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        deviceTokens = DeviceTokenStore(this)
        settings = ServerSettings(this)

        statusView = findViewById(R.id.status_text)
        phaseView = findViewById(R.id.phase_text)
        slotView = findViewById(R.id.slot_text)
        serverView = findViewById(R.id.server_text)
        taskView = findViewById(R.id.task_text)
        scoreView = findViewById(R.id.score_text)
        resultView = findViewById(R.id.result_text)
        statsView = findViewById(R.id.stats_text)
        overlayView = findViewById(R.id.overlay_text)
        inputPad = findViewById(R.id.input_pad)

        inputPad.onInputChanged = { input = it }
        serverView.setOnLongClickListener {
            showServerDialog()
            true
        }

        // ADR-0002: keep the screen awake while the tablet is a controller.
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        applyImmersive()
        renderStatus(ConnectionState.IDLE, getString(R.string.status_disconnected))
        updateInputEnabled()
    }

    override fun onStart() {
        super.onStart()
        restartClient()
    }

    override fun onStop() {
        client?.stop()
        client = null
        super.onStop()
    }

    override fun onWindowFocusChanged(hasFocus: Boolean) {
        super.onWindowFocusChanged(hasFocus)
        if (hasFocus) applyImmersive()
    }

    override fun onStateChanged(state: ConnectionState, detail: String?) {
        runOnUiThread {
            this.state = state
            renderStatus(state, detail)
            when (state) {
                ConnectionState.CONNECTING -> showOverlay(R.string.overlay_connecting)
                ConnectionState.RECONNECTING -> showOverlay(R.string.overlay_reconnecting)
                ConnectionState.FATAL -> showOverlay(R.string.overlay_stopped)
                else -> overlayView.visibility = View.GONE
            }
            updateInputEnabled()
        }
    }

    override fun onMessage(message: ServerMessage) {
        runOnUiThread {
            when (message) {
                is ServerMessage.Welcome -> {
                    slotView.text = message.label.ifEmpty { getString(R.string.slot_none) }
                    slotView.setTextColor(resources.getColor(colorForSlot(message.color)))
                    phase = message.phase
                    timeLeftSeconds = 0L
                    renderPhase()
                }

                is ServerMessage.Phase -> {
                    phase = message.phase
                    timeLeftSeconds = message.timeLeftMs / 1_000L
                    renderPhase()
                    updateInputEnabled()
                }

                is ServerMessage.Feedback -> {
                    scoreView.text = getString(R.string.score_format, message.score)
                    val task = message.task
                    taskView.text = if (task == null) {
                        getString(R.string.task_none)
                    } else {
                        getString(
                            R.string.task_format,
                            task.spotLabel,
                            task.spotColor,
                            task.hint,
                        )
                    }
                }

                is ServerMessage.TaskResult -> {
                    scoreView.text = getString(R.string.score_format, message.totalScore)
                    resultView.text = getString(
                        R.string.result_task_format,
                        message.points,
                        message.rating,
                        message.totalScore,
                    )
                }

                is ServerMessage.RoundResult -> {
                    val scores = message.scores.joinToString(separator = "  ") {
                        "P${it.slot}: ${it.score}"
                    }
                    val header = getString(
                        R.string.result_round_format,
                        message.roundId,
                        message.winnerSlot,
                    )
                    resultView.text = if (scores.isEmpty()) header else "$header · $scores"
                }

                is ServerMessage.ServerError -> resultView.text = getString(
                    R.string.result_error_format,
                    message.code,
                    message.message,
                )

                is ServerMessage.Pong -> Unit
                is ServerMessage.Unknown -> Unit
            }
            updateInputEnabled()
        }
    }

    override fun onStats(stats: ClientStats) {
        runOnUiThread {
            statsView.text = getString(
                R.string.stats_format,
                stats.sentTotal,
                stats.measuredHz,
                stats.lastRttMs,
                stats.reconnects,
                stats.sequence,
            )
        }
    }

    private fun restartClient() {
        client?.stop()
        client = null

        val url = settings.serverUrl
        serverView.text = url.ifEmpty { getString(R.string.server_unset) }
        if (url.isEmpty()) {
            renderStatus(ConnectionState.IDLE, getString(R.string.set_server_hint))
            overlayView.visibility = View.GONE
            updateInputEnabled()
            return
        }

        val newClient = ControllerClient(
            serverUrl = url,
            deviceToken = deviceTokens.token(),
            appVersion = appVersion(),
            inputProvider = { input },
            listener = this,
        )
        client = newClient
        newClient.start()
    }

    private fun showServerDialog() {
        val field = EditText(this).apply {
            setText(settings.serverUrl)
            hint = getString(R.string.server_dialog_hint)
            inputType = InputType.TYPE_TEXT_VARIATION_URI
            setSingleLine()
        }
        AlertDialog.Builder(this)
            .setTitle(R.string.server_dialog_title)
            .setView(field)
            .setPositiveButton(android.R.string.ok) { _, _ ->
                val value = field.text.toString().trim()
                if (value.isEmpty() || value.startsWith("ws://") || value.startsWith("wss://")) {
                    settings.serverUrl = value
                    restartClient()
                } else {
                    resultView.text = getString(R.string.server_invalid)
                }
            }
            .setNegativeButton(android.R.string.cancel, null)
            .show()
    }

    private fun renderStatus(state: ConnectionState, detail: String?) {
        val label = when (state) {
            ConnectionState.IDLE -> getString(R.string.status_disconnected)
            ConnectionState.CONNECTING -> getString(R.string.status_connecting)
            ConnectionState.CONNECTED -> getString(R.string.status_connected)
            ConnectionState.RECONNECTING -> getString(R.string.status_reconnecting)
            ConnectionState.FATAL -> getString(R.string.status_stopped)
        }
        statusView.text = if (detail.isNullOrEmpty()) label else "$label — $detail"
    }

    private fun renderPhase() {
        phaseView.text = if (timeLeftSeconds > 0L) {
            getString(R.string.phase_time_format, phase, timeLeftSeconds)
        } else {
            getString(R.string.phase_format, phase)
        }
    }

    private fun updateInputEnabled() {
        val enabled = state == ConnectionState.CONNECTED && phase == "playing"
        inputPad.isEnabled = enabled
        inputPad.alpha = if (enabled) 1f else 0.4f
    }

    private fun showOverlay(textRes: Int) {
        overlayView.setText(textRes)
        overlayView.visibility = View.VISIBLE
    }

    private fun colorForSlot(color: String): Int = when (color) {
        "blue" -> R.color.slot_blue
        "red" -> R.color.slot_red
        "green" -> R.color.slot_green
        "yellow" -> R.color.slot_yellow
        else -> R.color.text_primary
    }

    private fun appVersion(): String = try {
        packageManager.getPackageInfo(packageName, 0).versionName ?: APP_VERSION_FALLBACK
    } catch (e: Exception) {
        APP_VERSION_FALLBACK
    }

    @Suppress("DEPRECATION")
    private fun applyImmersive() {
        window.decorView.systemUiVisibility =
            View.SYSTEM_UI_FLAG_LAYOUT_STABLE or
            View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION or
            View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN or
            View.SYSTEM_UI_FLAG_HIDE_NAVIGATION or
            View.SYSTEM_UI_FLAG_FULLSCREEN or
            View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
    }

    private companion object {
        const val APP_VERSION_FALLBACK = "0.1.0-spike"
    }
}
