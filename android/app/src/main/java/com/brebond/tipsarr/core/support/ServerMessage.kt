package com.brebond.tipsarr.core.support

import android.content.Context

/**
 * A message the server sent as a code plus values (job summaries): the translated text with the values in
 * placeholder order, or [fallback] (the server's English text) for a code this version does not know.
 */
fun serverMessage(context: Context, code: String, args: List<String>, fallback: String): String {
    if (code.isEmpty()) return fallback
    val id = context.resources.getIdentifier(code.replace('.', '_').replace('-', '_').lowercase(), "string", context.packageName)
    if (id == 0) return fallback
    return runCatching { context.getString(id, *args.toTypedArray()) }.getOrDefault(fallback)
}
