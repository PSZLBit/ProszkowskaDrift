package pszlbit.drift.storage

import android.content.Context

/**
 * Server address configured once by the team (ADR-0002, FR-C07).
 *
 * Empty until someone sets it through the hidden setup dialog.
 */
class ServerSettings(context: Context) {
    private val preferences =
        context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    var serverUrl: String
        get() = preferences.getString(KEY_SERVER_URL, "") ?: ""
        set(value) {
            preferences.edit().putString(KEY_SERVER_URL, value.trim()).apply()
        }

    private companion object {
        const val PREFS_NAME = "controller"
        const val KEY_SERVER_URL = "serverUrl"
    }
}
