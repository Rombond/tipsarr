package com.brebond.tipsarr.features.profile

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.ImageDecoder
import android.net.Uri
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Photo
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.brebond.tipsarr.R
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.deleteAvatar
import com.brebond.tipsarr.core.api.uploadAvatar
import com.brebond.tipsarr.core.support.LocalToast
import com.brebond.tipsarr.core.support.ToastKind
import com.brebond.tipsarr.design.Tokens
import com.brebond.tipsarr.design.palette
import com.brebond.tipsarr.ui.ButtonKind
import com.brebond.tipsarr.ui.TipsarrButton
import com.brebond.tipsarr.ui.TipsarrSheet
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream

/** Change or remove the profile picture. Max 4 MB: the photo is scaled down before it is sent. */
@Composable
fun PictureSheet(api: TipsarrApi, onDismiss: () -> Unit, onChanged: () -> Unit) {
    val context = LocalContext.current
    val toast = LocalToast.current
    val scope = rememberCoroutineScope()
    var working by remember { mutableStateOf(false) }

    val picker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        scope.launch {
            working = true
            try {
                val jpeg = withContext(Dispatchers.IO) { scaledJpeg(context, uri) }
                if (jpeg == null) {
                    toast?.show(context.getString(R.string.profile_picture_invalid), ToastKind.Error)
                } else {
                    api.uploadAvatar(jpeg)
                    toast?.show(context.getString(R.string.profile_picture_saved))
                    onChanged()
                    onDismiss()
                }
            } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) } finally { working = false }
        }
    }

    TipsarrSheet(onDismiss) {
        Column(
            Modifier.fillMaxWidth().navigationBarsPadding().padding(horizontal = Tokens.Spacing.xl).padding(bottom = Tokens.Spacing.xl),
            verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.lg),
        ) {
            Text(stringResource(R.string.m_profile_picture_title), fontSize = Tokens.FontSize.title3, fontWeight = FontWeight.SemiBold, color = Tokens.palette.fg)
            Column(verticalArrangement = Arrangement.spacedBy(Tokens.Spacing.sm)) {
                TipsarrButton(stringResource(R.string.m_picture_library), { picker.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)) }, fullWidth = true, kind = ButtonKind.Secondary, enabled = !working) {
                    Icon(Icons.Outlined.Photo, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm))
                }
                TipsarrButton(
                    stringResource(R.string.profile_remove_picture),
                    {
                        scope.launch {
                            working = true
                            try { api.deleteAvatar(); onChanged(); onDismiss() } catch (e: ApiError) { toast?.show(e.message(context), ToastKind.Error) } finally { working = false }
                        }
                    },
                    fullWidth = true, kind = ButtonKind.Secondary, enabled = !working,
                ) { Icon(Icons.Outlined.Delete, null, Modifier.size(20.dp)); Spacer(Modifier.width(Tokens.Spacing.sm)) }
                TipsarrButton(stringResource(R.string.common_cancel), onDismiss, fullWidth = true, kind = ButtonKind.Ghost)
            }
            Text(stringResource(R.string.m_picture_footer), fontSize = Tokens.FontSize.footnote, color = Tokens.palette.mutedFg, textAlign = TextAlign.Center, modifier = Modifier.fillMaxWidth())
        }
    }
}

/** 768 px on the long side, JPEG: well under the 4 MB limit. Null when the file is not a picture. */
private fun scaledJpeg(context: Context, uri: Uri): ByteArray? = runCatching {
    val bitmap: Bitmap = if (Build.VERSION.SDK_INT >= 28) {
        ImageDecoder.decodeBitmap(ImageDecoder.createSource(context.contentResolver, uri)) { decoder, info, _ ->
            val longSide = maxOf(info.size.width, info.size.height)
            val scale = minOf(1f, 768f / longSide)
            decoder.setTargetSize((info.size.width * scale).toInt().coerceAtLeast(1), (info.size.height * scale).toInt().coerceAtLeast(1))
            decoder.allocator = ImageDecoder.ALLOCATOR_SOFTWARE
        }
    } else {
        val source = context.contentResolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it) } ?: return null
        val longSide = maxOf(source.width, source.height)
        val scale = minOf(1f, 768f / longSide)
        Bitmap.createScaledBitmap(source, (source.width * scale).toInt().coerceAtLeast(1), (source.height * scale).toInt().coerceAtLeast(1), true)
    }
    ByteArrayOutputStream().also { bitmap.compress(Bitmap.CompressFormat.JPEG, 85, it) }.toByteArray()
}.getOrNull()
