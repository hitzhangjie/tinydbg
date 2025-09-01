package main

import (
	"fmt"
	"os"
	"runtime"
)

// #cgo LDFLAGS: -L. -ladd
// #include <stdint.h>
// int64_t add(int64_t a, int64_t b);
// int64_t bad_add(int64_t a, int64_t b);
import "C"

func main() {
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("GOTRACEBACK: %s\n", os.Getenv("GOTRACEBACK"))

	// First test the normal add function
	result := C.add(C.int64_t(5), C.int64_t(3))
	fmt.Printf("Normal add result: %d\n", result)

	// Then test the bad_add function which should trigger a core dump
	fmt.Println("Testing bad_add function...")
	result = C.bad_add(C.int64_t(5), C.int64_t(3))
	fmt.Printf("Normal add result: %d\n", result)
}
