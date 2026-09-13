use mylib::add;

#[test]
fn test_add() {
    assert_eq!(add(2, 3), 5);
}

#[test]
fn test_add_negative() {
    assert_eq!(add(-2, -3), -5);
}
