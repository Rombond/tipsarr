package com.brebond.tipsarr.core.api

import android.content.Context

/**
 * Every failure the API layer can surface (the iOS `APIError`): transport failures are [Unreachable],
 * non-2xx responses are [Http], anything the app cannot read is [Unexpected].
 */
sealed class ApiError : Exception() {
    data object Unreachable : ApiError()
    data class Http(val status: Int, val code: String?, val detail: String?) : ApiError()
    data object Unexpected : ApiError()

    val isUnauthorized: Boolean get() = this is Http && status == 401

    /** Localized text for the error code, then its HTTP status, then the server detail. */
    fun message(context: Context): String = when (this) {
        Unreachable -> lookup(context, "m_connect_unreachable").orEmpty()
        Unexpected -> lookup(context, "error_status_500").orEmpty()
        is Http -> code?.let { lookup(context, "error_$it") }
            ?: lookup(context, "error_status_$status")
            ?: detail
            ?: lookup(context, "error_status_500").orEmpty()
    }

    override val message: String? get() = toString()

    companion object {
        fun from(error: Throwable): ApiError = when (error) {
            is ApiError -> error
            is java.io.IOException -> Unreachable
            else -> Unexpected
        }

        private fun lookup(context: Context, name: String): String? {
            val id = context.resources.getIdentifier(name.replace('.', '_').lowercase(), "string", context.packageName)
            return if (id == 0) null else context.getString(id)
        }
    }
}
