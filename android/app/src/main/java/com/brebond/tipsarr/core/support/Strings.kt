package com.brebond.tipsarr.core.support

import android.content.Context
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle
import java.util.Locale

/** A string resource looked up by its generated name (`error_not_found`), or null when there is none. */
fun Context.stringByName(name: String): String? {
    val id = resources.getIdentifier(name.replace('.', '_').lowercase(), "string", packageName)
    return if (id == 0) null else getString(id)
}

/**
 * Web strings carry plurals as "one | other" (see `seasons.episodes`). Picks the form and fills `%1$s`.
 * English: one only for 1. French: one for 0 and 1.
 */
fun Context.plural(name: String, count: Int): String {
    val raw = stringByName(name) ?: return count.toString()
    val forms = raw.split(" | ")
    val singular = if (Locale.getDefault().language == "fr") count <= 1 else count == 1
    val form = if (forms.size > 1) (if (singular) forms[0] else forms[1]) else raw
    return form.replace("%1\$s", count.toString()).replace("%s", count.toString())
}

/** "2h 46m" / "45m". */
fun formatRuntime(minutes: Int): String {
    val h = minutes / 60
    val m = minutes % 60
    return when {
        h > 0 && m > 0 -> "${h}h ${m}m"
        h > 0 -> "${h}h"
        else -> "${m}m"
    }
}

/** "2026-10-09" as a long local date; the text itself when it does not parse. */
fun formatDate(iso: String, style: FormatStyle = FormatStyle.LONG): String =
    runCatching { LocalDate.parse(iso.take(10)).format(DateTimeFormatter.ofLocalizedDate(style)) }.getOrDefault(iso)

/** Language name in the app's language, e.g. "fr" gives "Français". */
fun languageName(code: String): String? {
    val name = Locale.forLanguageTag(code).getDisplayLanguage(Locale.getDefault())
    return name.takeIf { it.isNotEmpty() && it != code }?.replaceFirstChar { it.titlecase(Locale.getDefault()) }
}

/** "4 hours ago" in the app's language, from epoch seconds. */
fun relativeTime(context: Context, epochSeconds: Long): String =
    android.text.format.DateUtils.getRelativeTimeSpanString(
        epochSeconds * 1000, System.currentTimeMillis(), android.text.format.DateUtils.MINUTE_IN_MILLIS,
    ).toString()

/** Picks the form of a text whose forms are separated by " | " (the first is for one, the second for every other number). */
fun pluralForm(text: String, count: Int): String {
    val forms = text.split(" | ")
    val one = count == 1 || (count == 0 && Locale.getDefault().language == "fr")
    return forms[if (one) 0 else minOf(1, forms.lastIndex)]
}

/**
 * What is left of a request limit, in words: "2 movie requests left in the next 7 days", or when the limit is used up the
 * day requests are possible again. Empty when there is no limit.
 */
fun quotaText(context: Context, type: com.brebond.tipsarr.core.api.MediaType, window: com.brebond.tipsarr.core.api.QuotaWindow): String {
    if (window.limit <= 0) return ""
    if (window.restricted) {
        val date = if (window.resetsAt > 0) java.text.DateFormat.getDateInstance(java.text.DateFormat.LONG).format(java.util.Date(window.resetsAt * 1000)) else ""
        return context.getString(com.brebond.tipsarr.R.string.req_quota_none, date)
    }
    val remaining = window.remaining ?: 0
    val raw = context.getString(if (type == com.brebond.tipsarr.core.api.MediaType.Movie) com.brebond.tipsarr.R.string.req_quota_movie else com.brebond.tipsarr.R.string.req_quota_tv)
    return runCatching { String.format(pluralForm(raw, remaining), remaining.toString(), window.days.toString()) }.getOrDefault("")
}
