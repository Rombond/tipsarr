package com.brebond.tipsarr.ui

import androidx.annotation.StringRes
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ArrowCircleDown
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.Cancel
import androidx.compose.material.icons.outlined.Schedule
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material.icons.filled.ArrowDownward
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.PriorityHigh
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Contrast
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import com.brebond.tipsarr.R
import com.brebond.tipsarr.design.Tokens

/** Lifecycle of a request or title, shared by badges, timelines and posters (same cases as the iOS `RequestState`). */
enum class RequestState(@StringRes val title: Int, val color: Color, val icon: ImageVector) {
    Requested(R.string.state_requested, Tokens.Status.requested, Icons.Outlined.Schedule),
    Approved(R.string.state_approved, Tokens.Status.approved, Icons.Outlined.CheckCircle),
    Searching(R.string.state_searching, Tokens.Status.searching, Icons.Outlined.Search),
    Downloading(R.string.state_downloading, Tokens.Status.downloading, Icons.Outlined.ArrowCircleDown),
    Available(R.string.state_available, Tokens.Status.available, Icons.Filled.CheckCircle),
    Partial(R.string.state_partial, Tokens.Status.partial, Icons.Filled.Contrast),
    Declined(R.string.state_declined, Tokens.Status.declined, Icons.Outlined.Cancel),
    Failed(R.string.state_failed, Tokens.Status.failed, Icons.Outlined.WarningAmber);

    /** Glyph for the solid poster badge: no circle of its own, the badge is the circle. */
    val solidIcon: ImageVector
        get() = when (this) {
            Requested -> Icons.Filled.Schedule
            Approved, Available -> Icons.Filled.Check
            Searching -> Icons.Filled.Search
            Downloading -> Icons.Filled.ArrowDownward
            Partial -> Icons.Filled.Contrast
            Declined -> Icons.Filled.Close
            Failed -> Icons.Filled.PriorityHigh
        }

    companion object {
        /** `stage` is the fine-grained state the server sends; older servers only send `status`. */
        fun from(stage: String?, status: String): RequestState = when (stage ?: status) {
            "requested", "pending" -> Requested
            "approved" -> Approved
            "searching" -> Searching
            "downloading" -> Downloading
            "available" -> Available
            "partial" -> Partial
            "declined" -> Declined
            "failed" -> Failed
            else -> Requested
        }
    }
}
