package main

import "fmt"

type Animal interface {
	Run() error
}

type Dog struct{}

func (d *Dog) Run() error {
	fmt.Println("dog run by 4 legs")
	return nil
}

func main() {
	var a = 100

	defer func() {
		var b = a + 1
		fmt.Println(b)
	}()
	var animal Animal
	animal = &Dog{}
	animal.Run()
}
