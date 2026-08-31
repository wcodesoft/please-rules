fn main() {
    let mut buffer = itoa::Buffer::new();
    let printed = buffer.format(12345);
    println!("Formatted via itoa crate: {}", printed);
}
