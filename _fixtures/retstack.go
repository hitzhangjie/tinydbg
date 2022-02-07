package main

import (
	"fmt"
	"runtime"
)

func ff() {
	runtime.Breakpoint()
}

func gg() int {
	runtime.Breakpoint()
	return 3
}

func main() {
	ff()
	n := gg() + 1
	fmt.Println(n)
}
