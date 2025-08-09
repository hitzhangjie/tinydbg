package main

import "fmt"

func main() {
	var sum int
	for i := range 10 {
		// this variable i, escapes to heap i,
		// but why `watch -r i` report watch stack allocated variable?
		sum += i
		fmt.Println("i:", i)
	}
	fmt.Println("sum:", sum)
}
