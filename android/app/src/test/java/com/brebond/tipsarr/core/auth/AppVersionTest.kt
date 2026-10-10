package com.brebond.tipsarr.core.auth

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AppVersionTest {
    @Test fun emptyMinimumAcceptsAnyVersion() = assertFalse(AppVersion.isOlder("0.0.1", ""))
    @Test fun olderMinorIsOlder() = assertTrue(AppVersion.isOlder("0.1.0", "0.2.0"))
    @Test fun sameVersionIsNotOlder() = assertFalse(AppVersion.isOlder("1.2.3", "1.2.3"))
    @Test fun missingPartsCountAsZero() = assertFalse(AppVersion.isOlder("1.2", "1.2.0"))
    @Test fun numbersCompareNumericallyNotAsText() = assertFalse(AppVersion.isOlder("1.10.0", "1.9.0"))
    @Test fun suffixesAreIgnored() = assertFalse(AppVersion.isOlder("1.2.0-beta", "1.2.0"))
}

class ServerAddressTest {
    @Test fun addsHttps() = assertEquals("https://tipsarr.example.com", ServerAddress.normalize("tipsarr.example.com"))
    @Test fun dropsTrailingSlashQueryAndFragment() = assertEquals("https://a.example.com/tipsarr", ServerAddress.normalize(" https://a.example.com/tipsarr/?x=1#y "))
    @Test fun keepsHttpAndPort() = assertEquals("http://192.168.1.10:8080", ServerAddress.normalize("http://192.168.1.10:8080/"))
    @Test fun rejectsOtherSchemes() = assertNull(ServerAddress.normalize("ftp://example.com"))
    @Test fun rejectsEmpty() = assertNull(ServerAddress.normalize("   "))
    @Test fun rejectsMissingHost() = assertNull(ServerAddress.normalize("https://"))
}
