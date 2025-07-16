package main

func main() {
	var sum int
	for i := range 10 {
		// `watch i`, will report error, because here i is stack allocated.
		// go runtime may resize the stack.
		sum += i
	}
	_ = sum
}
