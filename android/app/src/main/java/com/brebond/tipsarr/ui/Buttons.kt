package com.brebond.tipsarr.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.ButtonColors
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.Button
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

enum class ButtonKind { Primary, Secondary, Ghost, Destructive }

/** The app's button: primary, secondary (outlined), ghost, destructive. `compact` = 36 dp for list rows. */
@Composable
fun TipsarrButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    kind: ButtonKind = ButtonKind.Primary,
    fullWidth: Boolean = false,
    compact: Boolean = false,
    enabled: Boolean = true,
    leading: (@Composable RowScope.() -> Unit)? = null,
) {
    val p = Tokens.palette
    val (container, content) = when (kind) {
        ButtonKind.Primary -> p.primary to p.primaryFg
        ButtonKind.Secondary -> p.card to p.fg
        ButtonKind.Ghost -> Color.Transparent to p.fg
        ButtonKind.Destructive -> p.destructive to p.bg
    }
    Button(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier
            .then(if (fullWidth) Modifier.fillMaxWidth() else Modifier)
            .defaultMinSize(minHeight = if (compact) 36.dp else Tokens.Size.touchTarget),
        shape = RoundedCornerShape(Tokens.Radius.md),
        colors = ButtonColors(container, content, container.copy(alpha = 0.4f), content.copy(alpha = 0.4f)),
        border = if (kind == ButtonKind.Secondary) BorderStroke(1.dp, p.border) else null,
        contentPadding = PaddingValues(horizontal = if (compact) Tokens.Spacing.md else Tokens.Spacing.lg),
        elevation = null,
    ) {
        leading?.invoke(this)
        Text(text, fontSize = if (compact) Tokens.FontSize.subhead else Tokens.FontSize.headline, fontWeight = FontWeight.SemiBold)
    }
}
