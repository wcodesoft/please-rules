package test.kotlin.lib

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class LibTest {

    @Test
    fun testAdd() {
        assertEquals(5, add(2, 3))
    }

    @Test
    fun testMultiply() {
        assertEquals(12, multiply(3, 4))
    }

    @Test
    fun testMavenAnnotations() {
        assertEquals("maven-verified", annotateMessage("maven"))
    }
}
