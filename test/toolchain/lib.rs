pub fn greet(name: &str) -> String {
    format!("Hello, {}! (from hermetic toolchain)", name)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_greet() {
        assert_eq!(greet("world"), "Hello, world! (from hermetic toolchain)");
    }
}
