package pszlbit.drift.storage

import android.content.Context
import java.util.UUID

/**
 * Anonymous per-tablet token used to reclaim a slot (D-21).
 *
 * It is a random UUID stored in private preferences: no Android ID, serial, or
 * personal data (D-25).
 */
class DeviceTokenStore(context: Context) {
    private val preferences =
        context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    fun token(): String {
        val existing = preferences.getString(KEY_TOKEN, null)
        if (existing != null && existing.isNotEmpty()) return existing
        val created = UUID.randomUUID().toString()
        preferences.edit().putString(KEY_TOKEN, created).apply()
        return created
    }

    fun reset() {
        preferences.edit().remove(KEY_TOKEN).apply()
    }

    private companion object {
        const val PREFS_NAME = "controller"
        const val KEY_TOKEN = "deviceToken"
    }
}
