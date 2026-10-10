package com.brebond.tipsarr

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.brebond.tipsarr.core.auth.AccountStore
import com.brebond.tipsarr.core.auth.SessionManager
import kotlinx.coroutines.launch

/** Keeps the [SessionManager] alive across rotation and folding; the launch flow starts once. */
class SessionViewModel(application: Application) : AndroidViewModel(application) {
    val session = SessionManager(AccountStore(application))

    init {
        viewModelScope.launch { session.start() }
    }
}
