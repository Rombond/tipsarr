package com.brebond.tipsarr

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.lifecycle.ViewModel
import com.brebond.tipsarr.core.api.MediaType
import com.brebond.tipsarr.core.api.RequestRecord
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.core.api.requestCounts
import com.brebond.tipsarr.features.detail.MediaDetailModel
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.features.discover.DiscoverModel
import com.brebond.tipsarr.features.discover.MediaListModel
import com.brebond.tipsarr.features.library.LibraryModel
import com.brebond.tipsarr.features.person.PersonModel
import com.brebond.tipsarr.features.person.PersonRoute
import com.brebond.tipsarr.features.requests.RequestsModel
import com.brebond.tipsarr.features.search.GenreRoute
import com.brebond.tipsarr.features.search.SearchModel

/** A screen pushed above a tab's root (the iOS `NavigationPath` entries). */
sealed interface Screen {
    class Title(val model: MediaDetailModel) : Screen
    class Person(val model: PersonModel) : Screen
    class Genre(val route: GenreRoute, val list: MediaListModel) : Screen
    class Request(val record: RequestRecord) : Screen
}

/** Everything the signed-in screens own for one account; survives rotation and folding. */
class MainViewModel(
    val api: TipsarrApi,
    context: Context,
    private val isAdmin: Boolean,
    private val userFolderChoice: Boolean,
) : ViewModel() {
    val discover = DiscoverModel(api)
    val search = SearchModel(api, context)
    val requests = RequestsModel(api, isAdmin)
    val library = LibraryModel(api)

    /** Requests waiting for approval, shown on the tab for admins. */
    var pendingCount by mutableIntStateOf(0)
        private set

    private val stacks = mutableStateMapOf<AppTab, SnapshotStateList<Screen>>()

    fun top(tab: AppTab): Screen? = stacks[tab]?.lastOrNull()

    private fun push(tab: AppTab, screen: Screen) {
        stacks.getOrPut(tab) { mutableStateListOf() }.add(screen)
    }

    fun openTitle(tab: AppTab, route: MediaRoute) = push(tab, Screen.Title(MediaDetailModel(route, api, isAdmin, userFolderChoice)))
    fun openPerson(tab: AppTab, route: PersonRoute) = push(tab, Screen.Person(PersonModel(route, api)))
    fun openGenre(tab: AppTab, type: MediaType, id: Int, name: String) =
        push(tab, Screen.Genre(GenreRoute(type, id, name), MediaListModel(MediaListModel.Source.Genre, api, type to id)))
    fun openRequest(tab: AppTab, record: RequestRecord) = push(tab, Screen.Request(record))

    /** Pops the top screen of the tab; false when only the root is left. */
    fun back(tab: AppTab): Boolean {
        val stack = stacks[tab] ?: return false
        if (stack.isEmpty()) return false
        stack.removeAt(stack.lastIndex)
        return true
    }

    suspend fun refreshPending() {
        if (!isAdmin) return
        runCatching { api.requestCounts() }.getOrNull()?.let { pendingCount = it.pending }
    }
}
