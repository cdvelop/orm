# Why Package-Level Schema Variables?

Generated `model_orm.go` files declare schema definitions as package-level variables instead of inlining them inside the `Schema()` method:

```go
// Generated pattern
var _schemaUser = []fmt.Field{
    {Name: "id", Type: fmt.FieldText, DB: &fmt.FieldDB{PK: true}},
    {Name: "name", Type: fmt.FieldText},
}

func (m *User) Schema() []fmt.Field { return _schemaUser }
```

## Why Not Inline?

An inline return would allocate a new slice on every call:

```go
func (m *User) Schema() []fmt.Field {
    return []fmt.Field{
        {Name: "id", Type: fmt.FieldText, DB: &fmt.FieldDB{PK: true}},
        {Name: "name", Type: fmt.FieldText},
    }
}
```

`Schema()` is called on every marshal, unmarshal, and query operation. Inlining would cause the slice to escape to the heap on each invocation, producing one allocation per call.

## Impact

| Approach | Allocations per call | Memory reuse |
|---|---|---|
| Package-level variable | 0 | Single shared instance |
| Inline return | 1 (slice + field structs) | None, GC must collect each |

This matters because:

1. **WASM environments** have constrained memory and a more expensive garbage collector. Avoiding per-call allocations directly reduces GC pressure.
2. **Hot paths** like batch inserts or query result scanning call `Schema()` repeatedly. Zero-allocation access keeps these paths fast.
3. **Consistency with zero-reflect philosophy**: the ORM avoids `reflect` at runtime for performance; heap-allocating the schema on every call would undermine that goal.

## Trade-off: Shared Mutability

The package-level slice is shared across all callers. A caller that mutates the returned slice (e.g. `schema[0].Name = "x"`) would corrupt the schema globally. This is an acceptable trade-off because:

- `Schema()` returns a descriptor, not a working buffer. Mutating it is a caller bug.
- The same pattern is used by Go's standard library (e.g. `reflect.Type` methods return shared descriptors).
- The code is generated and consumed by the ORM internals, not edited or manipulated by users.

## Summary

Package-level schema variables exist for a single reason: **zero allocations per `Schema()` call**. The visual separation from the method is irrelevant since `model_orm.go` is generated code that is never edited by hand.
