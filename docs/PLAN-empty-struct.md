# Plan — ormc: empty struct support (zero-field structs)

## Problem

`ormc` currently skips structs with zero fields entirely — no `Schema()`,
no `Pointers()`, no `Validate()` is generated.

This causes a problem when a protocol needs to pass a typed value that carries
no fields (e.g. a JSON-RPC notification with an empty params object `{}`).

**Current workaround**: pass `nil` for `fmt.Fielder` parameters when the struct
has no fields. This works for `SendNotification` but is not type-safe.

## Decision

Two options:

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A | Keep skipping zero-field structs; callers pass `nil` | No change to ormc | Not type-safe; `nil` can be passed where a typed value is expected |
| B | Generate minimal `Schema()`/`Pointers()` for zero-field structs | Type-safe; explicit intent | Small change to ormc generator |

**Chosen: Option B** — generate empty implementations.

## Required Change

For any struct tagged `// ormc:formonly` with zero fields, ormc should generate:

```go
func (s *EmptyStruct) Schema() []fmt.Field  { return nil }
func (s *EmptyStruct) Pointers() []any      { return nil }
```

For structs without `formonly` (full SafeFields), also generate:

```go
func (s *EmptyStruct) Validate(_ byte) error { return nil }
```

These methods make the type satisfy `fmt.Fielder` / `fmt.SafeFields` without
panicking — encoding produces `{}`, decoding is a no-op.

## Implementation

File: `ormc_handler.go` (or wherever struct generation is driven)

Current guard (pseudo-code):
```go
if len(fields) == 0 {
    continue  // skip
}
```

Replace with:
```go
if len(fields) == 0 {
    writeEmptyFielder(w, structName, isFormOnly)
    continue
}
```

Where `writeEmptyFielder` emits the minimal methods shown above.

## Test

Add to `tests/` directory:

```go
// ormc:formonly
type emptyParams struct{}

func TestEmptyStruct_Schema(t *testing.T) {
    var p emptyParams
    if p.Schema() != nil {
        t.Fatal("expected nil schema")
    }
    if p.Pointers() != nil {
        t.Fatal("expected nil pointers")
    }
}

func TestEmptyStruct_EncodesDecode(t *testing.T) {
    var p emptyParams
    var s string
    if err := json.Encode(&p, &s); err != nil {
        t.Fatal(err)
    }
    // encoded value should be valid (empty object or empty string)
    var p2 emptyParams
    _ = json.Decode([]byte(s), &p2) // must not panic
}
```

## Impact

- **tinywasm/mcp**: `toolListChangedParams` and similar zero-field notification
  params can become typed structs instead of `nil`.
- **Breaking change**: none — callers passing `nil` continue to work (interface
  accepts `nil`); newly typed structs are additive.

## Version

Bump patch: `ormc vX.Y.(Z+1)` — no API change, only generator output change.
