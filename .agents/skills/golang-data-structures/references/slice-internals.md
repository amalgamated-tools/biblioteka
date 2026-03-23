# Slice Internals

## Memory Layout

Conceptually, a slice value is a small header containing three machine words (24 bytes on 64-bit architectures):

- **Pointer** — points to backing array (heap-allocated)
- **Length** — number of elements in use
- **Capacity** — allocated size of backing array

Assigning or passing a slice copies this header value, not the backing array. Both the original and copy point to the same underlying data—mutations are visible to both.

## Capacity Growth

On current Go releases, when `append` exceeds capacity, the runtime typically grows the backing array roughly as follows (an implementation detail that may change between versions and architectures):

- For small capacities (around `oldCap < 256` on 64-bit architectures): roughly double capacity
- For larger capacities: grow by about 25% (`oldCap + (oldCap + 3*256) / 4`)

Code must not rely on these exact thresholds or formulas; they describe current runtime behavior for intuition and performance reasoning only.
### Growth Cost

Each growth is O(n) — the entire array is copied to a new location. For a slice growing from 0 to N elements one at a time, the amortized cost per append is O(1), but the total copies are roughly 2N. **Preallocation eliminates all intermediate copies:**

```go
// Known size — direct indexing
out := make([]Result, len(input))
for i, v := range input {
    out[i] = transform(v)
}

// Approximate size
out := make([]Result, 0, len(input)*2)
for _, v := range input {
    out = append(out, transform(v))
}
```

## `slices` Package (Go 1.21+)

| Category | Key Functions |
| --- | --- |
| **Sort** | `Sort`, `SortFunc`, `SortStableFunc`, `IsSorted` |
| **Search** | `BinarySearch`, `BinarySearchFunc`, `Contains`, `Index`, `IndexFunc` |
| **Mutate** | `Insert`, `Delete`, `Replace`, `Compact`, `Reverse`, `Grow`, `Clip` |
| **Create** | `Concat` (1.22+), `Repeat` (1.23+), `Chunk` (1.23+) |
| **Compare** | `Clone`, `Equal`, `EqualFunc`, `Compare`, `DeleteFunc` |

## `copy()` vs `append()` vs `slices.Clone()`

| Operation             | Use When                         |
| --------------------- | -------------------------------- |
| `copy(dst, src)`      | Copying into pre-allocated slice |
| `append(dst, src...)` | Appending to a slice             |
| `slices.Clone(s)`     | Creating independent copy        |
| `s[:len(s):len(s)]`   | Preventing append aliasing       |
