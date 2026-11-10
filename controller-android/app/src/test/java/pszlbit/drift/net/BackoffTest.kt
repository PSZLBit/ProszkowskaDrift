package pszlbit.drift.net

import org.junit.Assert.assertEquals
import org.junit.Test

class BackoffTest {

    @Test
    fun firstAttemptWaitsForBaseDelay() {
        assertEquals(500L, Backoff.delayMs(attempt = 0))
        assertEquals(500L, Backoff.delayMs(attempt = -1))
    }

    @Test
    fun delayDoublesAndStopsAtTheCap() {
        assertEquals(1_000L, Backoff.delayMs(attempt = 1))
        assertEquals(2_000L, Backoff.delayMs(attempt = 2))
        assertEquals(4_000L, Backoff.delayMs(attempt = 3))
        assertEquals(8_000L, Backoff.delayMs(attempt = 4))
        assertEquals(10_000L, Backoff.delayMs(attempt = 5))
        assertEquals(10_000L, Backoff.delayMs(attempt = 100))
    }

    @Test
    fun customBaseAndCapAreRespected() {
        assertEquals(20L, Backoff.delayMs(attempt = 0, baseMs = 20L, capMs = 60L))
        assertEquals(40L, Backoff.delayMs(attempt = 1, baseMs = 20L, capMs = 60L))
        assertEquals(60L, Backoff.delayMs(attempt = 2, baseMs = 20L, capMs = 60L))
    }
}
