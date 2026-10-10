package com.brebond.tipsarr.core.api

import java.net.URLEncoder

private fun MediaType.path() = if (this == MediaType.Movie) "movie" else "tv"

suspend fun TipsarrApi.genres(type: MediaType): List<Genre> =
    get("/discover/genres/${type.path()}", kotlinx.serialization.builtins.ListSerializer(Genre.serializer()))

suspend fun TipsarrApi.byGenre(type: MediaType, genre: Int, page: Int): MediaPage =
    get("/discover/${if (type == MediaType.Movie) "movies" else "tv"}?page=$page&genre=$genre", MediaPage.serializer())

suspend fun TipsarrApi.search(query: String, page: Int): SearchPage =
    get("/search?page=$page&q=${URLEncoder.encode(query, "UTF-8")}", SearchPage.serializer())

suspend fun TipsarrApi.person(id: Int): PersonDetail = get("/person/$id", PersonDetail.serializer())

suspend fun TipsarrApi.requests(filter: RequestFilter, skip: Int, take: Int = 20): RequestPage {
    val list = get("/requests?filter=${filter.wire}&take=$take&skip=$skip", RequestList.serializer())
    return RequestPage(list.items.map { it.toRecord() }, list.total)
}

suspend fun TipsarrApi.requestCounts(): RequestCounts = get("/requests/counts", RequestCounts.serializer())

suspend fun TipsarrApi.request(id: String): RequestRecord = get("/requests/$id", RequestView.serializer()).toRecord()

suspend fun TipsarrApi.approve(id: String): RequestRecord =
    post("/requests/$id/approve", ApproveBody(), ApproveBody.serializer(), RequestView.serializer()).toRecord()

suspend fun TipsarrApi.decline(id: String, reason: String): RequestRecord =
    post("/requests/$id/decline", DeclineBody(reason.ifEmpty { null }), DeclineBody.serializer(), RequestView.serializer()).toRecord()
