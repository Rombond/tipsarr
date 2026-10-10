package com.brebond.tipsarr.core.api

import kotlinx.serialization.builtins.ListSerializer

// Issues

suspend fun TipsarrApi.issues(filter: IssueFilter, skip: Int, take: Int = 20): IssueList =
    get("/issues?filter=${filter.wire}&take=$take&skip=$skip", IssueList.serializer())

suspend fun TipsarrApi.openIssueCount(): Int = get("/issues/counts", IssueCounts.serializer()).open

suspend fun TipsarrApi.issue(id: String): IssueThread = get("/issues/$id", IssueThread.serializer())

suspend fun TipsarrApi.comment(id: String, message: String): IssueThread =
    post("/issues/$id/comments", CommentBody(message), CommentBody.serializer(), IssueThread.serializer())

suspend fun TipsarrApi.resolveIssue(id: String): IssueThread = postForResult("/issues/$id/resolve", IssueThread.serializer())

suspend fun TipsarrApi.reopenIssue(id: String): IssueThread = postForResult("/issues/$id/reopen", IssueThread.serializer())

suspend fun TipsarrApi.deleteIssue(id: String) = delete("/issues/$id")

// Users

suspend fun TipsarrApi.users(): List<Profile> = get("/admin/users", ListSerializer(Profile.serializer()))

suspend fun TipsarrApi.userDetail(id: String): UserDetail = get("/users/$id", UserDetail.serializer())

suspend fun TipsarrApi.setRole(id: String, admin: Boolean): Profile =
    patch("/admin/users/$id", RoleBody(if (admin) "admin" else "user"), RoleBody.serializer(), Profile.serializer())

// Sync

suspend fun TipsarrApi.syncStatus(): SyncStatus = get("/admin/sync", SyncStatus.serializer())

suspend fun TipsarrApi.runJob(name: String) = postEmpty("/admin/sync/$name")

// Stats

suspend fun TipsarrApi.stats(period: StatsPeriod, user: String?): FullStats =
    get("/stats?period=${period.wire}" + (user?.let { "&user=$it" } ?: ""), FullStats.serializer())

/** Available requests whose requester has not watched them. `user` null = everyone. */
suspend fun TipsarrApi.unwatchedRequests(user: String?): RequestPage {
    val list = get("/requests?filter=unwatched&take=40" + (user?.let { "&user=$it" } ?: ""), RequestList.serializer())
    return RequestPage(list.items.map { it.toRecord() }, list.total)
}

/** Library titles nobody ever played, oldest first (admins). */
suspend fun TipsarrApi.neverWatched(): LibraryPage =
    get("/library?neverWatched=true&sort=added&dir=asc&page=1&pageSize=30", LibraryPage.serializer())
