package structures

import example.structures.DisjointSet as DisjointSetInterface

class DisjointSet : DisjointSetInterface {
    private val parent = mutableMapOf<Int, Int>()

    override fun findRoot(element: Int): Int {
        var root = element
        while (parent.containsKey(root)) {
            root = parent[root]!!
        }
        return root
    }

    override fun unionSets(a: Int, b: Int): Boolean {
        val rootA = findRoot(a)
        val rootB = findRoot(b)
        if (rootA == rootB) return false
        parent[rootA] = rootB
        return true
    }

    override fun reset() {
        parent.clear()
    }
}
