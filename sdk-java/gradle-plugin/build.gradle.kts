plugins {
    `java-gradle-plugin`
    `maven-publish`
    kotlin("jvm") version "1.9.22"
}

group = "io.probex"
version = "1.0.0-SNAPSHOT"

repositories {
    mavenCentral()
    mavenLocal()
}

dependencies {
    implementation("io.probex:probex-sdk:1.0.0-SNAPSHOT")
    implementation("com.fasterxml.jackson.core:jackson-databind:2.18.2")
}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

gradlePlugin {
    plugins {
        create("probex") {
            id = "io.probex"
            implementationClass = "io.probex.gradle.ProbexPlugin"
            displayName = "PROBEX API Testing Plugin"
            description = "Zero-Test API Intelligence Engine — Gradle plugin for automated API testing"
        }
    }
}

publishing {
    publications {
        create<MavenPublication>("pluginMaven") {
            groupId = "io.probex"
            artifactId = "probex-gradle-plugin"
        }
    }
}
