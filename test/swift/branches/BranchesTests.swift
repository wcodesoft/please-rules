import Testing
import Branches

@Test func testClassify() {
    #expect(classify(-1) == "neg")
    #expect(classify(5) == "pos")
}

@Test func testBoth() {
    #expect(both(false, true) == false)
}
