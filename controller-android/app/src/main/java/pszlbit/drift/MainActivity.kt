package pszlbit.drift

import android.app.Activity
import android.os.Bundle
import android.view.WindowManager

/**
 * Single controller activity.
 *
 * D-20 / ADR-0002: the app must run on the team's Android 4.4.2 (API 19)
 * tablets, so it uses plain platform Views only. No Compose, no AppCompat,
 * no AndroidX. Do not add a library here unless it is proven to support
 * API 19.
 */
class MainActivity : Activity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        // ADR-0002: keep the screen awake while the tablet is a controller.
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
    }
}
