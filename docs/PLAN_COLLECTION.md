# ORM Plan: Strongly Typed Collections (No Generics)

## Goal
Provide a seamless, user-friendly way to execute `ReadOne` and `ReadAll` operations returning correctly typed structs/slices without relying on Go 1.18 Generics and without making the developer write boilerplate `factory` and `onRow` closures.

## Implementation Details

1. **Extend `ormgen`:**
   - The CLI tool `cmd/ormgen/main.go` will intercept the struct (e.g., `User`) and, in addition to the `Model` interface, generate highly optimized read accessors.

2. **Generated Output Example:**
   ```go
   // Generated inside <file>_orm.go

   func ReadOneUser(qb *orm.QB) (*User, error) {
       var m User
       // Needs QB instance connected to right table logic
       err := qb.ReadOne()
       // Assume internal mapping logic injects m.Pointers()
       if err != nil {
           return nil, err
       }
       return &m, nil
   }

   func ReadAllUsers(qb *orm.QB) ([]*User, error) {
       var results []*User
       err := qb.ReadAll(
           func() orm.Model { return &User{} },
           func(m orm.Model) { results = append(results, m.(*User)) },
       )
       return results, err
   }
   ```

3. **Justification:**
   - Avoids introducing generic functions that the team disapproves of.
   - Offloads the complex, memory-efficient closure patterns out of the user's business logic, pushing it completely into auto-generated code.
