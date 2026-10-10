package com.brebond.tipsarr.core.images

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import coil3.ImageLoader
import coil3.compose.AsyncImage
import coil3.disk.DiskCache
import coil3.memory.MemoryCache
import coil3.network.okhttp.OkHttpNetworkFetcherFactory
import coil3.request.ImageRequest
import coil3.request.crossfade
import okhttp3.OkHttpClient
import okio.Path.Companion.toOkioPath
import java.util.concurrent.TimeUnit

enum class TmdbSize(val path: String) { W92("w92"), W185("w185"), W342("w342"), W500("w500"), W780("w780"), W1280("w1280") }

/** Where images come from: the active account's server and token (TMDB images are proxied by the server). */
data class ImageSource(val serverUrl: String, val token: String) {
    fun tmdbUrl(size: TmdbSize, path: String): String =
        serverUrl.trimEnd('/') + "/api/v1/images/tmdb/${size.path}" + (if (path.startsWith("/")) path else "/$path")

    /** A path on the Tipsarr server itself, e.g. `/api/v1/images/jellyfin/<id>?tag=<tag>`. */
    fun serverUrl(path: String): String = serverUrl.trimEnd('/') + (if (path.startsWith("/")) path else "/$path")
}

/** Image loader that sends the account's Bearer token: memory cache in front of a 200 MB disk cache. */
fun createImageLoader(context: Context, source: ImageSource): ImageLoader {
    val client = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(15, TimeUnit.SECONDS)
        .addInterceptor { chain ->
            chain.proceed(chain.request().newBuilder().header("Authorization", "Bearer ${source.token}").build())
        }
        .build()
    return ImageLoader.Builder(context)
        .components {
            add(OkHttpNetworkFetcherFactory(callFactory = { client }))
            add(coil3.svg.SvgDecoder.Factory())
        }
        .memoryCache { MemoryCache.Builder().maxSizePercent(context, 0.2).build() }
        .diskCache { DiskCache.Builder().directory(context.cacheDir.resolve("images").toOkioPath()).maxSizeBytes(200L shl 20).build() }
        .crossfade(true)
        .build()
}

val LocalImageSource = compositionLocalOf<ImageSource?> { null }
val LocalAppImageLoader = compositionLocalOf<ImageLoader?> { null }

/** A TMDB image (`path` such as `/abc.jpg`) or a path on the Tipsarr server; draws nothing while loading or when there is none. */
@Composable
fun RemoteImage(
    path: String?,
    size: TmdbSize,
    modifier: Modifier = Modifier,
    contentScale: ContentScale = ContentScale.Crop,
    server: Boolean = false,
) {
    val source = LocalImageSource.current
    val loader = LocalAppImageLoader.current
    if (path.isNullOrEmpty() || source == null || loader == null) return
    AsyncImage(
        model = ImageRequest.Builder(coil3.compose.LocalPlatformContext.current)
            .data(if (server) source.serverUrl(path) else source.tmdbUrl(size, path))
            .build(),
        imageLoader = loader,
        contentDescription = null,
        contentScale = contentScale,
        modifier = modifier,
    )
}
