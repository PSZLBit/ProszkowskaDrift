package pszlbit.drift.input

import org.junit.Assert.assertEquals
import org.junit.Test

class InputTrackerTest {

    @Test
    fun startsNeutral() {
        assertEquals(InputState.NEUTRAL, InputTracker().state)
    }

    @Test
    fun oppositeZonesCancelOut() {
        val tracker = InputTracker()
        tracker.setPointer(0, Zone.LEFT)
        tracker.setPointer(1, Zone.RIGHT)
        tracker.setPointer(2, Zone.FORWARD)
        tracker.setPointer(3, Zone.REVERSE)

        assertEquals(InputState(steer = 0, throttle = 0), tracker.state)
    }

    @Test
    fun twoPointersCanSteerAndDriveAtOnce() {
        val tracker = InputTracker()
        tracker.setPointer(0, Zone.LEFT)
        tracker.setPointer(1, Zone.FORWARD)

        assertEquals(InputState(steer = -1, throttle = 1), tracker.state)
    }

    @Test
    fun movingAPointerOutsideEveryZoneReleasesIt() {
        val tracker = InputTracker()
        tracker.setPointer(0, Zone.RIGHT)
        tracker.setPointer(0, null)

        assertEquals(InputState.NEUTRAL, tracker.state)
    }

    @Test
    fun releaseAndClearRemovePressedZones() {
        val tracker = InputTracker()
        tracker.setPointer(0, Zone.RIGHT)
        tracker.setPointer(1, Zone.REVERSE)
        tracker.releasePointer(0)
        assertEquals(InputState(steer = 0, throttle = -1), tracker.state)

        tracker.clear()
        assertEquals(InputState.NEUTRAL, tracker.state)
    }
}
