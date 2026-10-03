pub fn classify(n: i32) -> &'static str {
    if n < 0 {
        "neg"
    } else if n == 0 {
        "zero"
    } else {
        "pos"
    }
}

pub fn both(a: bool, b: bool) -> bool {
    a && b
}

pub fn unused() -> i32 {
    42
}
