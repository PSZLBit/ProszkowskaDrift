package pszlbit.drift.net

/**
 * Exponential reconnect delay in milliseconds, capped, starting at [baseMs].
 *
 * Pure Kotlin and unit-tested: reconnect behaviour must stay deterministic.
 */
object Backoff {
    fun delayMs(attempt: Int, baseMs: Long = 500L, capMs: Long = 10_000L): Long {
        if (attempt <= 0) return baseMs
        var delay = baseMs
        repeat(minOf(attempt, 16)) {
            delay = minOf(delay * 2, capMs)
        }
        return delay
    }
}
