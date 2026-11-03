package pszlbit.drift

import android.app.Activity
import android.content.SharedPreferences
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.view.WindowManager
import android.widget.TextView
import pszlbit.drift.controller.ControllerClient
import pszlbit.drift.controller.ControllerListener
import pszlbit.drift.controller.ConnectionState
import pszlbit.drift.controller.OkHttpTransport
import pszlbit.drift.controller.PHASE_ATTRACT
import pszlbit.drift.controller.PHASE_COUNTDOWN
import pszlbit.drift.controller.PHASE_ENDED
import pszlbit.drift.controller.PHASE_PLAYING
import pszlbit.drift.controller.PHASE_PAUSED
import pszlbit.drift.controller.PHASE_READY
import pszlbit.drift.controller.PHASE_RESULTS

/**
 * Single controller activity.
 *
 * D-20 / ADR-0002: the app must run on the team's Android 4.4.2 (API 19)
 * tablets, so it uses plain platform Views only. No Compose, no AppCompat,
 * no AndroidX. Do not add a library here unless it is proven to support
 * API 19.
 *
 * The activity is a thin UI adapter around [ControllerClient]; all connection
 * state handling lives in the client and is covered by unit tests with a fake
 * transport.
 */
class MainActivity : Activity(), ControllerListener {

    private lateinit var client: ControllerClient
    private lateinit var statusText: TextView
    private val handler = Handler(Looper.getMainLooper())

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        // ADR-0002: keep the screen awake while the tablet is a controller.
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        statusText = findViewById(R.id.status_text)
        startController()
    }

    override fun onDestroy() {
        client.close()
        super.onDestroy()
    }

    private fun startController() {
        val prefs = getSharedPreferences(PREFS_NAME, MODE_PRIVATE)
        val deviceToken =
            prefs.getString(KEY_DEVICE_TOKEN, null)
                ?: java.util.UUID.randomUUID().toString().also {
                    prefs.edit().putString(KEY_DEVICE_TOKEN, it).apply()
                }
        client =
            ControllerClient(
                transport = OkHttpTransport(CONTROLLER_ENDPOINT),
                deviceToken = deviceToken,
                listener = this,
            )
        client.start()
    }

    override fun onStateChange(state: ConnectionState) {
        handler.post { statusText.text = stateStatusRes(state) }
    }

    override fun onWelcome(slot: Int, label: String, color: String, phase: String) {
        handler.post { statusText.text = R.string.status_connected }
    }

    override fun onPhaseUpdate(phase: String, timeLeftMs: Long, roundId: Int) {
        handler.post { statusText.text = phaseStatusRes(phase) }
    }

    override fun onError(code: String, message: String) {
        handler.post { statusText.text = R.string.status_failed }
    }

    private fun stateStatusRes(state: ConnectionState): Int =
        when (state) {
            ConnectionState.DISCONNECTED -> R.string.status_disconnected
            ConnectionState.CONNECTING -> R.string.status_connecting
            ConnectionState.CONNECTED -> R.string.status_connected
            ConnectionState.RECONNECTING -> R.string.status_reconnecting
            ConnectionState.FAILED -> R.string.status_failed
        }

    private fun phaseStatusRes(phase: String): Int =
        when (phase) {
            PHASE_PLAYING -> R.string.status_playing
            PHASE_PAUSED -> R.string.status_paused
            PHASE_ATTRACT -> R.string.status_attract
            PHASE_READY -> R.string.status_ready
            PHASE_COUNTDOWN -> R.string.status_countdown
            PHASE_RESULTS -> R.string.status_results
            PHASE_ENDED -> R.string.status_ended
            else -> R.string.status_unknown
        }

    companion object {
        private const val PREFS_NAME = "controller"
        private const val KEY_DEVICE_TOKEN = "device_token"
        private const val CONTROLLER_ENDPOINT = "ws://192.168.1.100:8080/ws/controller"
    }
}
