package pszlbit.drift.controller

class FakeControllerListener : ControllerListener {
    val states: MutableList<ConnectionState> = mutableListOf()
    var welcomeSlot: Int? = null
    var welcomePhase: String? = null
    val phaseUpdates: MutableList<String> = mutableListOf()
    val errors: MutableList<String> = mutableListOf()

    override fun onStateChange(state: ConnectionState) {
        states.add(state)
    }

    override fun onWelcome(slot: Int, label: String, color: String, phase: String) {
        welcomeSlot = slot
        welcomePhase = phase
    }

    override fun onPhaseUpdate(phase: String, timeLeftMs: Long, roundId: Int) {
        phaseUpdates.add(phase)
    }

    override fun onError(code: String, message: String) {
        errors.add(code)
    }
}
