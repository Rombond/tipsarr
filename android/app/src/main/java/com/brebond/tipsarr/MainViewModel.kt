package com.brebond.tipsarr

import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.compose.runtime.mutableStateListOf
import androidx.lifecycle.ViewModel
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.features.detail.MediaDetailModel
import com.brebond.tipsarr.features.detail.MediaRoute
import com.brebond.tipsarr.features.discover.DiscoverModel

/** Everything the signed-in screens own for one account; survives rotation and folding. */
class MainViewModel(val api: TipsarrApi, private val isAdmin: Boolean, private val userFolderChoice: Boolean) : ViewModel() {
    val discover = DiscoverModel(api)

    /** One stack of opened titles per tab (the iOS `NavigationPath` of each tab); the root screen is not in it. */
    private val stacks = mutableStateMapOf<AppTab, SnapshotStateList<MediaDetailModel>>()

    fun top(tab: AppTab): MediaDetailModel? = stacks[tab]?.lastOrNull()

    fun open(tab: AppTab, route: MediaRoute) {
        stacks.getOrPut(tab) { mutableStateListOf() }.add(MediaDetailModel(route, api, isAdmin, userFolderChoice))
    }

    /** Pops the top screen of the tab; false when only the root is left. */
    fun back(tab: AppTab): Boolean {
        val stack = stacks[tab] ?: return false
        if (stack.isEmpty()) return false
        stack.removeAt(stack.lastIndex)
        return true
    }
}
