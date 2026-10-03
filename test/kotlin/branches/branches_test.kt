package test.kotlin.branches

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Test

class BranchesTest {

    @Test
    fun testClassify() {
        assertEquals("neg", classify(-1))
        assertEquals("pos", classify(5))
    }

    @Test
    fun testBoth() {
        assertFalse(both(false, true))
    }
}
