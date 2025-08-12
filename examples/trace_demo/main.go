package main

import "time"

type Person struct {
	Name string
	Age  int
}

func main() {
	people := []Person{
		{Name: "Alice", Age: 25},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 35},
	}

	for _, p := range people {
		time.Sleep(time.Second) // 添加一些延迟以便于调试
		processPerson(p)        // tinydbg> trace main.go:19
	}
}

//go:noinline
func processPerson(p Person) { // tinydbg> trace main.go:24
	_ = p
}
