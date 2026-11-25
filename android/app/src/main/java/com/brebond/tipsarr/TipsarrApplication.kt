package com.brebond.tipsarr

import android.app.Application
import com.brebond.tipsarr.core.push.PushManager

class TipsarrApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        PushManager.init(this)
        com.brebond.tipsarr.core.push.LiveFollow.init(this)
    }
}
