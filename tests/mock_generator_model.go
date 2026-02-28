package tests

//go:generate ormgen -struct User

type User struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	age       int // unexported
}
