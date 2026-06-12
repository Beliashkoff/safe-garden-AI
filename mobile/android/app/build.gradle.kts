import java.util.Properties

plugins {
    id("com.android.application")
    id("kotlin-android")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

// Release signing is read from android/key.properties (gitignored). The file is
// absent on dev machines and in CI without the keystore secrets; the release
// build then falls back to the debug keystore so `flutter run --release` and
// sideload-only CI builds still work. With it, the APK is signed by a STABLE
// upload key — required for Play/RuStore uploads.
// See mobile/README.md "Подпись release-сборки".
val keystorePropertiesFile = rootProject.file("key.properties")
val hasReleaseKeystore = keystorePropertiesFile.exists()
val keystoreProperties = Properties().apply {
    if (hasReleaseKeystore) {
        keystorePropertiesFile.inputStream().use { load(it) }
    }
}

// VK ID credentials come from android/local.properties (gitignored):
//   vkid.clientId=<ID приложения VK>
//   vkid.clientSecret=<защищённый ключ VK>
// CI supplies them via -PvkidClientId/-PvkidClientSecret. Empty defaults keep
// the build green until the VK app is registered; VK sign-in simply fails at
// runtime until then.
val localProperties = Properties().apply {
    val f = rootProject.file("local.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
val vkidClientId = (project.findProperty("vkidClientId") as String?)
    ?: localProperties.getProperty("vkid.clientId") ?: ""
val vkidClientSecret = (project.findProperty("vkidClientSecret") as String?)
    ?: localProperties.getProperty("vkid.clientSecret") ?: ""

android {
    namespace = "site.agronomai.app"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = JavaVersion.VERSION_17.toString()
    }

    defaultConfig {
        applicationId = "site.agronomai.app"
        minSdk = 26
        targetSdk = 34
        versionCode = flutter.versionCode
        versionName = flutter.versionName
        // VK ID SDK deep-link return: vk{client_id}://vk.ru.
        addManifestPlaceholders(
            mapOf(
                "VKIDClientID" to vkidClientId,
                "VKIDClientSecret" to vkidClientSecret,
                "VKIDRedirectScheme" to "vk$vkidClientId",
                "VKIDRedirectHost" to "vk.ru",
            )
        )
    }

    signingConfigs {
        if (hasReleaseKeystore) {
            create("release") {
                keyAlias = keystoreProperties["keyAlias"] as String
                keyPassword = keystoreProperties["keyPassword"] as String
                storeFile = file(keystoreProperties["storeFile"] as String)
                storePassword = keystoreProperties["storePassword"] as String
            }
        }
    }

    buildTypes {
        release {
            // Stable upload key when key.properties is present; debug fallback
            // otherwise so the build never breaks without the keystore secrets.
            signingConfig = if (hasReleaseKeystore) {
                signingConfigs.getByName("release")
            } else {
                signingConfigs.getByName("debug")
            }
        }
    }
}

flutter {
    source = "../.."
}
