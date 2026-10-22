import org.gradle.api.tasks.testing.Test

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
        versionName = "0.1.0-spike"
    }

    buildTypes {
        release {
            optimization {
                enable = true
            }
        }
    }

    lint {
        // D-20 / ADR-0002: the APK is sideloaded onto the four Android 4.4.2
        // tablets and is never published on Google Play, so the Play target-SDK
        // policy does not apply. targetSdk stays 19 on purpose.
        disable += "ExpiredTargetSdkVersion"
    }
}

// D-20 / ADR-0002: no Compose and no AndroidX. The controller uses plain
// platform Views so it installs and runs on API 19.
//
// OkHttp 3.12 is the explicitly allowed WebSocket client (ADR-0002); OkHttp 4
// requires API 21 and must not be added. org.json is the platform JSON parser
// in the app; the Maven artifact only backs JVM unit tests, where the android
// stub throws.
dependencies {
    implementation(libs.okhttp)
    testImplementation(libs.junit)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.json)
}

// Opt-in live check against a running Go server:
//   ./gradlew :app:testDebugUnitTest --tests "*LiveServerTest" \
//     -Dpsz.liveServerUrl=ws://127.0.0.1:8080/ws/controller
tasks.withType<Test>().configureEach {
    systemProperty("psz.liveServerUrl", System.getProperty("psz.liveServerUrl", ""))
}
