package pszlbit.drift.net

import java.util.concurrent.CopyOnWriteArrayList
import pszlbit.drift.protocol.ServerMessage

/** Thread-safe listener used by the client tests; callbacks arrive off-thread. */
class RecordingControllerListener : ControllerClient.Listener {
    val states = CopyOnWriteArrayList<ConnectionState>()
    val messages = CopyOnWriteArrayList<ServerMessage>()
    val errors = CopyOnWriteArrayList<String>()

    @Volatile
    var state: ConnectionState = ConnectionState.IDLE
        private set

    @Volatile
    var welcome: ServerMessage.Welcome? = null
        private set

    @Volatile
    var lastStats: ClientStats? = null
        private set

    override fun onStateChanged(state: ConnectionState, detail: String?) {
        this.state = state
        states.add(state)
    }

    override fun onMessage(message: ServerMessage) {
        messages.add(message)
        when (message) {
            is ServerMessage.Welcome -> welcome = message
            is ServerMessage.ServerError -> errors.add(message.code)
            else -> Unit
        }
    }

    override fun onStats(stats: ClientStats) {
        lastStats = stats
    }
}
