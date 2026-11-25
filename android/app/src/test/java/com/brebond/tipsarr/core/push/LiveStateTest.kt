package com.brebond.tipsarr.core.push

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class LiveStateTest {
    private val message = mapOf(
        "event" to "live", "id" to "r1", "server" to "tipsarr.example:8080", "stage" to "downloading",
        "percent" to "42", "etaSeconds" to "1800", "updatedAt" to "1800000000", "end" to "false", "dismissAfter" to "0",
    )

    @Test fun parsesMessage() {
        val state = LiveState.parse(message)!!
        assertEquals(LiveState("r1", "tipsarr.example:8080", "downloading", 42, 1800, 1_800_000_000L, false, 0), state)
        assertFalse(state.isFinal)
    }

    @Test fun ignoresOtherEvents() {
        assertNull(LiveState.parse(message + ("event" to "request.available")))
        assertNull(LiveState.parse(message - "event"))
    }

    @Test fun rejectsMissingOrBadRequiredValues() {
        assertNull(LiveState.parse(message - "id"))
        assertNull(LiveState.parse(message + ("stage" to "partial")))
        assertNull(LiveState.parse(message + ("updatedAt" to "soon")))
        assertNull(LiveState.parse(message - "updatedAt"))
    }

    @Test fun clampsBadNumbers() {
        val state = LiveState.parse(message + mapOf("percent" to "140", "etaSeconds" to "-5", "dismissAfter" to "x", "server" to ""))!!
        assertEquals(100, state.percent)
        assertEquals(0, state.etaSeconds)
        assertEquals(0, state.dismissAfter)
        assertNull(state.server)
        assertEquals(0, LiveState.parse(message + ("percent" to "abc"))!!.percent)
    }

    @Test fun readsEndFlag() {
        val state = LiveState.parse(message + mapOf("stage" to "available", "end" to "true", "dismissAfter" to "3600"))!!
        assertTrue(state.end)
        assertTrue(state.isFinal)
        assertEquals(3600, state.dismissAfter)
        assertFalse(LiveState.parse(message + ("end" to "yes"))!!.end)
    }

    @Test fun dropsStaleUpdates() {
        val state = LiveState.parse(message)!!
        assertFalse(state.isStale(null))
        assertFalse(state.isStale(1_799_999_999L))
        assertTrue(state.isStale(1_800_000_000L))
        assertTrue(state.isStale(1_800_000_001L))
    }
}
