package com.brebond.tipsarr.core.api

private fun MediaType.path() = if (this == MediaType.Movie) "movie" else "tv"

suspend fun TipsarrApi.detail(type: MediaType, id: Int): MediaDetail = get("/media/${type.path()}/$id", MediaDetail.serializer())

suspend fun TipsarrApi.episodes(tmdbId: Int, season: Int): List<Episode> =
    get("/media/tv/$tmdbId/seasons/$season", SeasonDetail.serializer()).episodes

/** Movie scores from IMDb, Rotten Tomatoes and Metacritic. */
suspend fun TipsarrApi.movieScores(id: Int): RatingsSummary {
    val scores = get("/media/movie/$id/ratings", MovieScores.serializer())
    return RatingsSummary(imdb = scores.imdb?.value, rottenTomatoes = scores.rottenTomatoes?.value, metacritic = scores.metacritic?.value, rottenTomatoesUrl = scores.rottenTomatoesUrl?.takeIf { it.isNotEmpty() })
}

suspend fun TipsarrApi.flags(type: MediaType, id: Int): TitleFlags = get("/media/${type.path()}/$id/flags", TitleFlags.serializer())

/** Null when the server has nothing to choose from (no Radarr/Sonarr configured for the user). */
suspend fun TipsarrApi.requestOptions(type: MediaType): RequestOptions? =
    try { get("/requests/options?type=${type.path()}", RequestOptions.serializer()) } catch (e: ApiError) { null }

suspend fun TipsarrApi.createRequest(type: MediaType, tmdbId: Int, seasons: List<Int>?, profileId: Int?, folder: String?): RequestRecord =
    post("/requests", CreateRequestBody(type, tmdbId, seasons, profileId, folder), CreateRequestBody.serializer(), RequestView.serializer()).toRecord()

suspend fun TipsarrApi.myRequests(take: Int = 100): List<RequestRecord> =
    get("/requests?filter=mine&take=$take", RequestList.serializer()).items.map { it.toRecord() }

suspend fun TipsarrApi.deleteRequest(id: String) = delete("/requests/$id")

suspend fun TipsarrApi.retryRequest(id: String): RequestRecord = postForResult("/requests/$id/retry", RequestView.serializer()).toRecord()

suspend fun TipsarrApi.setWatchlisted(on: Boolean, type: MediaType, id: Int) {
    if (on) postUnit("/watchlist", WatchFlagBody(id, type), WatchFlagBody.serializer()) else delete("/watchlist/${type.path()}/$id")
}

suspend fun TipsarrApi.setBlocklisted(on: Boolean, type: MediaType, id: Int) {
    if (on) postUnit("/blocklist", WatchFlagBody(id, type), WatchFlagBody.serializer()) else delete("/blocklist/${type.path()}/$id")
}

suspend fun TipsarrApi.reportIssue(type: MediaType, tmdbId: Int, kind: IssueKind, message: String) =
    postUnit("/issues", CreateIssueBody(kind, message, tmdbId, type), CreateIssueBody.serializer())
