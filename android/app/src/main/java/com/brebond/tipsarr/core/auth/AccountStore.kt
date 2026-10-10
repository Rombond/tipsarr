package com.brebond.tipsarr.core.auth

import android.content.Context
import com.brebond.tipsarr.core.api.ApiClient
import kotlinx.serialization.builtins.ListSerializer

/** Accounts in the encrypted store, the active one in plain preferences. */
class AccountStore(context: Context) {
    private val secure = SecureStore(context, "accounts")
    private val prefs = context.getSharedPreferences("session", Context.MODE_PRIVATE)

    var accounts: List<Account> = load().sortedByDescending { it.lastUsed }
        private set
    var activeId: String? = prefs.getString(ACTIVE, null)?.takeIf { saved -> accounts.any { it.id == saved } } ?: accounts.firstOrNull()?.id
        private set

    val active: Account? get() = accounts.firstOrNull { it.id == activeId }

    /** Adds or replaces an account and makes it the active one. */
    fun add(account: Account) {
        val stamped = account.copy(lastUsed = System.currentTimeMillis())
        accounts = listOf(stamped) + accounts.filter { it.id != stamped.id }
        save()
        setActive(stamped.id)
    }

    fun setActive(id: String?) {
        activeId = id
        prefs.edit().putString(ACTIVE, id).apply()
    }

    fun remove(account: Account) {
        accounts = accounts.filter { it.id != account.id }
        save()
        if (activeId == account.id) setActive(accounts.firstOrNull()?.id)
    }

    private fun load(): List<Account> = secure.read(KEY)?.let {
        runCatching { ApiClient.json.decodeFromString(ListSerializer(Account.serializer()), it) }.getOrNull()
    } ?: emptyList()

    private fun save() = secure.write(KEY, ApiClient.json.encodeToString(ListSerializer(Account.serializer()), accounts))

    private companion object {
        const val KEY = "accounts"
        const val ACTIVE = "activeAccountId"
    }
}
