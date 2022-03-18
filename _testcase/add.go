package main

import "fmt"

func main() {
	for {
		a := 1
		b := 2
		sum := add(a, b)
		fmt.Println(sum)
	}
}

func add(a, b int) (c []int) {
	return []int{a + b}
}
