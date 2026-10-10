package com.brebond.tipsarr.features.person

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.brebond.tipsarr.core.api.ApiError
import com.brebond.tipsarr.core.api.PersonDetail
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.person

data class PersonRoute(val id: Int, val name: String)

class PersonModel(val route: PersonRoute, private val api: TipsarrApi) {
    var person by mutableStateOf<PersonDetail?>(null)
        private set
    var error by mutableStateOf<ApiError?>(null)
        private set

    suspend fun load() {
        try {
            person = api.person(route.id)
            error = null
        } catch (e: ApiError) {
            error = e
        }
    }
}
