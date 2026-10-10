package com.brebond.tipsarr.core.api

import java.net.URLEncoder
import kotlinx.serialization.builtins.ListSerializer

private fun enc(text: String) = URLEncoder.encode(text, "UTF-8")

suspend fun TipsarrApi.library(filters: LibraryFilters, page: Int, pageSize: Int = 48): LibraryPage {
    val params = buildList {
        add("type=${filters.kind.wire}")
        if (filters.query.isNotEmpty()) add("q=${enc(filters.query)}")
        filters.genres.sorted().forEach { add("genre=${enc(it)}") }
        if (filters.anyGenre && filters.genres.size > 1) add("genreMode=any")
        filters.yearFrom?.let { add("yearFrom=$it") }
        filters.yearTo?.let { add("yearTo=$it") }
        filters.minRating?.let { add("minRating=$it") }
        filters.maxRuntime?.let { add("maxRuntime=$it") }
        add("watched=${filters.watched.wire}")
        add("sort=${filters.sort.wire}")
        add("dir=${if (filters.descending) "desc" else "asc"}")
        add("page=$page")
        add("pageSize=$pageSize")
    }
    return get("/library?" + params.joinToString("&"), LibraryPage.serializer())
}

suspend fun TipsarrApi.libraryFacets(): LibraryFacets = get("/library/facets", LibraryFacets.serializer())

suspend fun TipsarrApi.watchlist(): List<MediaItem> = get("/watchlist", ListSerializer(MediaItem.serializer()))

suspend fun TipsarrApi.blocklist(): List<MediaItem> = get("/blocklist", ListSerializer(MediaItem.serializer()))
