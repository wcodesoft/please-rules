package test.kotlin.branches

fun classify(n: Int): String {
    return if (n < 0) {
        "neg"
    } else if (n == 0) {
        "zero"
    } else {
        "pos"
    }
}

fun both(a: Boolean, b: Boolean): Boolean {
    return a && b
}

fun unused(): Int {
    return 42
}
