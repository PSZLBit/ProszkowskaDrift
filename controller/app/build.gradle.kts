plugins {
    alias(libs.plugins.android.application)
}

android {
    namespace = "pszlbit.drift"
    compileSdk {
        version = release(37)
    }

    defaultConfig {
        applicationId = "pszlbit.drift"
        // D-20 / ADR-0002: the four tablets run Android 4.4.2 (API 19).
        minSdk = 19
        targetSdk = 19
        versionCode = 1
        versionName = "1.0"
    }

    buildTypes {
        release {
            optimization {
                enable = true
            }
        }
    }
}

// D-20 / ADR-0002: no Compose and no AndroidX. The controller uses plain
// platform Views so it installs and runs on API 19. Add a dependency only
// after it is proven to support API 19.
dependencies {
    testImplementation(libs.junit)
}
