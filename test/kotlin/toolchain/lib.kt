package test.kotlin.toolchain

fun toolchainCheck(): String {
    val vendor = System.getProperty("java.vendor") ?: ""
    val version = System.getProperty("java.version") ?: ""
    println("Runtime Java Vendor: $vendor, Version: $version")
    if (!version.startsWith("21.")) {
        throw AssertionError("Expected Java 21 from hermetic toolchain, but got: $version (Vendor: $vendor)")
    }
    return "hermetic-ok"
}
