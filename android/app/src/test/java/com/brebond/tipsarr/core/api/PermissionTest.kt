package com.brebond.tipsarr.core.api

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PermissionTest {
    private fun flag(name: String, value: Long, vararg implies: String) = PermissionFlag(name, value, "requests", implies.toList().ifEmpty { null })

    // the same implications as backend/internal/perms
    private val flags = listOf(
        flag("manageRequests", Permission.MANAGE_REQUESTS, "requestAdvanced", "requestView", "autoApprove", "autoApproveMovie", "autoApproveTV"),
        flag("request", Permission.REQUEST, "requestMovie", "requestTV"),
        flag("requestMovie", Permission.REQUEST_MOVIE), flag("requestTV", Permission.REQUEST_TV),
        flag("requestAdvanced", Permission.REQUEST_ADVANCED), flag("requestView", Permission.REQUEST_VIEW),
        flag("autoApprove", Permission.AUTO_APPROVE, "autoApproveMovie", "autoApproveTV"),
        flag("autoApproveMovie", Permission.AUTO_APPROVE_MOVIE), flag("autoApproveTV", Permission.AUTO_APPROVE_TV),
    )

    private fun profile(role: String = "user", effective: Long = Permission.DEFAULT) =
        Profile(id = "u", name = "n", role = role, region = "", language = "", ratingSource = "tmdb", effectivePermissions = effective)

    @Test fun hasNeedsEveryFlag() {
        assertTrue(Permission.has(Permission.REQUEST or Permission.MANAGE_USERS, Permission.REQUEST, Permission.MANAGE_USERS))
        assertFalse(Permission.has(Permission.REQUEST, Permission.REQUEST, Permission.MANAGE_USERS))
    }

    @Test fun defaultMaskIsWhatEveryoneHadBefore() {
        assertEquals(4_980_768L, Permission.DEFAULT)
        val p = profile()
        assertTrue(p.mayRequest(MediaType.Movie) && p.mayRequest(MediaType.Tv))
        assertTrue(p.can(Permission.CREATE_ISSUES))
        assertFalse(p.can(Permission.MANAGE_REQUESTS) || p.can(Permission.REQUEST_VIEW) || p.autoApproves(MediaType.Movie))
    }

    @Test fun adminCanDoEverything() {
        val admin = profile(role = "admin", effective = 0)
        assertTrue(admin.can(Permission.MANAGE_USERS, Permission.MANAGE_SETTINGS, Permission.MANAGE_REQUESTS))
        assertTrue(admin.mayRequest(MediaType.Tv) && admin.autoApproves(MediaType.Movie))
    }

    @Test fun requestRightsPerType() {
        val movies = profile(effective = Permission.REQUEST_MOVIE)
        assertTrue(movies.mayRequest(MediaType.Movie))
        assertFalse(movies.mayRequest(MediaType.Tv))
        assertFalse(profile(effective = Permission.CREATE_ISSUES).mayRequest(MediaType.Movie))
        assertTrue(profile(effective = Permission.REQUEST).mayRequest(MediaType.Tv))
    }

    @Test fun autoApproveFollowsTheFlags() {
        val moviesOnly = profile(effective = Permission.REQUEST or Permission.AUTO_APPROVE_MOVIE)
        assertTrue(moviesOnly.autoApproves(MediaType.Movie))
        assertFalse(moviesOnly.autoApproves(MediaType.Tv))
        assertTrue(profile(effective = Permission.REQUEST or Permission.AUTO_APPROVE).autoApproves(MediaType.Tv))
        assertTrue(profile(effective = Permission.MANAGE_REQUESTS).autoApproves(MediaType.Tv))
    }

    @Test fun parentFlagsSwitchOnTheirChildren() {
        val m = Permission.normalize(Permission.MANAGE_REQUESTS, flags)
        listOf(Permission.AUTO_APPROVE, Permission.AUTO_APPROVE_MOVIE, Permission.AUTO_APPROVE_TV, Permission.REQUEST_ADVANCED, Permission.REQUEST_VIEW)
            .forEach { assertTrue("flag $it", Permission.has(m, it)) }
        val r = Permission.normalize(Permission.REQUEST, flags)
        assertTrue(Permission.has(r, Permission.REQUEST_MOVIE, Permission.REQUEST_TV))
        assertEquals(Permission.DEFAULT and Permission.CREATE_ISSUES.inv(), Permission.normalize(Permission.DEFAULT and Permission.CREATE_ISSUES.inv(), flags))
    }

    @Test fun forcedFlagsAreTheImpliedOnes() {
        val forced = Permission.forced(Permission.MANAGE_REQUESTS or Permission.REQUEST, flags)
        assertTrue(Permission.has(forced, Permission.AUTO_APPROVE, Permission.REQUEST_MOVIE, Permission.REQUEST_TV, Permission.REQUEST_VIEW))
        assertFalse("a parent is not forced by itself", Permission.has(forced, Permission.MANAGE_REQUESTS) || Permission.has(forced, Permission.REQUEST))
        assertEquals(0L, Permission.forced(Permission.REQUEST_VIEW, flags))
    }
}
