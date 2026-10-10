package com.brebond.tipsarr.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import com.brebond.tipsarr.R
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** A pushed screen: back button and an inline title over the app background. */
@Composable
fun TopBarScaffold(title: String, onBack: () -> Unit, modifier: Modifier = Modifier, actions: @Composable RowScope.() -> Unit = {}, content: @Composable () -> Unit) {
    Column(modifier.fillMaxSize().background(Tokens.palette.bg)) {
        Row(Modifier.fillMaxWidth().statusBarsPadding().padding(horizontal = Tokens.Spacing.xs, vertical = Tokens.Spacing.xs), verticalAlignment = Alignment.CenterVertically) {
            IconButton(onBack) { Icon(Icons.AutoMirrored.Outlined.ArrowBack, stringResource(R.string.common_back), tint = Tokens.palette.fg) }
            Text(
                title, fontSize = Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg,
                maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f).semantics { heading() },
            )
            actions()
        }
        Column(Modifier.weight(1f)) { content() }
    }
}
