package test.kotlin.lib

import org.jetbrains.annotations.NotNull

fun add(a: Int, b: Int): Int {
    return a + b
}

fun multiply(a: Int, b: Int): Int {
    return a * b
}

@NotNull
fun annotateMessage(@NotNull prefix: String): String {
    return "$prefix-verified"
}
