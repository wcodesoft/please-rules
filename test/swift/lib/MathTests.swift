import Testing
import Math

@Test func testAdd() {
    #expect(Math.add(2, 3) == 5)
    #expect(Math.add(-1, 1) == 0)
}

@Test func testSubtract() {
    #expect(Math.subtract(5, 3) == 2)
}

@Test func testMultiply() {
    #expect(Math.multiply(4, 3) == 12)
}

@Test("Parameterized test", arguments: [
    (2, 2, 4),
    (3, 5, 8),
    (10, -5, 5),
])
func parameterizedAdd(a: Int, b: Int, expected: Int) {
    #expect(Math.add(a, b) == expected)
}
