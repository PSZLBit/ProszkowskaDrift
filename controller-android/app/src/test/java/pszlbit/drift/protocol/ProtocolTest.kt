package pszlbit.drift.protocol

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ProtocolTest {

    @Test
    fun helloCarriesVersionDeviceTokenAndAppVersion() {
        val json = JSONObject(Protocol.hello("device-token-1", "0.1.0-test"))

        assertEquals(1, json.getInt("v"))
        assertEquals("hello", json.getString("type"))
        val data = json.getJSONObject("data")
        assertEquals("device-token-1", data.getString("deviceToken"))
        assertEquals("0.1.0-test", data.getString("appVersion"))
    }

    @Test
    fun inputCarriesSequenceAxesAndClientTime() {
        val json = JSONObject(Protocol.input(seq = 42L, steer = -0.35, throttle = 0.8, ts = 1234L))

        assertEquals(1, json.getInt("v"))
        assertEquals("input", json.getString("type"))
        assertEquals(42L, json.getLong("seq"))
        val data = json.getJSONObject("data")
        assertEquals(-0.35, data.getDouble("steer"), 1e-9)
        assertEquals(0.8, data.getDouble("throttle"), 1e-9)
        assertEquals(1234L, data.getLong("ts"))
    }

    @Test
    fun pingAndLeaveMatchTheProtocol() {
        val ping = JSONObject(Protocol.ping(9876L))
        assertEquals("ping", ping.getString("type"))
        assertEquals(9876L, ping.getJSONObject("data").getLong("clientTime"))

        val leave = JSONObject(Protocol.leave())
        assertEquals("leave", leave.getString("type"))
        assertEquals(0, leave.getJSONObject("data").length())
    }

    @Test
    fun parsesWelcomeAndPhase() {
        val welcome = Protocol.parse(
            """{"v":1,"type":"welcome","data":{"slot":2,"label":"P2","color":"red","phase":"ready"}}""",
        ) as ServerMessage.Welcome
        assertEquals(2, welcome.slot)
        assertEquals("P2", welcome.label)
        assertEquals("red", welcome.color)
        assertEquals("ready", welcome.phase)

        val phase = Protocol.parse(
            """{"v":1,"type":"phase","data":{"phase":"playing","timeLeftMs":154200,"roundId":7}}""",
        ) as ServerMessage.Phase
        assertEquals("playing", phase.phase)
        assertEquals(154200L, phase.timeLeftMs)
        assertEquals(7L, phase.roundId)
    }

    @Test
    fun parsesPlayerFeedbackWithTask() {
        val feedback = Protocol.parse(
            """{"v":1,"type":"player_feedback","data":{"task":{"taskId":12,""" +
                """"spotLabel":"B-07","spotColor":"red","hint":"north","deadlineMs":30000},""" +
                """"score":240,"state":"playing"}}""",
        ) as ServerMessage.Feedback

        assertEquals(240L, feedback.score)
        assertEquals("playing", feedback.state)
        assertEquals("B-07", feedback.task?.spotLabel)
        assertEquals("north", feedback.task?.hint)
    }

    @Test
    fun parsesLegacyTaskMessage() {
        val feedback = Protocol.parse(
            """{"v":1,"type":"task","data":{"taskId":3,"spotLabel":"A-01",""" +
                """"spotColor":"blue","hint":"west","deadlineMs":20000}}""",
        ) as ServerMessage.Feedback

        assertEquals(3L, feedback.task?.taskId)
        assertEquals("A-01", feedback.task?.spotLabel)
    }

    @Test
    fun parsesTaskResultAndRoundResult() {
        val taskResult = Protocol.parse(
            """{"v":1,"type":"task_result","data":{"taskId":12,"points":86,""" +
                """"timeScore":62,"precisionScore":34,"collisionPenalty":10,""" +
                """"rating":"great","totalScore":240}}""",
        ) as ServerMessage.TaskResult
        assertEquals(86L, taskResult.points)
        assertEquals("great", taskResult.rating)

        val roundResult = Protocol.parse(
            """{"v":1,"type":"round_result","data":{"roundId":7,"winnerSlot":2,""" +
                """"scores":[{"slot":1,"score":620,"tasks":5,"collisions":1},""" +
                """{"slot":2,"score":710,"tasks":5,"collisions":0}]}}""",
        ) as ServerMessage.RoundResult
        assertEquals(2, roundResult.scores.size)
        assertEquals(710L, roundResult.scores[1].score)
        assertEquals(0, roundResult.scores[1].collisions)
    }

    @Test
    fun parsesErrorAndPong() {
        val error = Protocol.parse(
            """{"v":1,"type":"error","data":{"code":"session_full","message":"No free slot."}}""",
        ) as ServerMessage.ServerError
        assertEquals("session_full", error.code)
        assertEquals("No free slot.", error.message)

        val pong = Protocol.parse(
            """{"v":1,"type":"pong","data":{"clientTime":1234,"serverTime":5678}}""",
        ) as ServerMessage.Pong
        assertEquals(1234L, pong.clientTime)
        assertEquals(5678L, pong.serverTime)
    }

    @Test
    fun unknownTypeAndUnknownFieldsAreIgnored() {
        val message = Protocol.parse(
            """{"v":1,"type":"future_message","data":{"x":1},"extra":"ignored"}""",
        )
        assertEquals(ServerMessage.Unknown("future_message"), message)

        val welcome = Protocol.parse(
            """{"v":1,"type":"welcome","data":{"slot":1,"label":"P1","color":"blue",""" +
                """"phase":"ready","futureField":true}}""",
        )
        assertTrue(welcome is ServerMessage.Welcome)
    }

    @Test(expected = ProtocolException::class)
    fun rejectsUnsupportedVersion() {
        Protocol.parse("""{"v":2,"type":"welcome","data":{}}""")
    }

    @Test(expected = ProtocolException::class)
    fun rejectsMalformedJson() {
        Protocol.parse("{not json")
    }

    @Test(expected = ProtocolException::class)
    fun rejectsFrameWithoutType() {
        Protocol.parse("""{"v":1,"data":{}}""")
    }
}
