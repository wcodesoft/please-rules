import Base

public struct Mid {
    public static func quad(_ a: Int) -> Int {
        return Base.twice(Base.twice(a))
    }
}
