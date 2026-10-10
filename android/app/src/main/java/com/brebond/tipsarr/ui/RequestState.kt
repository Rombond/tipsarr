package com.brebond.tipsarr.ui

import androidx.annotation.StringRes
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ArrowCircleDown
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.Cancel
import androidx.compose.material.icons.outlined.Schedule
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material.icons.filled.CheckCircle
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
    Failed(R.string.state_failed, Tokens.Status.failed, Icons.Outlined.WarningAmber),
}
