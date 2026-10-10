package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable
import java.text.NumberFormat
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.util.Currency
import java.util.Locale

@Serializable
data class BoxOfficeWeek(val key: String, val label: String = "")

@Serializable
data class BoxOfficeEntry(
    val position: Int,
    val title: String,
    val weekendGross: Long = 0,
    val totalGross: Long = 0,
    val weeksInRelease: Int = 0,
    /** The matching TMDB title; null when the chart title was not matched. */
    val item: MediaItem? = null,
)

/** One weekend (or week) of the box office in a region, as the server stores it. */
@Serializable
data class BoxOfficeChart(
    val region: String,
    val regions: List<String> = emptyList(),
    val week: String = "",
    val start: String = "",
    val end: String = "",
    val weeks: List<BoxOfficeWeek> = emptyList(),
    val entries: List<BoxOfficeEntry> = emptyList(),
) {
    /** "Oct 2 – Oct 4 · US" */
    fun title(): String {
        val from = runCatching { LocalDate.parse(start) }.getOrNull()
        val to = runCatching { LocalDate.parse(end) }.getOrNull()
        if (from == null || to == null) return region
        val short = { d: LocalDate -> d.format(DateTimeFormatter.ofPattern("MMM d", Locale.getDefault())) }
        return "${short(from)} – ${short(to)} · $region"
    }

    /** Gross in the region's currency, compact ("$12M"). */
    fun money(amount: Long): String {
        val currency = runCatching { Currency.getInstance(Locale.Builder().setRegion(region).build()) }.getOrDefault(Currency.getInstance("USD"))
        val format = NumberFormat.getCurrencyInstance().apply { this.currency = currency; maximumFractionDigits = 0 }
        val (value, suffix) = when {
            amount >= 1_000_000_000 -> amount / 1e9 to "B"
            amount >= 1_000_000 -> amount / 1e6 to "M"
            amount >= 1_000 -> amount / 1e3 to "K"
            else -> amount.toDouble() to ""
        }
        format.maximumFractionDigits = if (suffix.isEmpty()) 0 else 1
        return format.format(value).let { if (suffix.isEmpty()) it else it + suffix }
    }
}

suspend fun TipsarrApi.boxOffice(region: String? = null, week: String? = null): BoxOfficeChart {
    val params = listOfNotNull(region?.let { "region=$it" }, week?.let { "week=$it" })
    return get("/boxoffice" + if (params.isEmpty()) "" else "?" + params.joinToString("&"), BoxOfficeChart.serializer())
}
