package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable

/**
 * Permission flags, the same values as `backend/internal/perms` and `frontend/src/lib/permissions.ts` (Seerr's numbers).
 * A person's `effectivePermissions` is the sum of the flags they have; admins have every flag.
 */
object Permission {
    const val MANAGE_SETTINGS = 4L
    const val MANAGE_USERS = 8L
    const val MANAGE_REQUESTS = 16L
    const val REQUEST = 32L
    const val AUTO_APPROVE = 128L
    const val AUTO_APPROVE_MOVIE = 256L
    const val AUTO_APPROVE_TV = 512L
    const val REQUEST_ADVANCED = 8192L
    const val REQUEST_VIEW = 16384L
    const val REQUEST_MOVIE = 262144L
    const val REQUEST_TV = 524288L
    const val MANAGE_ISSUES = 1048576L
    const val VIEW_ISSUES = 2097152L
    const val CREATE_ISSUES = 4194304L

    /** What a new person gets until the server says otherwise: request movies and shows, report problems. */
    const val DEFAULT = REQUEST or REQUEST_MOVIE or REQUEST_TV or CREATE_ISSUES

    /** True when the mask has every one of the flags. */
    fun has(mask: Long, vararg flags: Long): Boolean = flags.all { mask and it == it }

    /** Switches on the flags that come with another one, as the server does when it saves a mask. */
    fun normalize(mask: Long, flags: List<PermissionFlag>): Long {
        val byName = flags.associate { it.name to it.value }
        var m = mask
        for (f in flags) {
            if (m and f.value == 0L) continue
            for (name in f.implies.orEmpty()) m = m or (byName[name] ?: 0L)
        }
        return m
    }

    /** The flags that are on only because another flag implies them (they cannot be switched off alone). */
    fun forced(mask: Long, flags: List<PermissionFlag>): Long {
        val byName = flags.associate { it.name to it.value }
        var out = 0L
        for (f in flags) {
            if (mask and f.value == 0L) continue
            for (name in f.implies.orEmpty()) out = out or (byName[name] ?: 0L)
        }
        return out
    }
}

/** One right as `GET /admin/permissions` lists it. `reserved` flags exist only so the numbers match Seerr: do not show them. */
@Serializable
data class PermissionFlag(
    val name: String,
    val value: Long,
    val group: String = "",
    val implies: List<String>? = null,
    val reserved: Boolean = false,
)

/** What new people start with: a permission mask and the request limits of people without their own (0 = no limit). */
@Serializable
data class PermissionDefaults(
    val permissions: Long = Permission.DEFAULT,
    val movieQuotaLimit: Int = 0,
    val movieQuotaDays: Int = 0,
    val tvQuotaLimit: Int = 0,
    val tvQuotaDays: Int = 0,
)

@Serializable
data class PermissionCatalog(val flags: List<PermissionFlag> = emptyList(), val defaults: PermissionDefaults = PermissionDefaults())

/** A person's own request limits; a missing value means the default applies. */
@Serializable
data class UserQuota(val movieLimit: Int? = null, val movieDays: Int? = null, val tvLimit: Int? = null, val tvDays: Int? = null)

@Serializable
data class AccessBody(val permissions: Long, val quota: UserQuota)

/** One request limit: `limit` requests (movies) or seasons (shows) every `days` days; 0 = no limit. */
@Serializable
data class QuotaWindow(
    val days: Int = 0,
    val limit: Int = 0,
    val used: Int = 0,
    val remaining: Int? = null,
    val restricted: Boolean = false,
    val resetsAt: Long = 0,
)

@Serializable
data class Quota(val movie: QuotaWindow = QuotaWindow(), val tv: QuotaWindow = QuotaWindow()) {
    fun of(type: MediaType): QuotaWindow = if (type == MediaType.Movie) movie else tv
}
