allprojects {
    repositories {
        google()
        mavenCentral()
        // VK ID SDK artifacts (vkid_flutter_sdk pulls the native Android SDK).
        maven(url = "https://artifactory-external.vkpartner.ru/artifactory/vkid-sdk-android/")
    }
}

val newBuildDir: Directory =
    rootProject.layout.buildDirectory
        .dir("../../build")
        .get()
rootProject.layout.buildDirectory.value(newBuildDir)

subprojects {
    val newSubprojectBuildDir: Directory = newBuildDir.dir(project.name)
    project.layout.buildDirectory.value(newSubprojectBuildDir)
}
subprojects {
    project.evaluationDependsOn(":app")
}

// vkid_flutter_sdk pulls com.vk.id:tracking-tracer (AppTracer Lite), which
// references ru.ok.tracer.* classes that are not on the classpath and fail R8
// minification in release builds. VK ID ships a no-op replacement; substituting
// it (official guidance) drops the tracker entirely and keeps OK analytics out.
subprojects {
    configurations.all {
        resolutionStrategy.dependencySubstitution {
            substitute(module("com.vk.id:tracking-tracer:2.6.0"))
                .using(module("com.vk.id:tracking-noop:2.6.0"))
        }
    }
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
