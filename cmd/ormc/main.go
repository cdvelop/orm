//go:build !wasm

package main

import "github.com/tinywasm/orm"

func main() {
	orm.RunOrmcCLI()
}
