package com.brebond.tipsarr

import androidx.lifecycle.ViewModel
import com.brebond.tipsarr.core.api.TipsarrApi
import com.brebond.tipsarr.features.discover.DiscoverModel

/** Everything the signed-in screens own for one account; survives rotation and folding. */
class MainViewModel(val api: TipsarrApi) : ViewModel() {
    val discover = DiscoverModel(api)
}
