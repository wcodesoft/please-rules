package test.kotlin.toolchain

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class HermeticTest {

    @Test
    fun testToolchainCheck() {
        assertEquals("hermetic-ok", toolchainCheck())
    }
}
