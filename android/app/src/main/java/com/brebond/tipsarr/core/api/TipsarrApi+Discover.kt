package com.brebond.tipsarr.core.api

suspend fun TipsarrApi.trending(page: Int): MediaPage = list("/discover/trending?page=$page")
suspend fun TipsarrApi.upcoming(page: Int): MediaPage = list("/discover/upcoming?page=$page")
suspend fun TipsarrApi.popularMovies(page: Int): MediaPage = list("/discover/movies?page=$page")
suspend fun TipsarrApi.popularTv(page: Int): MediaPage = list("/discover/tv?page=$page")
suspend fun TipsarrApi.suggestions(): SuggestionResult = get("/suggestions", SuggestionResult.serializer())

private suspend fun TipsarrApi.list(path: String): MediaPage = get(path, MediaPage.serializer())
