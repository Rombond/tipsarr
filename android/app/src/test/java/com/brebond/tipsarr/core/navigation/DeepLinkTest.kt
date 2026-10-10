package com.brebond.tipsarr.core.navigation

import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.live.EventStream
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class DeepLinkTest {
    private fun parse(host: String, vararg parts: String) = DeepLink.parse("tipsarr", host, parts.toList())

    @Test fun movie() = assertEquals(DeepLink("tipsarr.example.com", DeepLinkTarget.Media(MediaType.Movie, 603)), parse("Tipsarr.Example.com", "media", "movie", "603"))
    @Test fun show() = assertEquals(DeepLinkTarget.Media(MediaType.Tv, 1396), parse("h", "media", "tv", "1396")?.target)
    @Test fun request() = assertEquals(DeepLinkTarget.Request("abc"), parse("h", "request", "abc")?.target)
    @Test fun issue() = assertEquals(DeepLinkTarget.Issue("7"), parse("h", "issue", "7")?.target)
    @Test fun library() = assertEquals(DeepLinkTarget.Library, parse("h", "library")?.target)
    @Test fun requests() = assertEquals(DeepLinkTarget.Requests, parse("h", "requests")?.target)
    @Test fun badMediaOpensDiscover() = assertEquals(DeepLinkTarget.Discover, parse("h", "media", "book", "1")?.target)
    @Test fun unknownOpensDiscover() = assertEquals(DeepLinkTarget.Discover, parse("h", "whatever")?.target)
    @Test fun otherSchemeIsIgnored() = assertNull(DeepLink.parse("https", "h", listOf("media", "movie", "1")))
    @Test fun noHostIsIgnored() = assertNull(DeepLink.parse("tipsarr", null, emptyList()))
}

class EventParseTest {
    @Test fun progress() {
        val event = EventStream.parse(5, "request.progress", """{"id":"r1","percent":42,"etaSeconds":600}""")
        assertEquals("r1", event.itemId); assertEquals(42, event.percent); assertEquals(600, event.etaSeconds); assertEquals(5, event.id)
    }
    @Test fun brokenDataStillGivesTheType() = assertEquals("suggestions.updated", EventStream.parse(null, "suggestions.updated", "not json").type)
}
