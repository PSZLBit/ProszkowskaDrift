package pszlbit.drift.controller

import org.junit.Assert.assertEquals
import org.junit.Test

class ReconnectPolicyTest {

    @Test
    fun firstRetryWaitsForBaseDelay() {
        assertEquals(ReconnectPolicy.DEFAULT_BASE_DELAY_MS, ReconnectPolicy().nextDelayMs())
    }

    @Test
    fun retryDelayDoublesUntilCap() {
        val policy = ReconnectPolicy(baseDelayMs = 10L, maxDelayMs = 100L, factor = 2.0)
        assertEquals(10L, policy.nextDelayMs())
        assertEquals(20L, policy.nextDelayMs())
        assertEquals(40L, policy.nextDelayMs())
        assertEquals(80L, policy.nextDelayMs())
        assertEquals(100L, policy.nextDelayMs())
    }

    @Test
    fun resetClearsFailureCount() {
        val policy = ReconnectPolicy(baseDelayMs = 10L, maxDelayMs = 100L)
        repeat(3) { policy.nextDelayMs() }
        policy.reset()
        assertEquals(10L, policy.nextDelayMs())
    }
}
