package main

import "fmt"

func main() {
	a := []int{1, 2, 3, 4, 5}
	sum := 0
	for i := 0; i <= len(a); i++ {  // Bug: loop condition should be i < len(a) to avoid index out of range
		sum += a[i]
	}
	fmt.Println("Sum:", sum)
}