package tests

import (
	"time"

	"github.com/tinywasm/fmt"
)

//go:generate ormc

type User struct {
	ID        int     `db:"pk"`
	FirstName string  `db:"not_null"`
	LastName  string
	Email     string  `db:"unique"`
	Score     float64
	IsActive  bool
	Avatar    []byte
	age       int // unexported
}

type Order struct {
	ID        string `db:"pk"`
	UserID    int    `db:"ref=user:id"`
	Total     float64
}

type BadTimeNoTag struct {
	ID        string `db:"pk"`
	Name      string
	CreatedAt time.Time
}

type ModelWithIgnored struct {
	ID      string   `db:"pk"`
	Name    string
	Tags    []string `db:"-"` // slice: silently ignored
	Friends []User   `db:"-"` // struct slice: silently ignored
	Score   float64
}

type MultiA struct {
	ID   string `db:"pk"`
	Name string
}
func (MultiA) TableName() string { return "multi_a_records" } // manually declared → D5

type MultiB struct {
	ID    string `db:"pk"`
	Value int64
}

type BadAutoInc struct {
	ID string `db:"autoincrement"`
}

type Unsupp struct {
	Ch chan int
}

// NumericTypes covers int32, uint64, float32 mapping and bitmask constraints.
type NumericTypes struct {
	IDNumeric int32   `db:"pk,not_null"` // PK + NotNull → bitmask 5
	CountUint uint64
	RatioF32  float32
}

// RefNoColumn covers db:"ref=table" without a specific column (RefColumn must be "").
type RefNoColumn struct {
	IDRef    string `db:"pk"`
	ParentID int64  `db:"ref=parent"`
}

// PointerReceiver tests that detectTableName handles pointer receivers (*T).
type PointerReceiver struct {
	ID   string `db:"pk"`
	Name string
}

func (*PointerReceiver) TableName() string { return "ptr_table" }

// MockParent / MockChild: relation auto-detection fixture.
type MockParent struct {
	ID   string
	Name string
	Kids []MockChild // no tag — relation auto-detected via MockChild.MockParentID
}

type MockChild struct {
	ID           string `db:"pk"`
	MockParentID string `db:"ref=mock_parent"`
	Value        string
}

type UserForm struct {
	ID       string `db:"pk"`
	Name     string
	Email    string `db:"not_null" form:"email"`
	Password string `form:"password"`
	Bio      string `form:"textarea"`
	Age      int64  `form:"-"`
}

// ormc:formonly
type LoginForm struct {
	Email    string `form:"email"`
	Password string `form:"password"`
}

type Address struct {
	Street string
	City   string
}

func (Address) Schema() []fmt.Field { return nil }
func (Address) Pointers() []any     { return nil }

type UserWithJSON struct {
	ID       string  `db:"pk"           json:"id"`
	Name     string  `json:"name"`
	Email    string  `form:"email"      json:"email"`
	Bio      string  `form:"textarea"   json:"bio,omitempty"`
	HomeAddr Address `json:"home_addr"`
}

type WithPointers struct {
	ID    string  `db:"pk"`
	Count *int    // pointer to primitive -> should be skipped with warning
	Addr  *Address // pointer to struct -> FieldStruct
}
