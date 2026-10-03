use mybranches::{both, classify};

#[test]
fn test_classify() {
    assert_eq!(classify(-1), "neg");
    assert_eq!(classify(5), "pos");
}

#[test]
fn test_both() {
    assert!(!both(false, true));
}
