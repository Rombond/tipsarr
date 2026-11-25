package com.brebond.tipsarr.core.api

import kotlinx.serialization.Serializable

/** Android sends only the FCM token and the server key (no other keys: the server rejects unknown ones). */
@Serializable
private data class FollowBody(val pushToken: String, val server: String? = null)

/** A followed request as the server keeps it (`RequestFollow`). */
@Serializable
data class RequestFollow(
    val requestId: String,
    val platform: String,
    val stage: String,
    val percent: Int = 0,
    val etaSeconds: Int = 0,
    val updatedAt: Long = 0,
    val expiresAt: Long = 0,
)

@Serializable
data class FollowList(val items: List<RequestFollow> = emptyList())

/** `PUT /requests/{id}/follow`: idempotent, call again when the FCM token changes. Pushes the expiry to now + 24 h. */
suspend fun TipsarrApi.follow(id: String, pushToken: String, server: String?): RequestFollow =
    put("/requests/$id/follow", FollowBody(pushToken, server), FollowBody.serializer(), RequestFollow.serializer())

/** `DELETE /requests/{id}/follow`: no push follows, the app ends its own notification. */
suspend fun TipsarrApi.unfollow(id: String) = delete("/requests/$id/follow")

/** `GET /me/follows`: the follows of this session, to reconcile at launch. */
suspend fun TipsarrApi.follows(): FollowList = get("/me/follows", FollowList.serializer())
