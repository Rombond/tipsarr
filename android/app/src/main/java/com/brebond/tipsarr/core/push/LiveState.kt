package com.brebond.tipsarr.core.push

/** One FCM `live` message (data-only, every value a string): the progress of a followed request. */
data class LiveState(
    val id: String,
    val server: String?,
    val stage: String,
    val percent: Int,
    val etaSeconds: Int,
    val updatedAt: Long,
    val end: Boolean,
    val dismissAfter: Long,
) {
    val isFinal: Boolean get() = stage in FINAL

    /** True when this message is older than (or the same as) what is already shown. */
    fun isStale(lastUpdatedAt: Long?): Boolean = lastUpdatedAt != null && updatedAt <= lastUpdatedAt

    companion object {
        val STAGES = setOf("requested", "approved", "searching", "downloading", "available", "declined", "failed")
        val FINAL = setOf("available", "declined", "failed")

        /** Null when the message is not a live message or misses its id, stage or time. Numbers out of range are clamped. */
        fun parse(data: Map<String, String>): LiveState? {
            if (data["event"] != "live") return null
            val id = data["id"]?.takeIf { it.isNotBlank() } ?: return null
            val stage = data["stage"]?.takeIf { it in STAGES } ?: return null
            val updatedAt = data["updatedAt"]?.toLongOrNull()?.takeIf { it > 0 } ?: return null
            return LiveState(
                id = id,
                server = data["server"]?.takeIf { it.isNotBlank() },
                stage = stage,
                percent = (data["percent"]?.toIntOrNull() ?: 0).coerceIn(0, 100),
                etaSeconds = (data["etaSeconds"]?.toIntOrNull() ?: 0).coerceAtLeast(0),
                updatedAt = updatedAt,
                end = data["end"] == "true",
                dismissAfter = (data["dismissAfter"]?.toLongOrNull() ?: 0).coerceAtLeast(0),
            )
        }
    }
}
