package com.brebond.tipsarr.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.sp
import com.brebond.tipsarr.core.images.RemoteImage
import com.brebond.tipsarr.core.images.TmdbSize
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** Round picture of a user, initials when there is none. `version` changes when the picture does, so the cached copy is not reused. */
@Composable
fun AvatarView(userId: String, name: String, size: Dp, version: Int = 0, modifier: Modifier = Modifier) {
    Box(modifier.size(size).clip(CircleShape).background(Tokens.palette.muted), contentAlignment = Alignment.Center) {
        Text(name.take(1).uppercase(), fontSize = (size.value * 0.4f).sp, fontWeight = FontWeight.SemiBold, color = Tokens.palette.mutedFg)
        RemoteImage("/api/v1/users/$userId/avatar?v=$version", TmdbSize.W185, Modifier.size(size), server = true)
    }
}
