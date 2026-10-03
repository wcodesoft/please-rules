public func classify(_ n: Int) -> String {
    if n < 0 {
        return "neg"
    } else if n == 0 {
        return "zero"
    } else {
        return "pos"
    }
}

public func both(_ a: Bool, _ b: Bool) -> Bool {
    return a && b
}

public func unused() -> Int {
    return 42
}
