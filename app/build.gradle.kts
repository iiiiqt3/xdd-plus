plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.goudong.jd"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.goudong.jd"
        minSdk = 23
        targetSdk = 36
        versionCode = 35
        versionName = "2.8.11"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        vectorDrawables.useSupportLibrary = true

        ndk {
            abiFilters += listOf("armeabi-v7a", "arm64-v8a")
        }

        manifestPlaceholders["JPUSH_PKGNAME"] = applicationId!!
        manifestPlaceholders["JPUSH_APPKEY"] = "173ed5f89f68ae2599e666ae"
        manifestPlaceholders["JPUSH_CHANNEL"] = "default_developer"
    }

    signingConfigs {
        create("release") {
            val keystoreFile = rootProject.file("goudong.jks")
            if (keystoreFile.exists()) {
                storeFile = keystoreFile
            } else {
                storeFile = file("../goudong.jks")
            }
            storePassword = "goudong123"
            keyAlias = "goudong"
            keyPassword = "goudong123"
        }
        create("sharedDebug") {
            val debugFile = rootProject.file("debug.keystore")
            if (debugFile.exists()) {
                storeFile = debugFile
            } else {
                storeFile = file("../debug.keystore")
            }
            storePassword = "android"
            keyAlias = "androiddebugkey"
            keyPassword = "android"
        }
    }

    buildTypes {
        debug {
            signingConfig = signingConfigs.getByName("sharedDebug")
        }
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            signingConfig = signingConfigs.getByName("release")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        viewBinding = true
        buildConfig = true
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("com.google.android.material:material:1.12.0")
    implementation("androidx.constraintlayout:constraintlayout:2.1.4")
    implementation("androidx.fragment:fragment-ktx:1.8.2")
    implementation("androidx.activity:activity-ktx:1.9.1")
    implementation("androidx.swiperefreshlayout:swiperefreshlayout:1.1.0")
    implementation("androidx.viewpager2:viewpager2:1.1.0")
    implementation("androidx.lifecycle:lifecycle-runtime-ktx:2.8.4")
    implementation("androidx.work:work-runtime-ktx:2.9.1")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.8.1")

    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.retrofit2:retrofit:2.11.0")
    implementation("com.squareup.retrofit2:converter-gson:2.11.0")

    implementation("com.google.code.gson:gson:2.11.0")

    implementation("cn.jiguang.sdk:jpush:5.8.0")
}
