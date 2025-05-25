package main

import (
	"fmt"
	"time"
)

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

	for i, p := range people {
		fmt.Printf("Processing person %d: %s\n", i, p.Name)
		time.Sleep(time.Second) // 添加一些延迟以便于调试
		processPerson(p)
	}
}

func processPerson(p Person) {
	fmt.Printf("Name: %s, Age: %d\n", p.Name, p.Age)
}
