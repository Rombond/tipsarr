package com.brebond.tipsarr.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette

/** Text input with the app's chrome: label above, card background, ring on focus, error line below. `secure` hides a password. */
@Composable
fun Field(
    title: String,
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    placeholder: String? = null,
    secure: Boolean = false,
    error: String? = null,
    keyboardType: KeyboardType = KeyboardType.Text,
    imeAction: ImeAction = ImeAction.Done,
    onSubmit: () -> Unit = {},
) {
    val p = Tokens.palette
    var focused by remember { mutableStateOf(false) }
    val border = when {
        error != null -> p.destructive
        focused -> p.ring
        else -> p.border
    }
    Column(modifier, verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs)) {
        Text(title, fontSize = Tokens.FontSize.footnote, fontWeight = FontWeight.Medium, color = p.mutedFg)
        OutlinedTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier
                .fillMaxWidth()
                .defaultMinSize(minHeight = Tokens.Size.touchTarget)
                .onFocusChanged { focused = it.isFocused },
            placeholder = placeholder?.let { { Text(it, color = p.mutedFg) } },
            singleLine = true,
            isError = error != null,
            visualTransformation = if (secure) PasswordVisualTransformation() else VisualTransformation.None,
            keyboardOptions = KeyboardOptions(
                capitalization = KeyboardCapitalization.None,
                autoCorrectEnabled = false,
                keyboardType = if (secure) KeyboardType.Password else keyboardType,
                imeAction = imeAction,
            ),
            keyboardActions = KeyboardActions(onDone = { onSubmit() }, onGo = { onSubmit() }, onNext = { onSubmit() }),
            shape = RoundedCornerShape(Tokens.Radius.md),
            colors = OutlinedTextFieldDefaults.colors(
                focusedContainerColor = p.card, unfocusedContainerColor = p.card, errorContainerColor = p.card,
                focusedBorderColor = border, unfocusedBorderColor = border, errorBorderColor = border,
                focusedTextColor = p.fg, unfocusedTextColor = p.fg, errorTextColor = p.fg,
                cursorColor = p.fg,
            ),
        )
        if (error != null) {
            Row(horizontalArrangement = Arrangement.spacedBy(Tokens.Spacing.xs), verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.ErrorOutline, contentDescription = null, tint = p.destructive, modifier = Modifier.defaultMinSize(16.dp))
                Text(error, fontSize = Tokens.FontSize.footnote, color = p.destructive)
            }
        }
    }
}
