package pszlbit.drift.net

/**
 * Deterministic scheduler for host-JVM tests.
 *
 * Time only moves when a test calls [advanceBy]. Periodic tasks use fixed-delay
 * semantics, matching [ControllerClient]'s production behavior.
 */
internal class ManualScheduler : TaskScheduler {
    private val lock = Any()
    private val tasks = mutableListOf<ManualTask>()
    private var currentMs = 0L

    /** Shared monotonic clock; pass this to the client's `timeSource`. */
    val nowMs: Long
        get() = synchronized(lock) { currentMs }

    override fun schedule(delayMs: Long, task: () -> Unit): ScheduledTask =
        add(delayMs, intervalMs = 0L, task = task)

    override fun scheduleWithFixedDelay(
        initialDelayMs: Long,
        delayMs: Long,
        task: () -> Unit,
    ): ScheduledTask = add(initialDelayMs, maxOf(1L, delayMs), task)

    override fun shutdownNow() {
        synchronized(lock) { tasks.clear() }
    }

    /** Runs every task due up to `now + deltaMs`, in due-time order. */
    fun advanceBy(deltaMs: Long) {
        val target = nowMs + deltaMs
        while (true) {
            val next = synchronized(lock) {
                tasks.removeAll { it.cancelled }
                val candidate = tasks.filter { it.dueMs <= target }.minByOrNull { it.dueMs }
                if (candidate != null) {
                    currentMs = maxOf(currentMs, candidate.dueMs)
                    if (candidate.intervalMs == 0L) tasks.remove(candidate)
                }
                candidate
            } ?: break
            next.task()
            if (next.intervalMs > 0L && !next.cancelled) {
                synchronized(lock) {
                    next.dueMs = currentMs + next.intervalMs
                    tasks.add(next)
                }
            }
        }
        synchronized(lock) { currentMs = maxOf(currentMs, target) }
    }

    private fun add(
        delayMs: Long,
        intervalMs: Long,
        task: () -> Unit,
    ): ScheduledTask {
        val manualTask = ManualTask(
            dueMs = nowMs + maxOf(0L, delayMs),
            intervalMs = intervalMs,
            task = task,
        )
        synchronized(lock) { tasks.add(manualTask) }
        return manualTask
    }

    private inner class ManualTask(
        var dueMs: Long,
        val intervalMs: Long,
        val task: () -> Unit,
    ) : ScheduledTask {
        @Volatile
        var cancelled = false
            private set

        override fun cancel() {
            cancelled = true
        }
    }
}
