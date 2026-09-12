use hermetic_lib::greet;

#[test]
fn test_greet() {
    assert_eq!(greet("world"), "Hello, world! (from hermetic toolchain)");
}
