package pszlbit.drift.controller

/**
 * Exponential backoff policy for controller reconnect attempts.
 *
 * The first retry waits `baseDelayMs`, then doubles per failed attempt until
 * `maxDelayMs` is reached. No jitter is applied so tests stay deterministic;
 * adding jitter later is a one-line change here.
 */
class ReconnectPolicy(
    private val baseDelayMs: Long = DEFAULT_BASE_DELAY_MS,
    private val maxDelayMs: Long = DEFAULT_MAX_DELAY_MS,
    private val factor: Double = DEFAULT_BACKOFF_FACTOR,
) {
    private var failures: Int = 0

    /** Returns the delay before the next reconnect attempt and counts this failure. */
    fun nextDelayMs(): Long {
        failures += 1
        var delayMs = baseDelayMs
        repeat((failures - 1).coerceAtLeast(0)) {
            delayMs = (delayMs * factor).toLong()
        }
        return minOf(delayMs, maxDelayMs)
    }

    /** Resets the failure counter, for example after a successful (re)connect. */
    fun reset() {
        failures = 0
    }

    companion object {
        const val DEFAULT_BASE_DELAY_MS: Long = 1000L
        const val DEFAULT_MAX_DELAY_MS: Long = 30000L
        const val DEFAULT_BACKOFF_FACTOR: Double = 2.0
    }
}
