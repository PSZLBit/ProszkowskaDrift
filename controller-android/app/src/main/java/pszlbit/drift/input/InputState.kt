package pszlbit.drift.input

/** Touch zones of the placeholder spike scheme (D-14 is still TBD). */
enum class Zone {
    LEFT,
    RIGHT,
    FORWARD,
    REVERSE,
}

/**
 * Normalised controller input, `-1..1` per axis.
 *
 * Values are integers because the spike scheme is digital (a zone is pressed or
 * not). The final scheme must be chosen by playtesting (D-14), so this class is
 * deliberately small and easy to replace.
 */
data class InputState(val steer: Int, val throttle: Int) {
    companion object {
        val NEUTRAL = InputState(steer = 0, throttle = 0)
    }
}

/**
 * Tracks one zone per pointer so two thumbs can steer and drive at once.
 *
 * Pure Kotlin, unit-tested on the JVM.
 */
class InputTracker {
    private val pressedZones = HashMap<Int, Zone>()

    /** Binds a pointer to a zone, or drops it when it is outside every zone. */
    fun setPointer(pointerId: Int, zone: Zone?) {
        if (zone == null) {
            pressedZones.remove(pointerId)
        } else {
            pressedZones[pointerId] = zone
        }
    }

    fun releasePointer(pointerId: Int) {
        pressedZones.remove(pointerId)
    }

    fun clear() {
        pressedZones.clear()
    }

    fun isPressed(zone: Zone): Boolean = pressedZones.values.any { it == zone }

    val state: InputState
        get() {
            val zones = pressedZones.values.toSet()
            return InputState(
                steer = (if (Zone.RIGHT in zones) 1 else 0) -
                    (if (Zone.LEFT in zones) 1 else 0),
                throttle = (if (Zone.FORWARD in zones) 1 else 0) -
                    (if (Zone.REVERSE in zones) 1 else 0),
            )
        }
}
