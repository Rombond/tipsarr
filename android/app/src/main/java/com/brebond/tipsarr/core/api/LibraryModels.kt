package com.brebond.tipsarr.core.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** A title that is in the Jellyfin library. Posters come from Jellyfin through the server. */
@Serializable
data class LibraryItem(
    val type: MediaType,
    val tmdbId: Int,
    val title: String,
    val year: Int? = null,
    val rating: Double? = null,
    val runtimeMinutes: Int? = null,
    val genres: List<String> = emptyList(),
    /** Server-relative path (`/api/v1/images/jellyfin/<id>?tag=<tag>`). */
    val posterUrl: String? = null,
    val watched: Boolean = false,
) {
    val id: String get() = "${type.name}-$tmdbId"
    val posterPath: String? get() = posterUrl?.takeIf { it.isNotEmpty() }
}

@Serializable
data class LibraryPage(val items: List<LibraryItem>, val page: Int, val totalPages: Int, val total: Int) {
    val hasMore: Boolean get() = page < totalPages
}

@Serializable
data class GenreCount(val name: String, val count: Int)

@Serializable
data class LibraryFacets(
    val genres: List<GenreCount> = emptyList(),
    val movies: Int = 0,
    val shows: Int = 0,
    val yearMin: Int = 0,
    val yearMax: Int = 0,
    val maxRuntimeMinutes: Int = 0,
)

enum class LibrarySort(val wire: String) {
    Added("added"), Title("title"), Year("year"), Rating("rating"), Runtime("runtime"), Popular("popular"),
}

enum class WatchedFilter(val wire: String) { Any("any"), Yes("yes"), No("no") }

enum class LibraryKind(val wire: String) { All("all"), Movie("movie"), Tv("tv") }

/** Everything the filters sheet and the chips can change. */
data class LibraryFilters(
    val kind: LibraryKind = LibraryKind.All,
    val query: String = "",
    val genres: Set<String> = emptySet(),
    val yearFrom: Int? = null,
    val yearTo: Int? = null,
    val minRating: Double? = null,
    val maxRuntime: Int? = null,
    val watched: WatchedFilter = WatchedFilter.Any,
    val sort: LibrarySort = LibrarySort.Added,
    val descending: Boolean = true,
) {
    /** Filters from the sheet only (not the chips or the search text). */
    val sheetIsDefault: Boolean
        get() = genres.isEmpty() && yearFrom == null && yearTo == null && minRating == null && maxRuntime == null && sort == LibrarySort.Added && descending
}
